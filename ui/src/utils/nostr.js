// Shared helpers for app-store comments on Nostr.
//
// Comments are plain kind-1 notes that carry two `t` tags — `yantr` and
// `yantr-<app-id>` (for example `yantr-ollama`) — plus a NIP-40 `expiration`
// tag five years out. They live on public relays, never on this host: reading
// and publishing both speak NIP-01 directly from the browser, and signing goes
// through the reader's own NIP-07 signer extension (`window.nostr`), so no key
// material ever touches Yantr.
//
// This file stays pure and browser-independent so it can be unit-tested: the
// WebSocket fan-out lives in `composables/useAppComments.js`.

// NOSTR_COMMENT_RELAYS mirrors the backend's measured relay set
// (core/system/nostr.go): each of these answered a real query during
// development, and together they covered every account tested.
export const NOSTR_COMMENT_RELAYS = [
  "wss://nos.lol",
  "wss://relay.primal.net",
  "wss://nostr.mom",
];

// NOSTR_COMMENT_TAG scopes every comment to Yantr; the per-app tag scopes it
// to one catalog entry.
export const NOSTR_COMMENT_TAG = "yantr";

// COMMENT_KIND is a plain short text note. Kind 1 renders everywhere —
// njump, Primal, Damus — so a comment left here is readable outside Yantr too.
export const COMMENT_KIND = 1;

// COMMENT_EXPIRY_SECONDS is five 365-day years. NIP-40 treats `expiration` as
// advisory and leap days are noise against five years, so a flat constant
// beats calendar math here.
export const COMMENT_EXPIRY_SECONDS = 5 * 365 * 24 * 60 * 60;

// COMMENT_MAX_LENGTH keeps a comment a comment. Relays accept far larger
// notes, but the card renders one line per line break and should not become a
// pastebin.
export const COMMENT_MAX_LENGTH = 2000;

// appCommentTag is the per-app `t` tag for one catalog id. Lowercased on the
// way out: tags are matched exactly by relays, and catalog ids are already
// lowercase (`^[a-z0-9][a-z0-9_-]*$`), so this is normalisation, not policy.
export function appCommentTag(appId) {
  return `${NOSTR_COMMENT_TAG}-${String(appId ?? "").trim().toLowerCase()}`;
}

// commentExpiryTimestamp returns the NIP-40 `expiration` value for an event
// created at `createdAtSec`: five years out.
export function commentExpiryTimestamp(createdAtSec) {
  const created = Number(createdAtSec);
  const base = Number.isFinite(created)
    ? Math.floor(created)
    : Math.floor(Date.now() / 1000);
  return base + COMMENT_EXPIRY_SECONDS;
}

// buildCommentTags assembles the tags for a new comment: the Yantr scope,
// the per-app scope, and the five-year NIP-40 expiry.
export function buildCommentTags(appId, createdAtSec) {
  const created = Number.isFinite(Number(createdAtSec))
    ? Math.floor(Number(createdAtSec))
    : Math.floor(Date.now() / 1000);
  return [
    ["t", NOSTR_COMMENT_TAG],
    ["t", appCommentTag(appId)],
    ["expiration", String(commentExpiryTimestamp(created))],
  ];
}

// commentFilter is the NIP-01 REQ filter that finds one app's comments.
// Relays AND across keys but OR within one key, so "both t tags" cannot be
// expressed in a single filter: query the selective per-app tag and require
// the `yantr` scope client-side in isAppComment.
export function commentFilter(appId, limit = 100) {
  const capped = Math.min(Math.max(Number(limit) || 100, 1), 500);
  return { kinds: [COMMENT_KIND], "#t": [appCommentTag(appId)], limit: capped };
}

// expirationOf returns the numeric NIP-40 expiry of an event, or null when it
// carries none. The first well-formed `expiration` tag wins; NIP-40 defines one.
export function expirationOf(event) {
  if (!event || !Array.isArray(event.tags)) return null;
  for (const tag of event.tags) {
    if (!Array.isArray(tag) || tag[0] !== "expiration") continue;
    const ts = Number(tag[1]);
    if (Number.isFinite(ts)) return Math.floor(ts);
  }
  return null;
}

// hasTag reports whether an event carries ["t", value], compared exactly.
// Tags are relay-matched bytes, so there is no case folding here.
function hasTag(event, value) {
  return event.tags.some(
    (tag) => Array.isArray(tag) && tag[0] === "t" && tag[1] === value
  );
}

