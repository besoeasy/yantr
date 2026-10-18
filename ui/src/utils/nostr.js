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
// WebSocket fan-out lives in `utils/relay.js`.

import { verifyEvent } from "nostr-tools";
import { bitcoinAddressOf } from "./bitcoin";

// NOSTR_RELAYS is the fallback relay set, used until the backend answers with
// its own (composables/useNostrRelays.js) and if that request fails. Each relay
// here answered a real query during development, and together they covered
// every account tested. YANTR_NOSTR_RELAYS on the host overrides the backend's
// copy; these three are what an unreachable backend falls back to.
export const NOSTR_RELAYS = [
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

// commentAppIds lists the catalog ids an event's per-app `t` tags name —
// values shaped `yantr-<id>` — in tag order, deduplicated. The developer page
// maps these back onto the catalog to say which app each comment was made on;
// the ids are whatever the author tagged, so the mapper, not this function,
// decides what is unknown there. Bare `yantr` is the scope, not an app.
export function commentAppIds(event) {
  const ids = [];
  if (!event || !Array.isArray(event.tags)) return ids;
  for (const tag of event.tags) {
    if (!Array.isArray(tag) || tag[0] !== "t") continue;
    const value = String(tag[1] ?? "");
    if (!value.startsWith(`${NOSTR_COMMENT_TAG}-`)) continue;
    const id = value.slice(NOSTR_COMMENT_TAG.length + 1).trim().toLowerCase();
    if (id && !ids.includes(id)) ids.push(id);
  }
  return ids;
}

// authorCommentFilter is the NIP-01 REQ that finds one account's Yantr
// comments — the author-shaped twin of commentFilter, which is app-shaped.
// Relays AND across keys, so "comments by this account under any per-app tag"
// cannot name the app tag on the relay: query the bare `yantr` scope with
// authorship pinned, and let isAuthorYantrComment require a per-app tag
// client-side.
//
// Null for a value that is not a valid pubkey, so a caller never sends an
// author-less filter — an empty `authors` array matches every author, which
// would turn a broken key into a firehose.
export function authorCommentFilter(pubkey, limit = 100) {
  const author = String(pubkey ?? "").trim().toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(author)) return null;
  const capped = Math.min(Math.max(Number(limit) || 100, 1), 500);
  return {
    kinds: [COMMENT_KIND],
    authors: [author],
    "#t": [NOSTR_COMMENT_TAG],
    limit: capped,
  };
}

// isYantrComment is the author-agnostic core of both comment checks: a kind-1
// note scoped to `yantr` that names at least one per-app tag, with displayable
// content, a sane timestamp and an unexpired NIP-40 lifetime. The checks are
// shared with isAppComment on purpose: the app-pinned check (a thread) and
// the feed-wide check (a showcase) must never diverge on what counts as a
// comment, or the two lists would disagree about the same note.
export function isYantrComment(event, nowSec = Math.floor(Date.now() / 1000)) {
  if (!event || event.kind !== COMMENT_KIND) return false;
  if (typeof event.pubkey !== "string" || !/^[0-9a-fA-F]{64}$/.test(event.pubkey)) {
    return false;
  }
  if (!Array.isArray(event.tags)) return false;

  const perApp = commentAppIds(event);
  if (perApp.length === 0) return false;
  let scopesToYantr = false;
  for (const tag of event.tags) {
    if (Array.isArray(tag) && tag[0] === "t" && tag[1] === NOSTR_COMMENT_TAG) {
      scopesToYantr = true;
      break;
    }
  }
  if (!scopesToYantr) return false;

  const content = typeof event.content === "string" ? event.content.trim() : "";
  if (!content) return false;

  const created = Number(event.created_at);
  if (!Number.isFinite(created)) return false;
  if (created > nowSec + 5 * 60) return false;

  const expiry = expirationOf(event);
  if (expiry !== null && expiry <= nowSec) return false;

  return true;
}

