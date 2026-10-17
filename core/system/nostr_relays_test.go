package system

import "testing"

// YANTR_NOSTR_RELAYS is the whole reason this package still exists, so the
// override's edges are pinned here: a host whose network blocks the public
// relays, or one running its own, sets one variable and every Nostr read and
// publish in the UI follows.
func TestDefaultNostrRelaysReadsTheOverride(t *testing.T) {
	t.Setenv("YANTR_NOSTR_RELAYS", " wss://relay.example.com , wss://other.example.com ")

	got := defaultNostrRelays()
	want := []string{"wss://relay.example.com", "wss://other.example.com"}
	if len(got) != len(want) {
		t.Fatalf("defaultNostrRelays() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("relay %d = %q, want %q (entries must be trimmed, order kept)", i, got[i], want[i])
		}
	}
}

// An override that is set but empty must not empty the set. The UI reads this
// list as "the relays to ask", and an empty list reads to it as "ask nothing",
// which would silently stop every Nostr feature instead of falling back.
func TestDefaultNostrRelaysIgnoresAnEmptyOverride(t *testing.T) {
	for _, raw := range []string{"", "   ", ",", " , "} {
		t.Setenv("YANTR_NOSTR_RELAYS", raw)
		if got := defaultNostrRelays(); len(got) == 0 {
			t.Fatalf("defaultNostrRelays() with override %q = empty, want the built-in set", raw)
		}
	}
}

// Without the variable the set is the measured default: three relays, all of
// which answered real queries during development.
func TestDefaultNostrRelaysFallsBackToTheMeasuredSet(t *testing.T) {
	t.Setenv("YANTR_NOSTR_RELAYS", "")

	got := defaultNostrRelays()
	want := []string{"wss://nos.lol", "wss://relay.primal.net", "wss://nostr.mom"}
	if len(got) != len(want) {
		t.Fatalf("defaultNostrRelays() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("relay %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// NostrRelays hands out a copy. The set is read by a handler on every UI page
// load; a caller that sorted or truncated it in place would change what every
// later reader sees, with no way to notice.
func TestNostrRelaysReturnsACopy(t *testing.T) {
	previous := nostrRelayURLs
	nostrRelayURLs = []string{"wss://a.example.com", "wss://b.example.com"}
	t.Cleanup(func() { nostrRelayURLs = previous })

	first := NostrRelays()
	if len(first) != 2 {
		t.Fatalf("NostrRelays() = %v, want two relays", first)
	}
	first[0] = "wss://mutated.example.com"

	if again := NostrRelays(); again[0] != "wss://a.example.com" {
		t.Errorf("mutating the returned slice changed the set: %v", again)
	}
}