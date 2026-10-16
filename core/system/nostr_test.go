package system

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// nostrStubRelay is a stand-in for relay.primal.net that speaks the subset of
// NIP-01 the client uses. It reads the client's first frame itself — a REQ is
// the only thing the client ever opens with — records it, and hands the
// connection to serve.
//
// Its frame reader is written separately from the client's, so the client's
// masking is verified against the wire format rather than against itself.
type nostrStubRelay struct {
	url   string
	serve func(conn net.Conn, br *bufio.Reader, req string) error

	ln    net.Listener
	wg    sync.WaitGroup
	mu    sync.Mutex
	errs  []error
	conns int
	reqs  []string
}

func newNostrStubRelay(t *testing.T, serve func(conn net.Conn, br *bufio.Reader, req string) error) *nostrStubRelay {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	relay := &nostrStubRelay{url: "ws://" + ln.Addr().String(), serve: serve, ln: ln}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			relay.wg.Add(1)
			go func() {
				defer relay.wg.Done()
				defer conn.Close()
				relay.handle(conn)
			}()
		}
	}()

	t.Cleanup(func() {
		ln.Close()
		relay.wg.Wait()

		relay.mu.Lock()
		defer relay.mu.Unlock()
		for _, err := range relay.errs {
			t.Errorf("stub relay: %v", err)
		}
	})

	return relay
}

func (r *nostrStubRelay) handle(conn net.Conn) {
	// The client always sets its own deadline; this is the backstop for a test
	// that got the client side wrong.
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	br, err := r.handshake(conn)
	if err != nil {
		r.fail(err)
		return
	}

	opcode, payload, err := readClientFrame(br)
	if err != nil {
		r.fail(err)
		return
	}
	if opcode != wsOpText {
		r.fail(fmt.Errorf("client's opening frame has opcode %#x, want text", opcode))
		return
	}

	r.mu.Lock()
	r.conns++
	r.reqs = append(r.reqs, string(payload))
	r.mu.Unlock()

	if err := r.serve(conn, br, string(payload)); err != nil {
		r.fail(err)
	}
}

// handshake completes the upgrade and returns the buffered reader the frames
// must be read from — a fresh reader would lose whatever the client already
// pipelined behind its request.
func (r *nostrStubRelay) handshake(conn net.Conn) (*bufio.Reader, error) {
	br := bufio.NewReader(conn)

	key := ""
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if name, value, ok := strings.Cut(line, ":"); ok {
			if strings.EqualFold(strings.TrimSpace(name), "Sec-WebSocket-Key") {
				key = strings.TrimSpace(value)
			}
		}
	}

	if _, err := fmt.Fprintf(conn,
		"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		wsAcceptKey(key)); err != nil {
		return nil, err
	}
	return br, nil
}

func (r *nostrStubRelay) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs = append(r.errs, err)
}

func (r *nostrStubRelay) connections() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conns
}

func (r *nostrStubRelay) requests() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.reqs...)
}

