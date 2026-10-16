package nip19

// Package nip19 decodes Nostr's bech32 identifiers.
//
// A developer is stored in `x-yantr.developers` as an `npub1…` (or `nprofile1…`)
// string, which is bech32. Two things need that decoded: a relay can only be asked
// about a 32-byte hex pubkey, and the catalog needs to know a key is a key before
// it renders it as a profile link. So this sits in its own package — it is pure
// string handling with no I/O and no dependency on anything else in the tree, and
// both core/apps and core/system need it.
//
// Hand-rolled rather than pulled from a library because this is the whole of the
// job: BIP-173 checksum verification and a 5-to-8 bit regroup, about a hundred
// lines with no state and no I/O, against a dependency that would otherwise be
// carried by every build for one feature.

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// bech32Charset is the BIP-173 data symbol set, ordered by value.
const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

// bech32Gen is the generator polynomial for the BIP-173 checksum.
var bech32Gen = [5]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

// bech32Polymod folds h(hrp) || data into the checksum register.
func bech32Polymod(values []byte) uint32 {
	chk := uint32(1)
	for _, v := range values {
		b := chk >> 25
		chk = (chk&0x01ffffff)<<5 ^ uint32(v)
		for i := 0; i < 5; i++ {
			if (b>>uint(i))&1 == 1 {
				chk ^= bech32Gen[i]
			}
		}
	}
	return chk
}

// bech32HRPExpand expands the human-readable part for the checksum: the high
// bits of each character, a zero separator, then the low bits.
func bech32HRPExpand(hrp string) []byte {
	out := make([]byte, 0, len(hrp)*2+1)
	for i := 0; i < len(hrp); i++ {
		out = append(out, hrp[i]>>5)
	}
	out = append(out, 0)
	for i := 0; i < len(hrp); i++ {
		out = append(out, hrp[i]&31)
	}
	return out
}

// convertBits regroups a symbol stream from one bit width to another, as
// BIP-173's convertbits does with pad=false.
func convertBits(data []byte, from, to uint, pad bool) []byte {
	var acc uint32
	var bits uint
	maxv := uint32(1)<<to - 1

	out := make([]byte, 0, len(data)*int(from)/int(to)+1)
	for _, v := range data {
		if uint32(v)>>from != 0 {
			return nil
		}
		acc = acc<<from | uint32(v)
		bits += from
		for bits >= to {
			bits -= to
			out = append(out, byte(acc>>bits&maxv))
		}
	}
	if pad {
		if bits > 0 {
			out = append(out, byte(acc<<(to-bits)&maxv))
		}
	} else if bits >= from || acc<<(to-bits)&maxv != 0 {
		// Trailing bits that do not compose a whole symbol, or a non-zero
		// remainder: the encoding is not canonical.
		return nil
	}
	return out
}

// bech32Decode verifies the checksum and returns the human-readable part with
// the decoded payload.
func bech32Decode(s string) (string, []byte, error) {
	if s == "" {
		return "", nil, fmt.Errorf("empty bech32 string")
	}

	// Lowercased before decoding rather than rejected for mixed case. BIP-173
	// allows either case but not both, and the catalog deliberately does not
	// rewrite a Nostr key's case on the way in (core/apps/catalog.go), so a
	// hand-written entry can legitimately arrive mixed. The checksum still has
	// to verify, so the tolerance costs nothing.
	lower := strings.ToLower(s)

	sep := strings.LastIndexByte(lower, '1')
	if sep < 1 || sep+7 > len(lower) {
		return "", nil, fmt.Errorf("not a bech32 string")
	}
	hrp := lower[:sep]
	for i := 0; i < len(hrp); i++ {
		if hrp[i] < 33 || hrp[i] > 126 {
			return "", nil, fmt.Errorf("bech32 human-readable part contains %q", hrp[i])
		}
	}

	values := make([]byte, 0, len(lower)-sep-1)
	for i := sep + 1; i < len(lower); i++ {
		idx := strings.IndexByte(bech32Charset, lower[i])
		if idx < 0 {
			return "", nil, fmt.Errorf("invalid bech32 character %q", lower[i])
		}
		values = append(values, byte(idx))
	}

	if bech32Polymod(append(bech32HRPExpand(hrp), values...)) != 1 {
		return "", nil, fmt.Errorf("bech32 checksum failed")
	}

	// The last six symbols are the checksum.
	payload := convertBits(values[:len(values)-6], 5, 8, false)
	if payload == nil {
		return "", nil, fmt.Errorf("bech32 payload is not byte-aligned")
	}
	return hrp, payload, nil
}

// PubkeyHex turns an NIP-19 public-key identifier into the hex form a relay
// filters on.
//
// `npub` carries the key directly; `nprofile` wraps it in a TLV list alongside
// optional relay hints, which are ignored — the query names the relays it uses.
//
// Exported for two reasons: the catalog has to reject a malformed key before it
// renders one, and the HTTP layer has to tell a key that is simply not a profile
// identifier, which is a bad request, from a relay that cannot be reached, which
// is a bad gateway.
func PubkeyHex(key string) (string, error) {
	hrp, payload, err := bech32Decode(strings.TrimSpace(key))
	if err != nil {
		return "", err
	}

	switch hrp {
	case "npub":
		if len(payload) != 32 {
			return "", fmt.Errorf("npub carries %d bytes, want 32", len(payload))
		}
		return hex.EncodeToString(payload), nil

	case "nprofile":
		pubkey, err := nprofilePubkey(payload)
		if err != nil {
			return "", err
		}
		return hex.EncodeToString(pubkey), nil
	}

	// note1… is an event id and nsec1… is a private key. Neither identifies a
	// profile, and accepting one would query a relay for the wrong thing.
	return "", fmt.Errorf("%q is not a profile identifier", hrp)
}

// nprofilePubkey walks a TLV list for the type-0 entry, which NIP-19 defines as
// the 32-byte pubkey.
func nprofilePubkey(payload []byte) ([]byte, error) {
	for i := 0; i+2 <= len(payload); {
		entryType := payload[i]
		length := int(payload[i+1])
		if i+2+length > len(payload) {
			return nil, fmt.Errorf("nprofile entry is truncated")
		}
		if entryType == 0 {
			if length != 32 {
				return nil, fmt.Errorf("nprofile pubkey is %d bytes, want 32", length)
			}
			return payload[i+2 : i+2+length], nil
		}
		i += 2 + length
	}
	return nil, fmt.Errorf("nprofile has no pubkey entry")
}