const HEX_PUBKEY_RE = /^[0-9a-fA-F]{64}$/;

// isAppComment decides whether a relay-delivered event is a displayable
// comment for one app. Anything unsigned-shaped, wrongly tagged, empty or
// past its NIP-40 expiry is dropped — relays serve what they hold, and what
// they hold includes backfill that predates this feature.
export function isAppComment(event, appId, nowSec = Math.floor(Date.now() / 1000)) {
  if (!event || event.kind !== COMMENT_KIND) return false;
  if (typeof event.pubkey !== "string" || !HEX_PUBKEY_RE.test(event.pubkey)) {
    return false;
  }
  if (!Array.isArray(event.tags)) return false;
  if (!hasTag(event, NOSTR_COMMENT_TAG)) return false;
  if (!hasTag(event, appCommentTag(appId))) return false;

  const content = typeof event.content === "string" ? event.content.trim() : "";
  if (!content) return false;

  const created = Number(event.created_at);
  if (!Number.isFinite(created)) return false;
  // Five minutes of clock skew, not a policy: a signer whose clock runs ahead
  // should not have its comment hidden for it.
  if (created > nowSec + 5 * 60) return false;

  const expiry = expirationOf(event);
  if (expiry !== null && expiry <= nowSec) return false;

  return true;
}

// ─── Content sanitisation ───────────────────────────────────────────────────
// Comment bodies render via Vue interpolation (never v-html), so markup is
// already inert. What interpolation does NOT stop is non-markup abuse from a
// relay that serves arbitrary bytes: C0/C1 control characters, bidi overrides
// that reorder displayed text Trojan-Source style (U+202A–U+202E,
// U+2066–U+2069), zero-width / invisible characters used to spoof one string
// as another, and multi-megabyte notes that would hang the card. Sanitise in
// the data layer — before display AND before signing — so what is shown is
// exactly what was signed, and no code path can render the raw bytes.

// COMMENT_DISPLAY_MAX_LENGTH caps one rendered comment. Publishing enforces
// COMMENT_MAX_LENGTH; relay backfill predates it, so overlong arrivals are
// truncated, never rejected (a long note is still a note).
export const COMMENT_DISPLAY_MAX_LENGTH = COMMENT_MAX_LENGTH;

// Ranges stripped by sanitizeCommentContent: C0/C1 controls except \n and \t
// (DEL kept out too), bidi controls and isolates, zero-width and invisible
// format characters including the soft hyphen, and the BOM — none of which a
// review can need.
const UNSAFE_CONTENT_RE =
  // Stripping controls is the entire job of this class, which is what
  // no-control-regex exists to question.
  // eslint-disable-next-line no-control-regex
  /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F\u00AD\u061C\u180E\u200B-\u200F\u202A-\u202E\u2066-\u2069\uFEFF]/gu;

// sanitizeCommentContent reduces relay- or user-supplied text to displayable
// plain text: normalised, free of controls and spoof characters, collapsed
// blank runs, and length-capped on code points (never splitting a surrogate
// pair). Returns "" when nothing displayable remains, which callers treat as
// "drop the comment".
export function sanitizeCommentContent(raw) {
  let text = String(raw ?? "").normalize("NFC");
  text = text.replace(UNSAFE_CONTENT_RE, "");
  text = text.replace(/\n{3,}/g, "\n\n").trim();
  if (!text) return "";
  const points = [...text];
  if (points.length > COMMENT_DISPLAY_MAX_LENGTH) {
    text = points.slice(0, COMMENT_DISPLAY_MAX_LENGTH - 1).join("").trimEnd() + "…";
  }
  return text;
}

// ─── bech32 (encode side) ───────────────────────────────────────────────────
// The backend (core/nip19) only decodes; displaying a comment author needs the
// other direction, hex → npub, which is small enough to own: 5-to-8 bit
// regroup one way plus a BIP-173 checksum.

const BECH32_CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l";
const BECH32_GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];

function bech32Polymod(values) {
  let chk = 1;
  for (const v of values) {
    const b = chk >> 25;
    chk = ((chk & 0x01ffffff) << 5) ^ v;
    for (let i = 0; i < 5; i++) {
      if ((b >> i) & 1) chk ^= BECH32_GEN[i];
    }
  }
  return chk;
}

