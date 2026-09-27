package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"
)

// newTestKey returns a fresh Ed25519 keypair plus its hex encodings.
func newTestKey(t *testing.T) (pubHex, privHex string, priv ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return hex.EncodeToString(pub), hex.EncodeToString(priv), priv
}

// mintToken builds a token exactly the way the frontend does: message is
// "{timestamp}:{nonce}" and the signature covers those raw message bytes.
func mintToken(t *testing.T, priv ed25519.PrivateKey, pubHex string, tsMs int64, nonce string) string {
	t.Helper()
	message := strconv.FormatInt(tsMs, 10) + ":" + nonce
	sig := ed25519.Sign(priv, []byte(message))
	payload := map[string]interface{}{
		"publickey": pubHex,
		"signature": hex.EncodeToString(sig),
		"message":   message,
		"timestamp": tsMs,
		"nonce":     nonce,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

// rewriteTimestampField re-encodes a token with a different `timestamp` JSON
// field while leaving the signed message untouched. This is the attack the
// timestamp binding is meant to defeat.
func rewriteTimestampField(t *testing.T, token string, newTs int64) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("decode token: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal token: %v", err)
	}
	payload["timestamp"] = newTs
	out, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("re-marshal token: %v", err)
	}
	return base64.StdEncoding.EncodeToString(out)
}

func testConfig(pubHex string) *AuthConfig {
	return &AuthConfig{PublicKeyHex: pubHex}
}

// A freshly minted token must verify.
func TestVerifyTokenAcceptsFreshToken(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	nonce := "0123456789abcdef0123456789abcdef"
	token := mintToken(t, priv, pubHex, time.Now().UnixMilli(), nonce)

	if err := VerifyToken(token, testConfig(pubHex)); err != nil {
		t.Fatalf("fresh token rejected: %v", err)
	}
}

// The regression this whole change exists for: the `timestamp` JSON field is
// not covered by the signature, so rewriting it must not resurrect an expired
// token. The server must read the timestamp out of the signed message.
func TestVerifyTokenIgnoresUnsignedTimestampField(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	nonce := "1111111111111111111111111111aaaa"

	// Mint well outside the 60s window.
	old := time.Now().Add(-10 * time.Minute).UnixMilli()
	stale := mintToken(t, priv, pubHex, old, nonce)

	// Sanity: the token really is expired.
	if err := VerifyToken(stale, testConfig(pubHex)); err == nil {
		t.Fatal("expected a 10-minute-old token to be rejected")
	}

	// The attack: bump the unsigned field to now, keep the signature as-is.
	forged := rewriteTimestampField(t, stale, time.Now().UnixMilli())
	if err := VerifyToken(forged, testConfig(pubHex)); err == nil {
		t.Fatal("unsigned timestamp field overrode the signed message: token accepted after expiry")
	}
}

// Same idea from the other direction: a future timestamp inside the message
// must be rejected too, or the window widens.
func TestVerifyTokenRejectsFutureSignedTimestamp(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	future := time.Now().Add(10 * time.Minute).UnixMilli()
	token := mintToken(t, priv, pubHex, future, "2222222222222222222222222222bbbb")

	if err := VerifyToken(token, testConfig(pubHex)); err == nil {
		t.Fatal("expected a token 10 minutes in the future to be rejected")
	}
}

// Exactly at the boundary: inside the window verifies, one second past it does not.
func TestVerifyTokenEnforcesSixtySecondWindow(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	cfg := testConfig(pubHex)

	inside := mintToken(t, priv, pubHex, time.Now().Add(-59*time.Second).UnixMilli(), "3333333333333333333333333333cccc")
	if err := VerifyToken(inside, cfg); err != nil {
		t.Fatalf("token at -59s should be inside the window: %v", err)
	}

	outside := mintToken(t, priv, pubHex, time.Now().Add(-61*time.Second).UnixMilli(), "4444444444444444444444444444dddd")
	if err := VerifyToken(outside, cfg); err == nil {
		t.Fatal("token at -61s should be outside the 60s window")
	}
}

// The same signed message must not be usable twice, even while it is still
// inside the validity window.
func TestVerifyTokenRejectsReplayOfSameMessage(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	cfg := testConfig(pubHex)
	token := mintToken(t, priv, pubHex, time.Now().UnixMilli(), "5555555555555555555555555555eeee")

	if err := VerifyToken(token, cfg); err != nil {
		t.Fatalf("first use should succeed: %v", err)
	}
	if err := VerifyToken(token, cfg); err == nil {
		t.Fatal("replayed token accepted a second time")
	}
}

