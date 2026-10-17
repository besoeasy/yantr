// Package system reads host facts over HTTP and holds the configuration other
// packages share. This file is the Nostr relay set and nothing else.
//
// Yantr's Nostr traffic does not touch this package any more. It used to: a
// developer's kind-0 was proxied through /api/nostr/profile/{key}, which meant
// this package had to dial relays, parse NIP-01 and carry a week-long cache —
// server state for a query with no state in it. Both Nostr reads (comments,
// kind-0) and the publish path now happen in the browser, in
// `ui/src/utils/relay.js`, through the reader's own signer.
//
// What remains server-side is the set, because the set is configuration.
// `YANTR_NOSTR_RELAYS` redirects every Nostr read and every publish in the UI
// at once, for a host whose network blocks the public relays or that runs its
// own. The UI asks for it at /api/nostr/relays and falls back to its own copy
// when the API cannot answer, so there is one override rather than two lists
// that can drift apart.
//
// The set is asked of more than one relay by the browser, for the reason the Go
// fan-out had it: coverage is per-relay. Measured during development, of three
// accounts checked `relay.primal.net` served kind-0 for two and had no event at
// all for the third — a clean end-of-stored-events, not a failure. Nostr has no
// rule that a relay must hold every event it is asked about, so one relay is one
// point of blindness.

package system

import (
	"os"
	"strings"
)

// defaultNostrRelays is the relay set. The order is a preference, not a
// requirement: the browser's fan-out takes the union of what they answer and
// asks them all at once.
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

// nostrRelayURLs is the set this package reports. A var so tests can replace it.
var nostrRelayURLs = defaultNostrRelays()

// NostrRelays returns a copy of the configured relay set, so a caller cannot
// reorder or truncate what every other caller sees.
func NostrRelays() []string {
	out := make([]string, len(nostrRelayURLs))
	copy(out, nostrRelayURLs)
	return out
}

// SetNostrRelays replaces the set at runtime.
//
// Nothing in the server changes it while running — the set is read from
// YANTR_NOSTR_RELAYS once at startup — so this exists for tests, which need to
// reach the states the environment makes awkward to reach: an override in the
// middle of a suite, or the empty set that no real configuration produces.
func SetNostrRelays(urls []string) {
	nostrRelayURLs = urls
}