// readClientFrame is a second implementation of client-frame parsing, used only
// by the stub. A frame it fails to decode is a frame the real client would have
// sent wrong.
func readClientFrame(br *bufio.Reader) (byte, []byte, error) {
	var head [2]byte
	if _, err := io.ReadFull(br, head[:]); err != nil {
		return 0, nil, err
	}

	opcode := head[0] & 0x0f
	if head[1]&0x80 == 0 {
		return 0, nil, errors.New("client frame is unmasked, which RFC 6455 §5.3 forbids")
	}

	length := uint64(head[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if length > wsMaxFrame {
		return 0, nil, fmt.Errorf("client frame of %d bytes", length)
	}

	var mask [4]byte
	if _, err := io.ReadFull(br, mask[:]); err != nil {
		return 0, nil, err
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(br, payload); err != nil {
		return 0, nil, err
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return opcode, payload, nil
}

// writeServerFrame sends one final, unmasked frame, which is what a server does.
func writeServerFrame(w io.Writer, payload string) error {
	_, err := w.Write(rawFrame(true, wsOpText, nil, []byte(payload)))
	return err
}

// The same externally-verified vectors core/nip19 uses. They are repeated rather
// than shared because the two are separate packages, and a test vector is cheap
// to state twice and annoying to export.
const (
	jackNpub = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m"
	jackHex  = "82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2"

	// Well-formed bech32 under a prefix that does not name a profile, and one
	// whose checksum does not verify.
	noteKey32          = "note1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q7k28gn"
	nsecKey32          = "nsec1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qr2zmhw"
	npubBrokenChecksum = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63q"
	npub31Bytes        = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmucs89urk"
)

// ─── Relay fan-out ────────────────────────────────────────────────────────────
//
// Single-relay coverage is the reason the query fans out at all: a relay that
// simply does not hold an account's kind-0 answers with a clean EOSE, which is
// indistinguishable from an account with no profile unless another relay is
// asked. These tests pin that distinction.

// TestFetchNostrProfileFallsBackToAnotherRelay is the whole reason the fan-out
// exists: the first relay answers cleanly and holds nothing, the second has the
// event.
func TestFetchNostrProfileFallsBackToAnotherRelay(t *testing.T) {
	isolatedNostrCache(t)

	empty := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
		return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
	})

	full := newNostrStubRelay(t, serveKind0(t, `{"name":"jack"}`))

	useStubRelay(t, empty.url, full.url)

	profile, err := GetNostrProfileCached(jackNpub, false)
	if err != nil {
		t.Fatalf("GetNostrProfileCached: %v", err)
	}
	if profile.Name != "jack" {
		t.Errorf("name = %q, want %q", profile.Name, "jack")
	}
	if got := empty.connections(); got != 1 {
		t.Errorf("the relay that held nothing saw %d connections, want 1", got)
	}
}

// TestFetchNostrProfileSurvivesAnUnreachableRelay covers the other reason to ask
// more than one: a relay that refuses outright must not end the query.
func TestFetchNostrProfileSurvivesAnUnreachableRelay(t *testing.T) {
	isolatedNostrCache(t)

	full := newNostrStubRelay(t, serveKind0(t, `{"name":"jack"}`))

	// Port 1 refuses connections immediately.
	useStubRelay(t, "ws://127.0.0.1:1", full.url)

	profile, err := GetNostrProfileCached(jackNpub, false)
	if err != nil {
		t.Fatalf("GetNostrProfileCached: %v", err)
	}
	if profile.Name != "jack" {
		t.Errorf("name = %q, want %q", profile.Name, "jack")
	}
}

// TestFetchNostrProfileOnlyReportsAbsenceWhenEveryRelayAnswered is the status
// mapping the handler depends on. If a relay was unreachable the absence is
// unproven, so it must surface as a failure rather than as "no profile".
func TestFetchNostrProfileOnlyReportsAbsenceWhenEveryRelayAnswered(t *testing.T) {
	t.Run("every relay held nothing", func(t *testing.T) {
		isolatedNostrCache(t)

		a := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
			return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
		})
		b := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
			return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
		})

		useStubRelay(t, a.url, b.url)

		if _, err := GetNostrProfileCached(jackNpub, false); !errors.Is(err, ErrNostrNoProfile) {
			t.Fatalf("error = %v, want ErrNostrNoProfile", err)
		}
	})

	t.Run("one relay was unreachable", func(t *testing.T) {
		isolatedNostrCache(t)

		empty := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
			return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
		})

		useStubRelay(t, empty.url, "ws://127.0.0.1:1")

		_, err := GetNostrProfileCached(jackNpub, false)
		if err == nil {
			t.Fatal("no error, want a failure: one relay never answered")
		}
		if errors.Is(err, ErrNostrNoProfile) {
			t.Errorf("error is ErrNostrNoProfile, but one relay was unreachable so the absence is unproven: %v", err)
		}
	})
}

// TestFetchNostrProfileWithNoRelaysConfigured guards the degenerate set. It is
// reachable: YANTR_NOSTR_RELAYS is user input.
func TestFetchNostrProfileWithNoRelaysConfigured(t *testing.T) {
	isolatedNostrCache(t)
	useStubRelay(t)

	if _, err := GetNostrProfileCached(jackNpub, false); err == nil {
		t.Error("no error with an empty relay set, want a configuration failure")
	}
}

// eventFrame builds the EVENT message a relay sends for a kind-0 match.
func eventFrame(t *testing.T, createdAt int64, content string) string {
	t.Helper()
	return fmt.Sprintf(`["EVENT","yantr-profile",{"id":"00","pubkey":%q,"created_at":%d,"kind":0,"tags":[],"content":%q}]`,
		jackHex, createdAt, content)
}

// useStubRelay points the package at the stub and restores the real relay set
// after. Pass several to describe a relay set where every relay behaves the same.
func useStubRelay(t *testing.T, urls ...string) {
	t.Helper()
	previous := nostrRelayURLs
	nostrRelayURLs = urls
	t.Cleanup(func() { nostrRelayURLs = previous })
}

