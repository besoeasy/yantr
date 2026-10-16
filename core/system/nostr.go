package system

// Nostr developer profiles.
//
// An app can credit a Nostr account in `developers` as an `npub1…`. Turning that
// into something worth showing needs a relay, which is the one place this package
// talks to a third party over a protocol other than HTTP.
//
// It asks more than one relay. That was a deliberate simplification first, on the
// reasoning that one account's name and picture are the same everywhere; measuring
// it against real accounts showed the reasoning is wrong. Of three accounts
// checked, `relay.primal.net` served kind-0 for two and had no event at all for
// the third — a clean end-of-stored-events, not a failure. Nostr has no rule that
// a relay must hold every event it is asked about, so coverage is per-relay and a
// single relay is a single point of blindness.
//
// So the query fans out to a small set and takes the first profile that arrives.
// That also makes it resilient: the same run saw one relay refuse a connection
// outright while another answered. The cost is a handful of sockets, and the
// seven-day cache means it is paid about once per week per credited developer
// rather than once per page view.
//
// The set is a var so tests can replace it, and overridable with
// YANTR_NOSTR_RELAYS for a host that cannot reach these.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"core/nip19"
)

// defaultNostrRelays is the relay set. The order is a preference, not a
// requirement: the fan-out takes whichever answers first.
//
// Chosen by measured coverage rather than reputation — each of these answered a
// real query during development, and between them they served every account
// tested. YANTR_NOSTR_RELAYS overrides it as a comma-separated list, for a host
// behind a network that blocks these or one running its own relay.
func defaultNostrRelays() []string {
	if raw := strings.TrimSpace(os.Getenv("YANTR_NOSTR_RELAYS")); raw != "" {
		var relays []string
		for _, relay := range strings.Split(raw, ",") {
			if relay = strings.TrimSpace(relay); relay != "" {
				relays = append(relays, relay)
			}
		}
		if len(relays) > 0 {
			return relays
		}
	}
	return []string{
		"wss://nos.lol",
		"wss://relay.primal.net",
		"wss://nostr.mom",
	}
}

// nostrRelayURLs is the set this package queries. A var so tests can replace it.
var nostrRelayURLs = defaultNostrRelays()

// nostrQueryTimeout bounds the whole fan-out — every dial, REQ, EVENT and EOSE.
// A relay that accepts the socket and then says nothing is the failure this
// exists for. A var, like the relay set, so a test can shorten it rather than
// wait eight seconds to prove the bound holds.
var nostrQueryTimeout = 8 * time.Second

const (
	// nostrProfileTTL matches the GitHub profile cache: seven days. Kind-0
	// metadata is reference data — a name, a picture, an about line — and the
	// same reasoning applies, except that the cost of a miss here is a socket
	// and a round trip rather than a rate-limit unit.
	nostrProfileTTL = 7 * 24 * time.Hour

	// A miss is cached briefly, as for GitHub: an account that has not published
	// a profile yet may publish one this afternoon, and that should not be
	// hidden for a week.
	nostrProfileMissTTL = time.Hour
)

// ErrNostrNoProfile separates "this key has no kind-0 metadata on the relay"
// from "the relay could not be reached". Both leave a developer page without a
// profile card, but only the second is worth logging as a problem, and only the
// first is a 404 rather than a bad gateway.
var ErrNostrNoProfile = errors.New("no nostr profile published")