function bech32HrpExpand(hrp) {
  const out = [];
  for (let i = 0; i < hrp.length; i++) out.push(hrp.charCodeAt(i) >> 5);
  out.push(0);
  for (let i = 0; i < hrp.length; i++) out.push(hrp.charCodeAt(i) & 31);
  return out;
}

function convertBits8to5(data) {
  let acc = 0;
  let bits = 0;
  const out = [];
  for (const b of data) {
    acc = (acc << 8) | b;
    bits += 8;
    while (bits >= 5) {
      bits -= 5;
      out.push((acc >> bits) & 31);
    }
  }
  if (bits > 0) out.push((acc << (5 - bits)) & 31);
  return out;
}

// hexToNpub renders a 32-byte hex pubkey as the `npub1…` identifier readers
// recognise. Empty string for anything that is not a pubkey — never a link to
// nowhere.
export function hexToNpub(hex) {
  const clean = String(hex ?? "").trim().toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(clean)) return "";
  const bytes = new Uint8Array(32);
  for (let i = 0; i < 32; i++) bytes[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);

  const hrp = "npub";
  const data = convertBits8to5(bytes);
  const mod = bech32Polymod([...bech32HrpExpand(hrp), ...data, 0, 0, 0, 0, 0, 0]) ^ 1;
  const checksum = [];
  for (let p = 0; p < 6; p++) checksum.push((mod >> (5 * (5 - p))) & 31);

  return hrp + "1" + [...data, ...checksum].map((v) => BECH32_CHARSET[v]).join("");
}

// ─── bech32 (decode side) ──────────────────────────────────────────────────
// The frontend needs npub/nprofile → hex to recognise an app's own developer
// among comment authors. A mirror of core/nip19/nip19.go: checksum-verified
// bech32 plus the nprofile TLV walk, no dependency. Tested against the same
// externally-verified vectors as the backend.

function convertBits5to8(data) {
  let acc = 0;
  let bits = 0;
  const out = [];
  for (const v of data) {
    if ((v >> 5) !== 0) return null;
    acc = (acc << 5) | v;
    bits += 5;
    while (bits >= 8) {
      bits -= 8;
      out.push((acc >> bits) & 0xff);
    }
  }
  // Trailing bits that do not compose a whole byte, or a non-zero remainder:
  // the encoding is not canonical.
  if (bits >= 5 || ((acc << (8 - bits)) & 0xff) !== 0) return null;
  return out;
}

function bech32DecodeToPayload(s) {
  const lower = String(s ?? "").toLowerCase();
  const sep = lower.lastIndexOf("1");
  if (sep < 1 || sep + 7 > lower.length) return null;

  const hrp = lower.slice(0, sep);
  for (let i = 0; i < hrp.length; i++) {
    const c = hrp.charCodeAt(i);
    if (c < 33 || c > 126) return null;
  }

  const values = [];
  for (let i = sep + 1; i < lower.length; i++) {
    const idx = BECH32_CHARSET.indexOf(lower[i]);
    if (idx < 0) return null;
    values.push(idx);
  }

  if (bech32Polymod([...bech32HrpExpand(hrp), ...values]) !== 1) return null;

  const payload = convertBits5to8(values.slice(0, -6));
  if (!payload) return null;
  return { hrp, payload };
}

// nprofilePubkey walks a TLV list for the type-0 entry, which NIP-19 defines
// as the 32-byte pubkey. Relay hints riding alongside are ignored — the read
// names its own relays.
function nprofilePubkey(payload) {
  for (let i = 0; i + 2 <= payload.length; ) {
    const type = payload[i];
    const len = payload[i + 1];
    if (i + 2 + len > payload.length) return null;
    if (type === 0) {
      if (len !== 32) return null;
      return payload.slice(i + 2, i + 2 + len);
    }
    i += 2 + len;
  }
  return null;
}

// npubToHex turns an NIP-19 profile identifier into the hex form relay events
// carry in `pubkey`. Empty string for anything that is not one — a note id, a
// private key, a broken checksum — so callers render no badge rather than a
// wrong one.
export function npubToHex(key) {
  const decoded = bech32DecodeToPayload(String(key ?? "").trim());
  if (!decoded) return "";

  if (decoded.hrp === "npub") {
    if (decoded.payload.length !== 32) return "";
    return bytesToHex(decoded.payload);
  }
  if (decoded.hrp === "nprofile") {
    const pubkey = nprofilePubkey(decoded.payload);
    if (!pubkey) return "";
    return bytesToHex(pubkey);
  }
  return "";
}