// isolatedNostrCache empties the package cache so a test starts from nothing.
// Tests in this package do not run in parallel, so this is safe.
func isolatedNostrCache(t *testing.T) {
	t.Helper()
	nostrCacheMu.Lock()
	nostrCache = map[string]nostrEntry{}
	nostrCacheMu.Unlock()
}

// serveKind0 is the common case: answer the query with one profile and EOSE.
func serveKind0(t *testing.T, profileJSON string) func(net.Conn, *bufio.Reader, string) error {
	t.Helper()
	return func(conn net.Conn, _ *bufio.Reader, _ string) error {
		if err := writeServerFrame(conn, eventFrame(t, 1_700_000_000, profileJSON)); err != nil {
			return err
		}
		return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
	}
}

func TestGetNostrProfileCachedReadsKind0(t *testing.T) {
	isolatedNostrCache(t)

	const content = `{"name":"jack","display_name":"Jack","about":"no state is the best state","picture":"https://example.test/j.jpg","nip05":"jack@primal.net","lud16":"jack@primal.net","website":"https://example.test"}`
	relay := newNostrStubRelay(t, serveKind0(t, content))
	useStubRelay(t, relay.url)

	profile, err := GetNostrProfileCached(jackNpub, false)
	if err != nil {
		t.Fatalf("GetNostrProfileCached: %v", err)
	}

	for _, tc := range []struct{ field, got, want string }{
		{"pubkey", profile.Pubkey, jackHex},
		{"key", profile.Key, jackNpub},
		{"name", profile.Name, "jack"},
		{"displayName", profile.DisplayName, "Jack"},
		{"about", profile.About, "no state is the best state"},
		{"picture", profile.Picture, "https://example.test/j.jpg"},
		{"nip05", profile.NIP05, "jack@primal.net"},
		{"lud16", profile.LUD16, "jack@primal.net"},
		{"website", profile.Website, "https://example.test"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.field, tc.got, tc.want)
		}
	}
	if profile.CreatedAt != 1_700_000_000 {
		t.Errorf("createdAt = %d, want %d", profile.CreatedAt, 1_700_000_000)
	}
	if profile.FetchedAt == "" {
		t.Error("fetchedAt is empty")
	}
}

func TestGetNostrProfileCachedFiltersByTheDecodedPubkey(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, serveKind0(t, `{"name":"jack"}`))
	useStubRelay(t, relay.url)

	if _, err := GetNostrProfileCached(jackNpub, false); err != nil {
		t.Fatalf("GetNostrProfileCached: %v", err)
	}

	requests := relay.requests()
	if len(requests) != 1 {
		t.Fatalf("relay saw %d requests, want 1", len(requests))
	}

	// Decoded and re-encoded rather than compared as text: what matters is the
	// hex the filter carries, not how the catalog spelled the key.
	var msg []json.RawMessage
	if err := json.Unmarshal([]byte(requests[0]), &msg); err != nil {
		t.Fatalf("request is not JSON: %v", err)
	}
	if len(msg) != 3 {
		t.Fatalf("request has %d elements, want 3", len(msg))
	}

	var verb, sub string
	if err := json.Unmarshal(msg[0], &verb); err != nil || verb != "REQ" {
		t.Errorf("request verb = %q (err %v), want REQ", verb, err)
	}
	if err := json.Unmarshal(msg[1], &sub); err != nil || sub == "" {
		t.Errorf("request subscription id = %q (err %v), want a non-empty id", sub, err)
	}

	var filter struct {
		Kinds   []int    `json:"kinds"`
		Authors []string `json:"authors"`
		Limit   int      `json:"limit"`
	}
	if err := json.Unmarshal(msg[2], &filter); err != nil {
		t.Fatalf("filter is not JSON: %v", err)
	}
	if len(filter.Kinds) != 1 || filter.Kinds[0] != 0 {
		t.Errorf("filter kinds = %v, want [0]", filter.Kinds)
	}
	if len(filter.Authors) != 1 || filter.Authors[0] != jackHex {
		t.Errorf("filter authors = %v, want [%s]", filter.Authors, jackHex)
	}
	if filter.Limit < 1 {
		t.Errorf("filter limit = %d, want a positive limit", filter.Limit)
	}
}

