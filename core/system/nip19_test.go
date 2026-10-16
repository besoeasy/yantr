package system

import (
	"strings"
	"testing"
)

// The vectors below were established outside this repository, because a
// hand-written decoder tested only against its own encoder proves nothing:
//
//   - fiatjafNpub/fiatjafHex and jackNpub/jackHex come from an independent
//     bech32 implementation written from BIP-173, and were then confirmed
//     against the live relay: a kind-0 query for each hex returns the account
//     those npubs belong to (one has nip05 `_@fiatjaf.com`, the other
//     `jack@primal.net`).
//   - fiatjafNprofile is the value njump.me publishes for that account, so it
//     is a real-world nprofile produced by a third-party client, relay hints
//     and all, rather than something our own encoder would choose to emit.
const (
	fiatjafNpub = "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6"
	fiatjafHex  = "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d"

	jackNpub = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m"
	jackHex  = "82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2"

	fiatjafNprofile = "nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpz3mhxue69uhhyetvv9ujuerpd46hxtnfduqs6amnwvaz7tmwdaejumr0dsfg70ls"

	// One pubkey entry followed by a single relay hint.
	jackNprofile = "nprofile1qqsgydql3q4ka27d9wnlrmus4tvkrnc8ftc4h8h5fgyln54gl0a7dgspzemhxue69uhhyetvv9ujuurjd9kkzmpwdejhgw2eqf8"

	// Well-formed bech32 whose payload is the wrong size for its prefix.
	npub31Bytes = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmucs89urk"
	npub33Bytes = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qqryfd9e"

	// The same 32-byte payload under prefixes that do not name a profile.
	noteKey32 = "note1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q7k28gn"
	nsecKey32 = "nsec1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qr2zmhw"

	// Valid bech32 wrapping malformed TLV.
	nprofileNoPubkey = "nprofile1qypkzcnre6e3r9"
	nprofilePubkey16 = "nprofile1qqggydql3q4ka27d9wnlrmus4tvkzru9ad2"
	nprofileOverrun  = "nprofile1qqsgydql3q4ka27d9wnlrmus4tvkrnc8ftcsal5jy4"

	// The last character changed to another symbol of the alphabet, so the
	// string stays well-formed but the checksum no longer verifies.
	npubBrokenChecksum = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63q"

	// `b` is not in the bech32 alphabet (q p z r y 9 x 8 g f 2 t v d w 0 s 3 j
	// n 5 4 k h c e 6 m u a 7 l).
	npubBadAlphabet = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63b"
)

func TestNostrPubkeyHex(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"npub", jackNpub, jackHex},
		{"a second account", fiatjafNpub, fiatjafHex},
		{"nprofile with several relay hints", fiatjafNprofile, fiatjafHex},
		{"nprofile with one relay hint", jackNprofile, jackHex},
		{"surrounding whitespace is tolerated", "  " + jackNpub + "\n", jackHex},
		{"uppercase is accepted", strings.ToUpper(jackNpub), jackHex},

		// BIP-173 allows either case but not both, and the catalog deliberately
		// does not rewrite a Nostr key's case, so a hand-written entry can
		// arrive mixed. The checksum still has to verify.
		{"mixed case is accepted", "npub1Sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m", jackHex},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NostrPubkeyHex(tc.key)
			if err != nil {
				t.Fatalf("NostrPubkeyHex(%q) returned an error: %v", tc.key, err)
			}
			if got != tc.want {
				t.Errorf("NostrPubkeyHex(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestNostrPubkeyHexRejects(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"no separator", "npubwithoutseparator"},
		{"empty human-readable part", "1qqqqqq"},
		{"too short to hold a checksum", "npub1qqq"},
		{"character outside the bech32 alphabet", npubBadAlphabet},
		{"broken checksum", npubBrokenChecksum},
		{"31-byte npub payload", npub31Bytes},
		{"33-byte npub payload", npub33Bytes},
		{"note is an event id, not a profile", noteKey32},
		{"nsec is a private key", nsecKey32},
		{"nprofile with no pubkey entry", nprofileNoPubkey},
		{"nprofile with a 16-byte pubkey", nprofilePubkey16},
		{"nprofile whose entry length overruns the payload", nprofileOverrun},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NostrPubkeyHex(tc.key)
			if err == nil {
				t.Fatalf("NostrPubkeyHex(%q) = %q, want an error", tc.key, got)
			}
			if got != "" {
				t.Errorf("NostrPubkeyHex(%q) returned %q alongside an error", tc.key, got)
			}
		})
	}
}
