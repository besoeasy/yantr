// Shared helpers for the developer credit and the #/developer/<key> route.
//
// A developer is written as a Nostr public key in x-yantr.developers
// (core/apps/catalog.go), and the route takes that key. Keep this file's rules in
// step with developerKey in core/handlers_developers.go: the backend rejects
// anything this accepts only by accident, and vice versa.

// DEVELOPER_LABEL is what to print for a developer in prose. There is only one
// kind now, so it is a constant rather than a lookup that could miss.
export const DEVELOPER_LABEL = "Nostr";

// developerProfileUrl is the account's page. njump resolves any NIP-19
// identifier and hands the reader to whichever client they have, which a bare
// nostr: scheme link cannot do. Empty string for anything that is not a key —
// never a link to nowhere.
export function developerProfileUrl(key) {
  return isNostrKey(key) ? `https://njump.me/${key}` : "";
}

// NOSTR_KEY_RE is a shape check, not a validator. Bech32 verification is the
// backend's job (core/nip19), and a catalog key is guarded by a test there. This
// exists for one narrow purpose: deciding whether to draw a QR code. A code that
// encodes a handle no client can parse looks perfectly fine and does nothing
// when scanned, which is the one failure a reader cannot see.
//
// The alphabet is bech32's: the digits without 1, and the letters without b, i
// and o.
const NOSTR_KEY_RE = /^n(?:pub|profile)1[02-9ac-hj-np-z]+$/i;

// isNostrKey reports whether a value looks like an npub or nprofile.
export function isNostrKey(key) {
  return NOSTR_KEY_RE.test(String(key ?? "").trim());
}

// nostrDmUri is the NIP-21 URI a Nostr client opens to message the developer.
//
// Lowercased on the way out: bech32 is defined lowercase, NIP-19 exchanges it
// lowercase, and the catalog is allowed to keep whatever case it was written
// with. Returns empty string for a value that is not a key, so the caller renders
// no code rather than a code that cannot work.
export function nostrDmUri(key) {
  const clean = String(key ?? "").trim().toLowerCase();
  return isNostrKey(clean) ? `nostr:${clean}` : "";
}

// developerInitials gives an avatar fallback that is stable for one developer
// and never empty. A key has no words in it, so this is derived from the first
// characters of the identifier rather than from a name.
export function developerInitials(key) {
  const clean = String(key ?? "").replace(/[^\p{L}\p{N}]/gu, "");
  return (clean.slice(0, 2) || "?").toUpperCase();
}

// developerShortKey abbreviates a key for a chip or a label. An npub is 63
// characters, which is not something you print next to a name. The ends are what
// a reader uses: the head says what kind of identifier it is, the tail is how
// they check it against one they already have. The full value stays available in
// the element's title, and on the developer page.
export function developerShortKey(key) {
  const clean = String(key ?? "").trim();
  if (clean.length <= 20) return clean;
  return `${clean.slice(0, 11)}…${clean.slice(-4)}`;
}

// appsByDeveloper filters a catalog app list to the ones crediting this key. The
// developer route prefers the server's /api/developers/{key} answer, which does
// the same walk against the live catalog; this is the local fallback for when
// that request is not available.
//
// Compared case-insensitively, matching the catalog: bech32 forbids rewriting a
// key, so two spellings of one account are one account and must not become two.
export function appsByDeveloper(apps, key) {
  const wanted = String(key ?? "").trim().toLowerCase();
  if (!wanted) return [];

  return (Array.isArray(apps) ? apps : []).filter((app) =>
    developersOf(app).some((d) => String(d.key).trim().toLowerCase() === wanted)
  );
}

// developersOf normalises an app's developers array for rendering. The catalog
// serialises it as a non-null array of {key, url}, but an older cached payload or
// a hand-rolled /api/apps response may not have the field at all.
export function developersOf(app) {
  if (!app || !Array.isArray(app.developers)) return [];
  return app.developers.filter((d) => d && d.key);
}

// developerRouteRef is the route segment for one developer: the key itself. There
// is no platform prefix to carry, so the route is just the identifier.
export function developerRouteRef(key) {
  return String(key ?? "").trim();
}

// ISSUE_BASE is the repository every in-app report is filed against. Yantr's
// own issue tracker, not the app's upstream — the catalog recipe is what the
// report is about.
export const ISSUE_BASE = "https://github.com/besoeasy/yantr/issues/new";

// buildIssueReportUrl assembles the pre-filled "report issue" link.
//
// The body names the app and, when the author has published one, their Nostr key
// so a reader can work out whose recipe this is. It used to also @mention a
// GitHub login, which was the only way a filed report could subscribe the person
// who can fix it. There is no equivalent for an npub — a GitHub issue cannot
// mention a Nostr account — so that notification is gone, and the key is
// information rather than a ping. `prefix` is the title tag ("pre-install" for
// the app page, "app" for a running stack); `extraLines` carry context only the
// caller has, such as the stack id.
export function buildIssueReportUrl(app, { prefix = "app", extraLines = [] } = {}) {
  if (!app) return ISSUE_BASE;

  const appName = app.name || app.id || "unknown";

  const lines = [`App: ${appName}${app.id ? ` (${app.id})` : ""}`];
  for (const line of extraLines) {
    if (line) lines.push(line);
  }

  const keys = developersOf(app).map((d) => d.key);
  if (keys.length > 0) lines.push(`Author Nostr key: ${keys.join(" ")}`);

  lines.push("", "<describe the problem>");

  return `${ISSUE_BASE}?title=${encodeURIComponent(`[${prefix}: ${appName}] `)}&body=${encodeURIComponent(lines.join("\n"))}`;
}