// NostrProfile is the subset of a kind-0 event a developer page shows. Field
// names are camelCase on the wire to match the GitHub profile payload the UI
// already reads.
//
// Key is the identifier as the catalog declared it, which may be an npub or an
// nprofile. Pubkey is the hex form both decode to. The UI does not need either
// to build its QR code — the code encodes the catalog's own handle, which is
// valid NIP-21 whichever of the two it is — but a page that shows a Nostr
// profile should be able to say which account it belongs to.
type NostrProfile struct {
	Pubkey      string `json:"pubkey"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	About       string `json:"about"`
	Picture     string `json:"picture"`
	NIP05       string `json:"nip05"`
	LUD16       string `json:"lud16"`
	Website     string `json:"website"`
	CreatedAt   int64  `json:"createdAt"`
	FetchedAt   string `json:"fetchedAt"`
}

type nostrEntry struct {
	profile *NostrProfile
	err     error
	exp     time.Time
}

var (
	nostrCacheMu sync.Mutex
	nostrCache   = map[string]nostrEntry{}
)

// nostrEvent is the part of a NIP-01 event this package reads.
type nostrEvent struct {
	Pubkey    string `json:"pubkey"`
	CreatedAt int64  `json:"created_at"`
	Kind      int    `json:"kind"`
	Content   string `json:"content"`
}

// nostrMetadata is the shape NIP-01 defines for kind-0 content. Clients differ
// over which of name/display_name holds the human-readable name, so both are
// carried and the caller decides.
type nostrMetadata struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	About       string `json:"about"`
	Picture     string `json:"picture"`
	NIP05       string `json:"nip05"`
	LUD16       string `json:"lud16"`
	Website     string `json:"website"`
}

// GetNostrProfileCached returns one account's kind-0 metadata, cached for a
// week, keyed by the npub as written in the catalog.
func GetNostrProfileCached(npub string, forceRefresh bool) (*NostrProfile, error) {
	key := strings.ToLower(strings.TrimSpace(npub))

	// Decoded before anything is cached, so a string that is not a profile
	// identifier can never occupy a cache entry.
	if key == "" {
		return nil, fmt.Errorf("empty nostr key")
	}
	if _, err := nip19.PubkeyHex(key); err != nil {
		return nil, err
	}

	nostrCacheMu.Lock()
	defer nostrCacheMu.Unlock()

	if e, ok := nostrCache[key]; ok && !forceRefresh && time.Now().Before(e.exp) {
		return e.profile, e.err
	}

	ctx, cancel := context.WithTimeout(context.Background(), nostrQueryTimeout)
	defer cancel()

	profile, err := fetchNostrProfile(ctx, key)

	if err != nil {
		// A failed refresh must not wipe a good cached value, exactly as in the
		// GitHub caches: offline, blocked and relay-having-a-day all look the
		// same from here, and a week-old name beats a blank card.
		if e, ok := nostrCache[key]; ok && e.profile != nil {
			return e.profile, nil
		}
		nostrCache[key] = nostrEntry{err: err, exp: time.Now().Add(nostrProfileMissTTL)}
		return nil, err
	}

	nostrCache[key] = nostrEntry{profile: profile, exp: time.Now().Add(nostrProfileTTL)}
	return profile, nil
}

// fetchNostrProfile asks every relay in parallel and returns the first profile
// that arrives.
//
// The relays are asked concurrently rather than in sequence because a miss is
// the common case for any given relay, and a sequence would make the wait grow
// with the number of them. First-answer-wins rather than wait-for-the-newest
// because the difference is one relay holding a stale kind-0 for an account
// another relay has current — a name and a picture, both cached for a week.
//
// A relay that simply does not have the event is not an error, and neither is one
// that cannot be reached. Both mean "try the next one". Only when no relay
// produced a profile does it matter which of those happened: if every relay
// answered and none had the event, that is a 404 rather than a gateway failure.
func fetchNostrProfile(ctx context.Context, npub string) (*NostrProfile, error) {
	pubkey, err := nip19.PubkeyHex(npub)
	if err != nil {
		return nil, err
	}

	relays := nostrRelayURLs
	if len(relays) == 0 {
		return nil, fmt.Errorf("no nostr relay configured")
	}

	// Cancelling on return is what stops the relays that lost the race: they
	// notice through their socket deadline, and the buffered channel means none
	// of them can block on the way out.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type outcome struct {
		profile *NostrProfile
		err     error
	}
	results := make(chan outcome, len(relays))

	for _, relay := range relays {
		go func(relay string) {
			profile, err := fetchNostrProfileFrom(ctx, relay, pubkey, npub)
			results <- outcome{profile, err}
		}(relay)
	}

	var (
		firstErr error
		sawNone  bool
	)

	for range relays {
		result := <-results
		switch {
		case result.profile != nil:
			return result.profile, nil
		case result.err == nil:
			// Nothing is ever both nil and nil: an absent profile is an error.
			sawNone = true
		case errors.Is(result.err, ErrNostrNoProfile):
			sawNone = true
		default:
			if firstErr == nil {
				firstErr = result.err
			}
		}
	}

	// One relay being unreachable means the absence is unproven, so a transport
	// failure outranks a clean "no events stored" from the others.
	if firstErr != nil {
		return nil, firstErr
	}
	if sawNone {
		return nil, fmt.Errorf("%w for %s", ErrNostrNoProfile, npub)
	}
	return nil, fmt.Errorf("no nostr relay answered for %s", npub)
}

// fetchNostrProfileFrom runs one REQ/EVENT/EOSE exchange against one relay.
//
// A relay that stores no event for this author returns ErrNostrNoProfile, which
// the caller treats as "ask the next one" rather than as a failure.
func fetchNostrProfileFrom(ctx context.Context, relay, pubkey, npub string) (*NostrProfile, error) {
	conn, err := dialWebSocket(ctx, relay)
	if err != nil {
		return nil, err
	}
	defer conn.close()

	const sub = "yantr-profile"

	req, err := json.Marshal([]interface{}{
		"REQ",
		sub,
		map[string]interface{}{
			"kinds":   []int{0},
			"authors": []string{pubkey},
			// A handful rather than one: kind-0 events replace rather than
			// chain, and a relay is free to answer `limit: 1` with any single
			// stored event instead of the newest. Five is still a trivial
			// response, and the newest of them is picked below.
			"limit": 5,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.writeText(req); err != nil {
		return nil, err
	}

	event, notice, err := readProfileEvent(conn, sub)
	if err != nil {
		return nil, err
	}

	// Unsubscribe before leaving, so a relay that keeps subscriptions alive does
	// not hold resources for a connection we are about to drop anyway.
	if closeMsg, err := json.Marshal([]interface{}{"CLOSE", sub}); err == nil {
		conn.writeText(closeMsg)
	}

	if event == nil {
		if notice != "" {
			return nil, fmt.Errorf("relay notice: %s", notice)
		}
		return nil, fmt.Errorf("%w for %s", ErrNostrNoProfile, npub)
	}

	// Kind-0 content is a JSON document inside a JSON string. A well-formed
	// event can still carry content that is not JSON — clients wrote plain text
	// there historically — so a decode failure is not an error: the pubkey and
	// the key itself are still worth showing.
	var meta nostrMetadata
	json.Unmarshal([]byte(event.Content), &meta)

	return &NostrProfile{
		Pubkey:      pubkey,
		Key:         npub,
		Name:        meta.Name,
		DisplayName: meta.DisplayName,
		About:       meta.About,
		Picture:     meta.Picture,
		NIP05:       meta.NIP05,
		LUD16:       meta.LUD16,
		Website:     meta.Website,
		CreatedAt:   event.CreatedAt,
		FetchedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// readProfileEvent reads the subscription until the relay signals the end of
// stored events, keeping the newest kind-0 event it delivered.
//
// The second return is the last NOTICE text, which is how a relay explains a
// refusal ("rate limited: ...") and is worth surfacing when no event arrives.
func readProfileEvent(conn *wsConn, sub string) (*nostrEvent, string, error) {
	var (
		newest *nostrEvent
		notice string
	)

	for {
		opcode, payload, err := conn.readMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// The relay closed the connection. Anything already delivered is
				// still valid, so this is not a failure.
				return newest, notice, nil
			}
			return nil, "", err
		}
		if opcode != wsOpText {
			continue
		}

		var msg []json.RawMessage
		if err := json.Unmarshal(payload, &msg); err != nil || len(msg) == 0 {
			continue
		}

		var verb string
		if err := json.Unmarshal(msg[0], &verb); err != nil {
			continue
		}

		switch verb {
		case "EVENT":
			if len(msg) < 3 || !rawStringIs(msg[1], sub) {
				continue
			}
			var event nostrEvent
			if err := json.Unmarshal(msg[2], &event); err != nil {
				continue
			}
			// Only kind 0 counts. A relay answering a different kind means the
			// filter was ignored, and a note is not a profile.
			if event.Kind != 0 {
				continue
			}
			if newest == nil || event.CreatedAt > newest.CreatedAt {
				newest = &event
			}

		case "EOSE":
			return newest, notice, nil

		case "NOTICE", "CLOSED":
			if len(msg) >= 2 {
				var text string
				if json.Unmarshal(msg[1], &text) == nil {
					notice = text
				}
			}

		default:
			// Relays send frames we have no use for. Skipping them is correct;
			// failing the query over one is not.
		}
	}
}

// rawStringIs reports whether a raw JSON value is the given string.
func rawStringIs(raw json.RawMessage, want string) bool {
	var got string
	return json.Unmarshal(raw, &got) == nil && got == want
}