// yantrFeedFilter is the feed-wide REQ: kind-1 notes under the bare `yantr`
// scope, any author. Necessarily broad — the per-app tag is where authorship
// and app pinning are applied client-side (isYantrComment / commentAppIds),
// because relays AND across keys and no single filter can say "any of 30
// per-app tags".
export function yantrFeedFilter(limit = 200) {
  const raw = Number(limit);
  // Non-numeric and non-positive limits fall back to the default rather than
  // clamping to 1 — a typo should not turn a feed read into a one-note read.
  const capped = Math.min(Math.max(Number.isFinite(raw) && raw > 0 ? Math.floor(raw) : 200, 1), 500);
  return {
    kinds: [COMMENT_KIND],
    "#t": [NOSTR_COMMENT_TAG],
    limit: capped,
  };
}

// isAuthorYantrComment decides whether a relay-delivered event is a displayable
// Yantr comment by one account — what the developer page means by "this
// author's app-store activity". Every check mirrors isAppComment's, with two
// differences: the event must be signed by the account the page is about, and
// any per-app tag passes — the page aggregates across the catalog, so the app
// requirement is "names at least one", not "names this one". A note under the
// bare `yantr` scope alone is not an app comment and stays out.
export function isAuthorYantrComment(
  event,
  pubkey,
  nowSec = Math.floor(Date.now() / 1000)
) {
  const author = String(pubkey ?? "").trim().toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(author)) return false;
  if (String(event?.pubkey ?? "").trim().toLowerCase() !== author) return false;

  return isYantrComment(event, nowSec);
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

// ─── bech32 ────────────────────────────────────────────────────────────────
// Both directions are here, hand-rolled, because the frontend needs them and
// they are ~80 lines with no state and no I/O. Encoding (hex → npub) has no
// backend counterpart: core/nip19 only decodes, yet displaying a comment
// author needs the identifier readers recognise.

function bytesToHex(bytes) {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

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

// bech32Encode is this file's encode side: 8-bit bytes in, a full bech32
// string out with the BIP-173 checksum. Until now only npub needed encoding
// (below); LNURL buys its piece of this code — LNURL is bech32 too, just of a
// pay URL (lnurlEncode) — so the projection lives here once, correct for
// every HRP.
export function bech32Encode(hrp, bytes) {
  const data = convertBits8to5(Array.isArray(bytes) ? bytes : [...(bytes ?? [])]);
  const mod = bech32Polymod([...bech32HrpExpand(hrp), ...data, 0, 0, 0, 0, 0, 0]) ^ 1;
  const checksum = [];
  for (let p = 0; p < 6; p++) checksum.push((mod >> (5 * (5 - p))) & 31);
  return hrp + "1" + [...data, ...checksum].map((v) => BECH32_CHARSET[v]).join("");
}

// hexToNpub renders a 32-byte hex pubkey as the `npub1…` identifier readers
// recognise. Empty string for anything that is not a pubkey — never a link to
// nowhere.
export function hexToNpub(hex) {
  const clean = String(hex ?? "").trim().toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(clean)) return "";
  const bytes = new Uint8Array(32);
  for (let i = 0; i < 32; i++) bytes[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);

  return bech32Encode("npub", bytes);
}

// ─── Lightning (lud16 → LNURL) ──────────────────────────────────────────────
// A Nostr profile can carry a NIP-57 Lightning address in lud16 ("user@host").
// The payload wallets scan is not the address itself but its LNURL-Pay URL,
// https://<host>/.well-known/lnurlp/<user>, bech32 encoded. Encoded here, not
// fetched: the browser asks no one — the wallet that scans the code contacts
// the domain itself, which is the point of LNURL-Pay and what keeps Yantr out
// of the donation path entirely.

// No full RFC 5322 quoted-string algebra: NIP-57 practice (and what every
// wallet in the wild publishes) is the simple form. Reject anything a relay
// could abuse as a path — no slashes, no query syntax, no whitespace.
const LUD16_RE = /^([a-z0-9._%+-]+)@([a-z0-9-]+(?:\.[a-z0-9-]+)+)$/;

// isLightningAddress answers "is this a Lightning address Yantr can turn into
// a pay-URL", the validation lnurlEncode already does — lifted out so the
// kind-0 parser can decide whether to publish an address at all, instead of
// letting the widget discover the truth from an empty LNURL. lud06 is the
// same `user@domain` shape under LNURL-based resolution, so one rule covers
// both spellings NIP-57 allows.
export function isLightningAddress(value) {
  const clean = String(value ?? "").trim().toLowerCase();
  return LUD16_RE.test(clean) ? clean : "";
}