// Concurrent replays of one token: exactly one may win. This also exercises the
// check-and-insert under concurrency.
func TestVerifyTokenReplayIsAtomicUnderConcurrency(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	cfg := testConfig(pubHex)
	token := mintToken(t, priv, pubHex, time.Now().UnixMilli(), "6666666666666666666666666666ffff")

	const racers = 16
	var wg sync.WaitGroup
	results := make([]error, racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx] = VerifyToken(token, cfg)
		}(i)
	}
	close(start)
	wg.Wait()

	accepted := 0
	for _, err := range results {
		if err == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("expected exactly 1 of %d concurrent replays to be accepted, got %d", racers, accepted)
	}
}

// A token signed by the wrong key must not pass, and must not consume a nonce.
func TestVerifyTokenRejectsForeignSignature(t *testing.T) {
	_, _, attackerPriv := newTestKey(t)
	adminPubHex, _, _ := newTestKey(t)
	token := mintToken(t, attackerPriv, adminPubHex, time.Now().UnixMilli(), "7777777777777777777777777777aaaa")

	if err := VerifyToken(token, testConfig(adminPubHex)); err == nil {
		t.Fatal("token signed by a foreign key was accepted")
	}
}

// A public key that is not the configured one is rejected before any signature
// work, and the token must not burn its nonce.
func TestVerifyTokenRejectsUnknownPublicKey(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	otherPubHex, _, _ := newTestKey(t)
	nonce := "8888888888888888888888888888bbbb"
	token := mintToken(t, priv, pubHex, time.Now().UnixMilli(), nonce)

	if err := VerifyToken(token, testConfig(otherPubHex)); err == nil {
		t.Fatal("token from an unconfigured public key was accepted")
	}
	// Same token must still be usable against its own key: a rejected request
	// for the wrong key must not consume the nonce.
	if err := VerifyToken(token, testConfig(pubHex)); err != nil {
		t.Fatalf("token should still be valid for its own key: %v", err)
	}
}

// Malformed messages must be rejected rather than silently treated as fresh.
func TestDecodeTokenRejectsMalformedMessages(t *testing.T) {
	cases := []struct {
		name    string
		message string
	}{
		{"no separator", "1234567890"},
		{"empty nonce", "1234567890:"},
		{"nonce too short", "1234567890:abcd"},
		{"non numeric timestamp", "notanumber:0123456789abcdef0123456789abcdef"},
		{"zero timestamp", "0:0123456789abcdef0123456789abcdef"},
		{"negative timestamp", "-5:0123456789abcdef0123456789abcdef"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := parseSignedMessage(tc.message); err == nil {
				t.Fatalf("expected %q to be rejected", tc.message)
			}
		})
	}
}

// A message with no separator must not decode, even carrying a valid signature.
func TestDecodeTokenRejectsUnsignedFieldSmuggling(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	// Sign a well-formed message, then swap in a different one in the payload.
	good := mintToken(t, priv, pubHex, time.Now().UnixMilli(), "9999999999999999999999999999cccc")
	raw, err := base64.StdEncoding.DecodeString(good)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	payload["message"] = "garbage-without-separator"
	bad, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := decodeToken(base64.StdEncoding.EncodeToString(bad)); err == nil {
		t.Fatal("token with a malformed signed message decoded successfully")
	}
}

// parseSignedMessage must return exactly the values embedded in the message.
func TestParseSignedMessageRoundTrip(t *testing.T) {
	ts := int64(1700000000123)
	nonce := "0123456789abcdef0123456789abcdef"
	gotTS, gotNonce, err := parseSignedMessage(strconv.FormatInt(ts, 10) + ":" + nonce)
	if err != nil {
		t.Fatalf("parseSignedMessage: %v", err)
	}
	if gotTS != ts {
		t.Fatalf("timestamp: got %d want %d", gotTS, ts)
	}
	if gotNonce != nonce {
		t.Fatalf("nonce: got %q want %q", gotNonce, nonce)
	}
}

