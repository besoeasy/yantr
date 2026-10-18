import { sha256 as nobleSha256 } from "@noble/hashes/sha2.js";

// Bitcoin address detection for the Developer of the day card.
//
// Two payment rails reach a Nostr profile, and the widget has to recognise
// both before it prints either: Lightning (a NIP-57 lud16/lud06, which
// becomes an `lnurl1…` pay-URL — see lnurlEncode in utils/nostr.js) and
// on-chain (a Bitcoin address, which becomes a BIP-21 `bitcoin:…` URI).
//
// Checksums are verified, not just shapes. An address is not something to
// print from a guess: a QR code that scans into a mistyped address is a
// payment sent to nobody, so anything that fails its own checksum is
// treated as no address at all — the same way an unverifiable event is no
// event. The vectors below are the BIP-173/BIP-350 spec ones, not
// hand-typed guesses.

const BECH32_ALPHABET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l";
const BECH32M_CONST = 0x2bc830a3;
const BECH32_CONST = 1;

// Segwit human-readable parts: mainnet, testnet, regtest. The network a
// profile publishes is the profile's business — a testnet address shown as
// a payable QR is still what the author asked for — but anything else
// (litecoin's ltc1, a typo'd prefix) is not a Bitcoin address at all.
const SEGWIT_HRPS = new Set(["bc", "tb", "bcrt"]);

const BASE58_ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

// Base58 version bytes: mainnet P2PKH / P2SH, testnet P2PKH / P2SH. Any
// other version byte is a different coin wearing a Bitcoin address.
const BASE58_VERSIONS = new Set([0x00, 0x05, 0x6f, 0xc4]);

function polymod(values) {
  const GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];
  let chk = 1;
  for (const value of values) {
    const top = chk >> 25;
    chk = ((chk & 0x1ffffff) << 5) ^ value;
    for (let i = 0; i < 5; i++) {
      if ((top >> i) & 1) chk ^= GEN[i];
    }
  }
  return chk >>> 0;
}

function hrpExpand(hrp) {
  const out = [];
  for (const ch of hrp) out.push(ch.codePointAt(0) >> 5);
  out.push(0);
  for (const ch of hrp) out.push(ch.codePointAt(0) & 31);
  return out;
}

function base58Decode(text) {
  let num = 0n;
  for (const ch of text) {
    const digit = BASE58_ALPHABET.indexOf(ch);
    if (digit < 0) return null;
    num = num * 58n + BigInt(digit);
  }
  const bytes = [];
  while (num > 0n) {
    bytes.unshift(Number(num & 0xffn));
    num >>= 8n;
  }
  for (const ch of text) {
    if (ch !== "1") break;
    bytes.unshift(0);
  }
  return Uint8Array.from(bytes);
}

// isSegwitAddress validates a bech32/bech32m segwit address against BIP-173,
// with BIP-350's bech32m for witness v1+ and BIP-141's program-length rules.
export function isSegwitAddress(raw) {
  const text = String(raw ?? "").trim();
  if (text.length < 14 || text.length > 74) return false;

  // BIP-173: mixed case is invalid outright; all-uppercase is the same
  // address as all-lowercase.
  const hasLower = /[a-z]/.test(text);
  const hasUpper = /[A-Z]/.test(text);
  if (hasLower && hasUpper) return false;
  const value = text.toLowerCase();

  const separator = value.lastIndexOf("1");
  if (separator < 1 || separator + 7 > value.length) return false;

  const hrp = value.slice(0, separator);
  if (!SEGWIT_HRPS.has(hrp)) return false;

  const data = [];
  for (const ch of value.slice(separator + 1)) {
    const digit = BECH32_ALPHABET.indexOf(ch);
    if (digit < 0) return false;
    data.push(digit);
  }

  // Witness version 0 uses bech32, everything above it uses bech32m.
  const expected = data[0] === 0 ? BECH32_CONST : BECH32M_CONST;
  if (polymod([...hrpExpand(hrp), ...data]) !== expected) return false;
  if (data[0] > 16) return false;

  // Witness program: 2 to 40 bytes, and BIP-141's exact rule for v0. The
  // data part is 5 bits per character, so the byte count is the bit count
  // rounded down — and the leftover bits must be zero padding, no more than
  // four of them (BIP-173 rejects "padding of more than 4 bits").
  const program = data.slice(1, -6);
  const bits = program.length * 5;
  const padding = bits % 8;
  if (padding > 4) return false;
  if (padding > 0) {
    const lastDigit = program[program.length - 1];
    if (lastDigit & ((1 << padding) - 1)) return false;
  }
  const programLength = Math.floor(bits / 8);
  if (programLength < 2 || programLength > 40) return false;
  if (data[0] === 0 && programLength !== 20 && programLength !== 32) return false;
  return true;
}