// TestGetNostrProfileCachedTakesNewestEvent covers the reason the filter asks for
// more than one event: kind-0 events replace rather than chain, and a relay is
// free to answer `limit: 1` with any stored event rather than the newest.
func TestGetNostrProfileCachedTakesNewestEvent(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
		// Newest first, then a stale one, so a client that keeps the last event
		// it is handed gets the wrong name.
		if err := writeServerFrame(conn, eventFrame(t, 2_000_000_000, `{"name":"current"}`)); err != nil {
			return err
		}
		if err := writeServerFrame(conn, eventFrame(t, 1_000_000_000, `{"name":"stale"}`)); err != nil {
			return err
		}
		return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
	})
	useStubRelay(t, relay.url)

	profile, err := GetNostrProfileCached(jackNpub, false)
	if err != nil {
		t.Fatalf("GetNostrProfileCached: %v", err)
	}
	if profile.Name != "current" {
		t.Errorf("name = %q, want %q", profile.Name, "current")
	}
}

// TestGetNostrProfileCachedIgnoresUnusableEvents covers a relay answering with
// more than was asked for: another subscription, another kind, or a frame that
// is not a NIP-01 message at all.
func TestGetNostrProfileCachedIgnoresUnusableEvents(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
		for _, frame := range []string{
			`["EVENT","some-other-sub",{"kind":0,"created_at":3000000000,"content":"{\"name\":\"wrong sub\"}"}]`,
			`["EVENT","yantr-profile",{"kind":1,"created_at":3000000000,"content":"{\"name\":\"wrong kind\"}"}]`,
			`not json at all`,
			`["EOSE"`,
			`["NOTICE","slow down"]`,
		} {
			if err := writeServerFrame(conn, frame); err != nil {
				return err
			}
		}
		if err := writeServerFrame(conn, eventFrame(t, 1_000_000_000, `{"name":"real"}`)); err != nil {
			return err
		}
		return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
	})
	useStubRelay(t, relay.url)

	profile, err := GetNostrProfileCached(jackNpub, false)
	if err != nil {
		t.Fatalf("GetNostrProfileCached: %v", err)
	}
	if profile.Name != "real" {
		t.Errorf("name = %q, want %q", profile.Name, "real")
	}
}

// TestGetNostrProfileCachedCachesHits is the point of the seven-day cache: a
// page view must not become a relay query.
func TestGetNostrProfileCachedCachesHits(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, serveKind0(t, `{"name":"jack"}`))
	useStubRelay(t, relay.url)

	for i := 0; i < 3; i++ {
		if _, err := GetNostrProfileCached(jackNpub, false); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}

	if got := relay.connections(); got != 1 {
		t.Errorf("relay saw %d connections for three reads, want 1", got)
	}
}

func TestGetNostrProfileCachedForceRefreshBypassesTheCache(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, serveKind0(t, `{"name":"jack"}`))
	useStubRelay(t, relay.url)

	if _, err := GetNostrProfileCached(jackNpub, false); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := GetNostrProfileCached(jackNpub, true); err != nil {
		t.Fatalf("forced call: %v", err)
	}

	if got := relay.connections(); got != 2 {
		t.Errorf("relay saw %d connections, want 2", got)
	}
}

// TestGetNostrProfileCachedRemembersNoProfile covers the negative cache. A key
// with no metadata is a real answer, but it should not be re-asked on every page
// view, and it should not be remembered for a week either.
func TestGetNostrProfileCachedRemembersNoProfile(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
		return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
	})
	useStubRelay(t, relay.url)

	for i := 0; i < 2; i++ {
		_, err := GetNostrProfileCached(jackNpub, false)
		if !errors.Is(err, ErrNostrNoProfile) {
			t.Fatalf("call %d returned %v, want ErrNostrNoProfile", i+1, err)
		}
	}

	if got := relay.connections(); got != 1 {
		t.Errorf("relay saw %d connections for two reads, want 1", got)
	}
}

// TestGetNostrProfileCachedSurfacesRelayNotice covers a relay that declines to
// answer and says why, which is more useful in the log than "no profile".
func TestGetNostrProfileCachedSurfacesRelayNotice(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
		if err := writeServerFrame(conn, `["NOTICE","rate limited: too many requests"]`); err != nil {
			return err
		}
		return writeServerFrame(conn, `["EOSE","yantr-profile"]`)
	})
	useStubRelay(t, relay.url)

	_, err := GetNostrProfileCached(jackNpub, false)
	if err == nil {
		t.Fatal("GetNostrProfileCached returned no error, want the relay's notice")
	}
	if errors.Is(err, ErrNostrNoProfile) {
		t.Errorf("error is ErrNostrNoProfile; a relay that explained itself is not the same as one with no profile: %v", err)
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("error %q does not carry the relay's notice", err)
	}
}