// ─── Kind-0 author profiles ─────────────────────────────────────────────────
// Comment authors are arbitrary accounts, not catalog developers, so the
// backend profile proxy (catalog keys only, by design) cannot serve them. The
// browser asks the relays directly instead — the same direct path the
// comments themselves already take, and nothing new for the server to proxy.

export const PROFILE_KIND = 0;

// profileFilter fetches kind-0 metadata for a batch of authors in one REQ.
// Deduplicated, hex-validated and capped: a busy thread must not become a
// thousand-author filter.
export function profileFilter(pubkeys, limit = 100) {
  const authors = [
    ...new Set(
      (Array.isArray(pubkeys) ? pubkeys : []).map((p) =>
        String(p ?? "").trim().toLowerCase()
      )
    ),
  ]
    .filter((h) => /^[0-9a-f]{64}$/.test(h))
    .slice(0, 100);
  return {
    kinds: [PROFILE_KIND],
    authors,
    limit: Math.min(Math.max(Number(limit) || 100, 1), 500),
  };
}

// safeHttpUrl keeps only fetchable avatar URLs. A relay can serve any string
// as `picture`, including schemes an <img> must never be handed.
function safeHttpUrl(raw, maxLen = 500) {
  const s = String(raw ?? "").trim();
  if (!s || s.length > maxLen) return "";
  try {
    const u = new URL(s);
    if (u.protocol !== "http:" && u.protocol !== "https:") return "";
    return u.href;
  } catch {
    return "";
  }
}

function shortDisplayText(raw, maxPoints = 80) {
  const points = [...sanitizeCommentContent(raw)];
  if (points.length <= maxPoints) return points.join("");
  return points.slice(0, maxPoints - 1).join("").trimEnd() + "…";
}

// parseKind0Profile maps a kind-0 event onto the display fields a comment
// header reads. Text gets the same sanitiser as comment bodies — a hostile
// name is the same attack through a shorter string — and the picture must be
// http(s) or there is no picture.
export function parseKind0Profile(event) {
  if (!event || event.kind !== PROFILE_KIND) return null;
  const pubkey = String(event.pubkey ?? "").trim().toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(pubkey)) return null;

  let meta = {};
  try {
    const parsed = JSON.parse(event.content);
    if (parsed && typeof parsed === "object") meta = parsed;
  } catch {
    meta = {};
  }

  const created = Number(event.created_at);
  return {
    pubkey,
    name: shortDisplayText(meta.name || meta.display_name || ""),
    about: shortDisplayText(meta.about || "", 160),
    picture: safeHttpUrl(meta.picture),
    nip05: shortDisplayText(meta.nip05 || ""),
    created_at: Number.isFinite(created) ? Math.floor(created) : 0,
  };
}

// ─── NIP-01 event id ────────────────────────────────────────────────────────
// The id is sha256 of the canonical serialisation
// [0, pubkey, created_at, kind, tags, content]. Recomputed client-side as a
// sanity check on relay-delivered events; signing itself is the NIP-07
// extension's job.

function bytesToHex(bytes) {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

async function sha256Hex(text) {
  // WebCrypto is the zero-dependency path in the browser; the synchronous
  // noble fallback in crypto.js is the other. SubtleCrypto needs a secure
  // context, which localhost and https both are.
  if (typeof crypto !== "undefined" && crypto.subtle) {
    const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(text));
    return bytesToHex(new Uint8Array(digest));
  }
  const { sha256 } = await import("@noble/hashes/sha2.js");
  return bytesToHex(sha256(new TextEncoder().encode(text)));
}

// computeEventId returns the NIP-01 id the given fields commit to.
export async function computeEventId({ pubkey, created_at, kind, tags, content }) {
  const canonical = JSON.stringify([0, pubkey, created_at, kind, tags, content]);
  return sha256Hex(canonical);
}

// hasValidId reports whether an event's `id` matches its fields. Best effort:
// a mismatch means relay backfill corruption, not an attack worth blocking the
// whole list over — callers drop the one event, not the fetch.
export async function hasValidId(event) {
  if (!event || typeof event.id !== "string" || !/^[0-9a-f]{64}$/i.test(event.id)) {
    return false;
  }
  try {
    const recomputed = await computeEventId(event);
    return recomputed.toLowerCase() === event.id.toLowerCase();
  } catch {
    return false;
  }
}