// decodeToken overwrites the unsigned fields with the signed values.
func TestDecodeTokenDerivesTimestampFromMessage(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	signedTS := time.Now().UnixMilli()
	nonce := "abababababababababababababababab"
	token := mintToken(t, priv, pubHex, signedTS, nonce)

	// Claim a wildly different timestamp on the wire.
	tampered := rewriteTimestampField(t, token, 1)

	tok, err := decodeToken(tampered)
	if err != nil {
		t.Fatalf("decodeToken: %v", err)
	}
	if tok.Timestamp != signedTS {
		t.Fatalf("Timestamp = %d, want the signed value %d", tok.Timestamp, signedTS)
	}
	if tok.Nonce != nonce {
		t.Fatalf("Nonce = %q, want %q", tok.Nonce, nonce)
	}
}

// Distinct tokens minted in the same millisecond must not collide, since the
// nonce is what distinguishes them.
func TestVerifyTokenAcceptsDistinctNoncesInSameMillisecond(t *testing.T) {
	pubHex, _, priv := newTestKey(t)
	cfg := testConfig(pubHex)
	ts := time.Now().UnixMilli()

	for _, nonce := range []string{
		"0000000000000000000000000000000a",
		"0000000000000000000000000000000b",
		"0000000000000000000000000000000c",
	} {
		if err := VerifyToken(mintToken(t, priv, pubHex, ts, nonce), cfg); err != nil {
			t.Fatalf("distinct nonce %q rejected: %v", nonce, err)
		}
	}
}

// The nonce cache prunes expired entries rather than growing without bound.
func TestConsumeNoncePrunesExpiredEntries(t *testing.T) {
	seenNoncesMu.Lock()
	seenNonces = make(map[string]int64)
	seenNoncesMu.Unlock()

	now := time.Now().UnixMilli()
	// Two entries that expired a minute ago, one still live.
	seenNoncesMu.Lock()
	seenNonces["stale-a"] = now - 60_000
	seenNonces["stale-b"] = now - 60_000
	seenNoncesMu.Unlock()

	if consumeNonce("fresh-c", now+60_000) {
		// expected
	} else {
		t.Fatal("a fresh nonce was reported as a replay")
	}

	seenNoncesMu.Lock()
	_, staleA := seenNonces["stale-a"]
	_, staleB := seenNonces["stale-b"]
	_, freshC := seenNonces["fresh-c"]
	size := len(seenNonces)
	seenNoncesMu.Unlock()

	if staleA || staleB {
		t.Fatal("expired nonces were not pruned")
	}
	if !freshC {
		t.Fatal("live nonce was dropped during pruning")
	}
	if size != 1 {
		t.Fatalf("cache size = %d, want 1 after pruning", size)
	}
}

// At the cap the cache evicts rather than growing. The entry closest to expiry
// goes first, since it stops mattering soonest.
func TestConsumeNonceStaysBoundedAtCap(t *testing.T) {
	seenNoncesMu.Lock()
	seenNonces = make(map[string]int64)
	seenNoncesMu.Unlock()

	now := time.Now().UnixMilli()
	// Fill so that the total lands exactly on the cap; the next insert must evict.
	for i := 0; i < maxTrackedNonces-1; i++ {
		nonce := "fill" + strconv.Itoa(i) + "0000000000000000"
		seenNoncesMu.Lock()
		seenNonces[nonce] = now + int64(10_000+i)
		seenNoncesMu.Unlock()
	}
	// One entry expires well before all the fillers, but still safely inside
	// the window — an expiry only a millisecond out would be swept by the
	// prune step instead of the eviction step this test is checking.
	seenNoncesMu.Lock()
	seenNonces["soonest"] = now + 5_000
	filled := len(seenNonces)
	seenNoncesMu.Unlock()
	if filled != maxTrackedNonces {
		t.Fatalf("setup: cache holds %d entries, want exactly the cap %d", filled, maxTrackedNonces)
	}

	if !consumeNonce("brandnew00000000000000000000", now+60_000) {
		t.Fatal("new nonce rejected at the cap")
	}

	seenNoncesMu.Lock()
	_, soonestStillThere := seenNonces["soonest"]
	_, keptNew := seenNonces["brandnew00000000000000000000"]
	size := len(seenNonces)
	seenNoncesMu.Unlock()

	if soonestStillThere {
		t.Fatal("soonest-to-expire entry was not evicted at the cap")
	}
	if !keptNew {
		t.Fatal("the newly consumed nonce should have been retained")
	}
	if size > maxTrackedNonces {
		t.Fatalf("cache grew past the cap: %d > %d", size, maxTrackedNonces)
	}
}