// lnurlEncode turns a lud16 into the `lnurl1…` string for the zap QR. Empty
// string for anything that is not an address — a QR that encodes prose scans
// into a dead wallet lookup, which is worse than no QR.
export function lnurlEncode(lud) {
  const match = LUD16_RE.exec(String(lud ?? "").trim().toLowerCase());
  if (!match) return "";
  const [, user, domain] = match;
  const url = `https://${domain}/.well-known/lnurlp/${encodeURIComponent(user)}`;
  return bech32Encode("lnurl", new TextEncoder().encode(url));
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
// header reads, and the payment fields the developer page shows. Text gets the
// same sanitiser as comment bodies — a hostile name is the same attack through
// a shorter string — and the picture must be http(s) or there is no picture.
//
// The payment fields are what a donation surface depends on, and both are
// read from the verified event only: lud16 (falling back to the lud06
// spelling) for Lightning, and an on-chain Bitcoin address for a native
// send. Both are validated at this boundary — an unvalidated value would put
// a QR code in front of a wallet that pays whatever string was published.
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
    lud16: isLightningAddress(meta.lud16) || isLightningAddress(meta.lud06),
    bitcoin: bitcoinAddressOf(meta),
    created_at: Number.isFinite(created) ? Math.floor(created) : 0,
  };
}

// ─── Authenticity ──────────────────────────────────────────────────────────
// An event's id is a hash of its own fields, which proves only that the event
// is internally consistent. A hostile relay can forge that freely: keep the
// victim's pubkey, rewrite the content, recompute the id. Only the schnorr
// signature over the id stops impersonation, so it is verified here with
// nostr-tools' verifyEvent, which checks the id and the signature together.
//
// The check runs on the RAW event, before display sanitising — the signature
// covers exactly the bytes the author signed. It is synchronous (nostr-tools
// bundles its own schnorr), so unlike the WebCrypto path it replaced it is
// never skipped for want of a secure context.
//
// One subtlety worth keeping: the event is rebuilt field-by-field rather than
// spread. nostr-tools caches a verification verdict on the event under a
// symbol, and a spread would copy that verdict onto an object whose contents
// have since changed — an attacker-supplied "already verified" flag.
export function isAuthentic(event) {
  try {
    if (!event || typeof event.id !== "string" || typeof event.sig !== "string") {
      return false;
    }
    if (!/^[0-9a-f]{64}$/i.test(event.id)) return false;
    if (!/^[0-9a-f]{128}$/i.test(event.sig)) return false;
    if (!/^[0-9a-f]{64}$/i.test(String(event.pubkey ?? ""))) return false;

    return (
      verifyEvent({
        id: event.id,
        pubkey: event.pubkey,
        created_at: event.created_at,
        kind: event.kind,
        tags: Array.isArray(event.tags) ? event.tags : [],
        content: typeof event.content === "string" ? event.content : "",
        sig: event.sig,
      }) === true
    );
  } catch {
    return false;
  }
}

// ─── Kind-0 merge ──────────────────────────────────────────────────────────
// newestKind0Profiles reduces relay answer sets to one profile per author,
// newest event winning: kind-0 replaces rather than chains, so the newest
// event an account published is the profile it means.
//
// Every candidate is verified first. A kind-0 is the closest thing Nostr has
// to a profile card, so an unverified one is not a cosmetic risk like a forged
// comment author: it is a relay impersonating the account whose name, picture
// and NIP-05 the UI is about to print. Verification happens before parsing, so
// the signature covers the bytes the account actually signed.
//
// Unverified events are dropped silently: one bad relay answer is not an error
// worth failing a page over, and another relay may hold the real one.
export function newestKind0Profiles(events) {
  const newest = {};
  for (const event of Array.isArray(events) ? events : []) {
    if (!isAuthentic(event)) continue;
    const profile = parseKind0Profile(event);
    if (!profile) continue;
    const seen = newest[profile.pubkey];
    if (!seen || profile.created_at > seen.created_at) {
      newest[profile.pubkey] = profile;
    }
  }
  return newest;
}