// TestGetNostrProfileCachedKeepsLastGoodProfile is the stale-on-error rule: a
// refresh that fails must not blank a page that worked a minute ago.
func TestGetNostrProfileCachedKeepsLastGoodProfile(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, serveKind0(t, `{"name":"jack"}`))
	useStubRelay(t, relay.url)

	first, err := GetNostrProfileCached(jackNpub, false)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	// A port nothing is listening on, so the dial fails immediately.
	nostrRelayURLs = []string{"ws://127.0.0.1:1"}

	again, err := GetNostrProfileCached(jackNpub, true)
	if err != nil {
		t.Fatalf("refresh returned %v, want the cached profile", err)
	}
	if again == nil || again.Name != first.Name {
		t.Errorf("refresh returned %+v, want the cached profile named %q", again, first.Name)
	}
}

// TestGetNostrProfileCachedRejectsNonProfileKeys keeps the cache bounded: a
// string that is not an npub or nprofile must never reach the relay, so it can
// never hold a slot.
func TestGetNostrProfileCachedRejectsNonProfileKeys(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, func(conn net.Conn, _ *bufio.Reader, _ string) error {
		return errors.New("the relay should not have been contacted")
	})
	useStubRelay(t, relay.url)

	for _, key := range []string{"", "   ", "not-a-key", nsecKey32, noteKey32, npubBrokenChecksum, npub31Bytes} {
		if _, err := GetNostrProfileCached(key, false); err == nil {
			t.Errorf("GetNostrProfileCached(%q) returned no error", key)
		}
	}

	if got := relay.connections(); got != 0 {
		t.Errorf("relay saw %d connections for unusable keys, want 0", got)
	}
}

// TestGetNostrProfileCachedTimesOut pins the bound on a relay that completes the
// handshake and then says nothing, which is the failure a raw socket invites.
func TestGetNostrProfileCachedTimesOut(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, func(conn net.Conn, br *bufio.Reader, _ string) error {
		// Deliberately silent: the client's deadline has to end this. Waiting on
		// the connection rather than sleeping means the test finishes as soon as
		// the client gives up.
		var buf [1]byte
		if _, err := br.Read(buf[:]); err != nil {
			return nil
		}
		return nil
	})
	useStubRelay(t, relay.url)

	previous := nostrQueryTimeout
	nostrQueryTimeout = 200 * time.Millisecond
	t.Cleanup(func() { nostrQueryTimeout = previous })

	start := time.Now()
	if _, err := GetNostrProfileCached(jackNpub, false); err == nil {
		t.Fatal("GetNostrProfileCached returned no error against a silent relay")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("query took %v, want it bounded by the %v timeout", elapsed, nostrQueryTimeout)
	}
}

// TestGetNostrProfileCachedReturnsEmptyProfileForUnparseableContent covers the
// kind-0 events written by older clients, whose content is not JSON. The key
// itself is still worth showing, so this is a profile rather than an error.
func TestGetNostrProfileCachedReturnsEmptyProfileForUnparseableContent(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, serveKind0(t, "my name is jack"))
	useStubRelay(t, relay.url)

	profile, err := GetNostrProfileCached(jackNpub, false)
	if err != nil {
		t.Fatalf("GetNostrProfileCached: %v", err)
	}
	if profile.Pubkey != jackHex || profile.Key != jackNpub {
		t.Errorf("profile lost its identity: %+v", profile)
	}
	if profile.Name != "" {
		t.Errorf("name = %q, want empty", profile.Name)
	}
}

// TestGetNostrProfileCachedNotesAKeyIsCaseInsensitive guards the cache key: the
// catalog may spell a bech32 key in any case, and the same account must not
// occupy two cache entries.
func TestGetNostrProfileCachedNotesAKeyIsCaseInsensitive(t *testing.T) {
	isolatedNostrCache(t)

	relay := newNostrStubRelay(t, serveKind0(t, `{"name":"jack"}`))
	useStubRelay(t, relay.url)

	if _, err := GetNostrProfileCached(jackNpub, false); err != nil {
		t.Fatalf("lowercase call: %v", err)
	}
	if _, err := GetNostrProfileCached(strings.ToUpper(jackNpub), false); err != nil {
		t.Fatalf("uppercase call: %v", err)
	}

	if got := relay.connections(); got != 1 {
		t.Errorf("relay saw %d connections for the same key in two cases, want 1", got)
	}
}
