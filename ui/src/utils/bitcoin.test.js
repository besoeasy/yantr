import { describe, expect, it } from "vitest";
import {
  bip21Uri,
  bitcoinAddressOf,
  extractBitcoinAddress,
  isBitcoinAddress,
  isLegacyAddress,
  isSegwitAddress,
} from "./bitcoin";

// Valid vectors are the BIP-350 segwit lists (they supersede BIP-173's own
// v1+ entries, which the bech32m redefinition invalidated) plus base58check
// examples, copied from the specs. Invalid vectors are those same lists: a
// wrong checksum, a v1 address carrying a bech32 checksum (the exact string
// BIP-173 once called valid), a v0 address carrying bech32m, a bad witness
// version, short and 41-byte programs, mixed case, foreign prefix. An
// address-shape test would pass all of these wrongly-typed strings; a
// checksum test does not.
const VALID_SEGWIT = [
  "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4", // v0 P2WPKH
  "BC1QW508D6QEJXTDG4Y5R3ZARVARY0C5XW7KV8F3T4", // all-uppercase form
  "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx", // testnet v0 P2WPKH
  "bc1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3qccfmv3", // v0 P2WSH
  "bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqzk5jj0", // v1 taproot (BIP-86)
  "bc1pw508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7kt5nd6y", // v1 bech32m
  "bc1zw508d6qejxtdg4y5r3zarvaryvaxxpcs", // v2
  "BC1SW50QGDZ25J", // v16
];

const INVALID_SEGWIT = [
  "tc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vq5zuyut", // invalid hrp
  "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t5", // invalid checksum
  "bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqh2y7hd", // v1 with a bech32 checksum
  "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kemeawh", // v0 with a bech32m checksum
  "BC13W508D6QEJXTDG4Y5R3ZARVARY0C5XW7KN40WF2", // invalid witness version
  "BC130XLXVLHEMJA6C4DQV22UAPCTQUPFHLXM9H8Z3K2E72Q4K9HCZ7VQ7ZWS8R", // invalid witness version
  "bc1rw5uspcuh", // 1-byte program
  "bc1pw5dgrnzv", // 1-byte program
  "bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7v8n0nx0muaewav253zgeav", // 41-byte program
  "BC1QR508D6QEJXTDG4Y5R3ZARVARYV98GJ9P", // wrong length for v0
  "tb1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vq47Zagq", // mixed case
  "bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7v07qwwzcrf", // padding over 4 bits
  "tb1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vpggkg4j", // non-zero padding
  "bc1gmk9yu", // empty data section
  "ltc1qw508d6qejxtdg4y5r3zarvary0c5xw7kg3g4ty", // not a Bitcoin hrp
];

const VALID_LEGACY = [
  "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2", // mainnet P2PKH
  "3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy", // mainnet P2SH
  "37XUTwntkCvGXezh4QL5ozYHW7w2V4Z57Z", // mainnet P2SH (BIP-16 example hash, generated independently)
  "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn", // testnet P2PKH
  "2My5gXgivMfRcjSdEjXwxRwXYiU9CJggjZ4", // testnet P2SH (same example hash)
];

describe("isSegwitAddress", () => {
  it("accepts every valid vector from the BIP-173/BIP-350 spec", () => {
    for (const address of VALID_SEGWIT) {
      expect(isSegwitAddress(address), address).toBe(true);
    }
  });

  it("rejects every invalid vector from the spec, checksums included", () => {
    for (const address of INVALID_SEGWIT) {
      expect(isSegwitAddress(address), address).toBe(false);
    }
  });

  it("rejects a valid address with one character changed", () => {
    const valid = "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4";
    const broken = `${valid.slice(0, 10)}${valid[10] === "q" ? "p" : "q"}${valid.slice(11)}`;
    expect(isSegwitAddress(broken)).toBe(false);
  });
});

describe("isLegacyAddress", () => {
  it("accepts base58check addresses on mainnet and testnet", () => {
    for (const address of VALID_LEGACY) {
      expect(isLegacyAddress(address), address).toBe(true);
    }
  });

  it("rejects a mistyped legacy address and a foreign version byte", () => {
    expect(isLegacyAddress("1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN3")).toBe(false);
    // Valid base58check, but version 0x00 is mainnet while the leading "D" is
    // a Dogecoin-style prefix this check must not wave through.
    expect(isLegacyAddress("DH5yaieqoZN36fDVciNyRueRGvGLR3mr7L")).toBe(false);
  });
});

describe("isBitcoinAddress", () => {
  it("is the union of both rails, and nothing else", () => {
    expect(isBitcoinAddress(VALID_SEGWIT[0])).toBe(true);
    expect(isBitcoinAddress(VALID_LEGACY[0])).toBe(true);
    expect(isBitcoinAddress("")).toBe(false);
    expect(isBitcoinAddress(null)).toBe(false);
    expect(isBitcoinAddress("besoeasy@wallet.yakihonne.com")).toBe(false);
    expect(isBitcoinAddress(INVALID_SEGWIT[1])).toBe(false);
  });
});

describe("extractBitcoinAddress", () => {
  it("reads a BIP-21 URI first, amount and all, and returns the bare address", () => {
    expect(extractBitcoinAddress("pay me: bitcoin:bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4?amount=0.01")).toBe(
      "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"
    );
    expect(extractBitcoinAddress("bitcoin:1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2?label=me")).toBe(
      "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"
    );
  });

  it("finds a bare address in profile prose", () => {
    expect(extractBitcoinAddress("Donations: 3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy thanks!")).toBe(
      "3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy"
    );
  });

  it("does not offer address-shaped noise as a payable destination", () => {
    // A checksum is what makes the bare scan safe: this is 34 base58-shaped
    // characters and still not an address.
    expect(extractBitcoinAddress("0x1234567890abcdefghijklmnopqrstuvwxyz")).toBe("");
    expect(extractBitcoinAddress("bitcoin:bc1brokenaddresshere")).toBe("");
    expect(extractBitcoinAddress("")).toBe("");
  });
});

describe("bitcoinAddressOf", () => {
  const VALID = "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4";

  it("reads the spellings profiles publish the address under", () => {
    for (const key of ["bitcoin", "btc", "bitcoin_address", "btc_address", "bitcoinAddress", "onchain"]) {
      expect(bitcoinAddressOf({ [key]: VALID }), key).toBe(VALID);
    }
  });

  it("accepts a BIP-21 URI in the metadata field", () => {
    expect(bitcoinAddressOf({ bitcoin: `bitcoin:${VALID}?amount=0` })).toBe(VALID);
  });

  it("falls back to scanning the profile text, which is where most profiles keep it", () => {
    expect(bitcoinAddressOf({ about: `Sats welcome at ${VALID}`, name: "besoeasy" })).toBe(VALID);
    expect(bitcoinAddressOf({ website: `https://example.com/pay/bitcoin:${VALID}` })).toBe(VALID);
  });

  it("returns nothing when there is no address, and never guesses", () => {
    expect(bitcoinAddressOf({})).toBe("");
    expect(bitcoinAddressOf(null)).toBe("");
    expect(bitcoinAddressOf({ bitcoin: "not-an-address", about: "just a bio" })).toBe("");
  });
});

describe("bip21Uri", () => {
  it("wraps a validated address, and refuses anything else", () => {
    expect(bip21Uri(VALID_LEGACY[0])).toBe(`bitcoin:${VALID_LEGACY[0]}`);
    expect(bip21Uri("nope")).toBe("");
    expect(bip21Uri("")).toBe("");
  });
});