// isLegacyAddress validates a base58check address (P2PKH / P2SH, mainnet or
// testnet): decode, verify the double-SHA256 checksum, check the version byte.
export function isLegacyAddress(raw) {
  const text = String(raw ?? "").trim();
  if (text.length < 26 || text.length > 35) return false;
  if (/[IlO0]/.test(text)) return false;

  const bytes = base58Decode(text);
  if (!bytes || bytes.length !== 25) return false;
  // 1 version byte + 20 hash bytes + 4 checksum bytes.
  if (!BASE58_VERSIONS.has(bytes[0])) return false;

  const payload = bytes.slice(0, -4);
  const checksum = bytes.slice(-4);
  const first = nobleSha256(nobleSha256(payload)).slice(0, 4);
  return first.every((byte, i) => byte === checksum[i]);
}

// isBitcoinAddress answers "would a wallet recognise this as payable".
export function isBitcoinAddress(value) {
  const text = String(value ?? "").trim();
  if (!text) return false;
  return isSegwitAddress(text) || isLegacyAddress(text);
}

// extractBitcoinAddress finds an address in free text the way clients that
// show an on-chain button do: a BIP-21 `bitcoin:ADDR?…` URI wins outright,
// otherwise the first token in the text that validates as an address. The
// checksum is what makes a bare scan safe — without it, any 34-character
// base58-looking word in a bio would be offered as a payable destination.
export function extractBitcoinAddress(text) {
  const raw = String(text ?? "");
  if (!raw) return "";

  const uri = raw.match(/bitcoin:([A-Za-z0-9]+)/i);
  if (uri && isBitcoinAddress(uri[1])) return uri[1];

  const tokens = raw.match(/[13mn2][1-9A-HJ-NP-Za-km-z]{25,34}|(?:bc|tb|bcrt)1[023456789acdefghjklmnpqrstuvwxyz]{11,71}/gi) || [];
  for (const token of tokens) {
    if (isBitcoinAddress(token)) return token;
  }
  return "";
}

// Metadata keys a kind-0 may carry an on-chain address under. There is no
// standard for this — NIP-57 covers Lightning only — so the common spellings
// are read, in the order clients write them, and the profile text is scanned
// as the fallback that actually catches most profiles.
const BITCOIN_META_KEYS = [
  "bitcoin",
  "btc",
  "bitcoin_address",
  "btc_address",
  "bitcoinAddress",
  "btcAddress",
  "onchain",
  "onchain_address",
  "onchainAddress",
];

const SCANNED_META_FIELDS = ["about", "website", "display_name", "name", "nickname", "pronouns"];

// bitcoinAddressOf reads an on-chain address out of a kind-0 metadata object.
export function bitcoinAddressOf(meta) {
  if (!meta || typeof meta !== "object") return "";

  for (const key of BITCOIN_META_KEYS) {
    const candidate = meta[key];
    if (typeof candidate !== "string") continue;
    if (isBitcoinAddress(candidate.trim())) return candidate.trim();
    const found = extractBitcoinAddress(candidate);
    if (found) return found;
  }

  for (const field of SCANNED_META_FIELDS) {
    const found = extractBitcoinAddress(meta[field]);
    if (found) return found;
  }

  return "";
}

// bip21Uri is the on-chain QR payload: the bare address, no amount, so the
// wallet asks the sender for the sum. (BIP-21 allows uppercasing bech32 for
// QR compactness; wallets accept either, and lowercase is what people compare
// against a published address, so that is what is printed here too.)
export function bip21Uri(address) {
  const value = String(address ?? "").trim();
  return isBitcoinAddress(value) ? `bitcoin:${value}` : "";
}