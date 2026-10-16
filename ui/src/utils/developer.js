// Shared helpers for the developer credit and the #/developer/<handle> route.
//
// A developer reference is written "platform:handle" in x-yantr.developers
// (core/apps/catalog.go), and the route accepts the bare handle as shorthand
// for GitHub. Keep this file's rules in step with developerRef in
// core/handlers_developers.go: the backend rejects anything this accepts only
// by accident, and vice versa.

// PLATFORMS mirrors validDeveloperPlatforms in core/apps/catalog.go. `api`
// marks the platforms a Yantr server can fetch profile data for; the rest get
// a link out and the catalog app list, which is the whole page for them.
export const DEVELOPER_PLATFORMS = {
  github: {
    label: "GitHub",
    profile: (handle) => `https://github.com/${handle}`,
    api: true,
  },
  gitlab: {
    label: "GitLab",
    profile: (handle) => `https://gitlab.com/${handle}`,
    api: false,
  },
  codeberg: {
    label: "Codeberg",
    profile: (handle) => `https://codeberg.org/${handle}`,
    api: false,
  },
  nostr: {
    // njump routes any NIP-19 identifier without assuming the reader has a
    // particular client, which a bare nostr: link does.
    label: "Nostr",
    profile: (handle) => `https://njump.me/${handle}`,
    // The server proxies kind-0 metadata from one relay for a Nostr developer,
    // cached for a week, the same way it proxies a GitHub profile.
    api: true,
  },
};

// parseDeveloperRef splits "platform:handle" into its parts, defaulting to
// GitHub so `#/developer/besoeasy` works. Returns null when the reference
// cannot be interpreted, which the caller renders as "unknown developer" rather
// than as a GitHub login.
export function parseDeveloperRef(raw) {
  const value = String(raw ?? "").trim();
  if (!value) return null;

  let platform = "github";
  let handle = value;

  const sep = value.indexOf(":");
  if (sep !== -1) {
    platform = value.slice(0, sep).trim().toLowerCase();
    handle = value.slice(sep + 1).trim();
  }

  handle = handle.replace(/^@/, "");
  if (!handle) return null;
  if (!Object.prototype.hasOwnProperty.call(DEVELOPER_PLATFORMS, platform)) return null;

  return { platform, handle };
}

// developerRefString is the canonical "platform:handle" spelling, used for
// route links and for keys when grouping apps by developer.
export function developerRefString(platform, handle) {
  return `${platform}:${handle}`;
}

// developerProfileUrl returns the platform profile URL, or empty string for a
// platform we do not know — never a link to nowhere.
export function developerProfileUrl(platform, handle) {
  const meta = DEVELOPER_PLATFORMS[platform];
  if (!meta) return "";
  return meta.profile(handle);
}

// developerLabel is what to print for a platform in prose.
export function developerLabel(platform) {
  return DEVELOPER_PLATFORMS[platform]?.label ?? platform;
}

// canFetchProfile reports whether a server proxy exists for this platform. The
// page is fully usable without one: the catalog half answers either way.
export function canFetchProfile(platform) {
  return Boolean(DEVELOPER_PLATFORMS[platform]?.api);
}

// NOSTR_KEY_RE is a shape check, not a validator. Bech32 verification belongs to
// the backend (core/system/nip19.go), and a catalog key is guarded by a test
// there. This exists for one narrow purpose: deciding whether to draw a QR code.
// A code that encodes a handle no client can parse looks perfectly fine and does
// nothing when scanned, which is the one failure a reader cannot see.
//
// The alphabet is bech32's: the digits without 1, and the letters without b, i
// and o.
const NOSTR_KEY_RE = /^n(?:pub|profile)1[02-9ac-hj-np-z]+$/i;

// isNostrKey reports whether a handle looks like an npub or nprofile.
export function isNostrKey(handle) {
  return NOSTR_KEY_RE.test(String(handle ?? "").trim());
}

// nostrDmUri is the NIP-21 URI a Nostr client opens to message the developer.
//
// Lowercased on the way out: bech32 is defined lowercase, NIP-19 exchanges it
// lowercase, and the catalog is allowed to keep whatever case it was written
// with. Returns empty string for a handle that does not look like a key, so the
// caller renders no code rather than a code that cannot work.
export function nostrDmUri(handle) {
  const key = String(handle ?? "").trim().toLowerCase();
  return isNostrKey(key) ? `nostr:${key}` : "";
}

// developerInitials gives an avatar fallback that is stable for one developer
// and never empty.
export function developerInitials(handle) {
  const clean = String(handle ?? "").replace(/[^\p{L}\p{N}]/gu, "");
  return (clean.slice(0, 2) || "?").toUpperCase();
}

// appsByDeveloper filters a catalog app list to the ones crediting this
// developer. The developer route prefers the server's /api/developers/{handle}
// answer, which does the same walk against the live catalog; this is the local
// fallback for when that request is not available.
export function appsByDeveloper(apps, platform, handle) {
  if (!Array.isArray(apps)) return [];
  const wanted = platform === "nostr" ? handle : String(handle).toLowerCase();

  return apps.filter((app) =>
    Array.isArray(app?.developers) &&
    app.developers.some((d) => {
      if (d?.platform !== platform) return false;
      const got = d.platform === "nostr" ? d.handle : String(d.handle ?? "").toLowerCase();
      return got === wanted;
    })
  );
}

// developersOf normalises an app's developers array for rendering. The catalog
// serialises it as a non-null array, but an older cached payload or a
// hand-rolled /api/apps response may not have the field at all.
export function developersOf(app) {
  if (!app || !Array.isArray(app.developers)) return [];
  return app.developers.filter((d) => d && d.platform && d.handle);
}

// developerRouteRef is the route segment for one developer. GitHub is the
// default platform, so it gets the short, shareable form — /#/developer/besoeasy
// rather than /#/developer/github:besoeasy.
export function developerRouteRef(platform, handle) {
  return platform === "github" ? String(handle) : developerRefString(platform, handle);
}

// githubMentionsOf returns the @mentions to put in a GitHub issue body. Only
// GitHub is included: an @mention works there and nowhere else, and inventing
// one for another platform captions the link with a handle that does not
// resolve.
export function githubMentionsOf(app) {
  return developersOf(app)
    .filter((d) => d.platform === "github")
    .map((d) => `@${d.handle}`);
}

// ISSUE_BASE is the repository every in-app report is filed against. Yantr's
// own issue tracker, not the app's upstream — the catalog recipe is what the
// report is about.
export const ISSUE_BASE = "https://github.com/besoeasy/yantr/issues/new";

// buildIssueReportUrl assembles the pre-filled "report issue" link.
//
// The body names the app and @mentions its GitHub authors, so the people who can
// actually fix it are subscribed the moment the report is filed rather than
// after someone reads it. `prefix` is the title tag ("pre-install" for the app
// page, "app" for a running stack); `extraLines` carry context that only the
// caller has, such as the stack id.
export function buildIssueReportUrl(app, { prefix = "app", extraLines = [] } = {}) {
  if (!app) return ISSUE_BASE;

  const appName = app.name || app.id || "unknown";

  const lines = [`App: ${appName}${app.id ? ` (${app.id})` : ""}`];
  for (const line of extraLines) {
    if (line) lines.push(line);
  }

  const mentions = githubMentionsOf(app);
  if (mentions.length > 0) lines.push(`Author: ${mentions.join(" ")}`);

  lines.push("", "<describe the problem>");

  return `${ISSUE_BASE}?title=${encodeURIComponent(`[${prefix}: ${appName}] `)}&body=${encodeURIComponent(lines.join("\n"))}`;
}
