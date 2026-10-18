// The deterministic daily pick behind the Featured Today card, plus the
// "up next" rotation the full-width card showcases with it.
//
// One date string in, one app out — no randomness, no server choice, so the
// same day reads the same on every host and a reload never reshuffles. The
// seed keeps its old shape on purpose: a date that hashes differently would
// silently move "today's" app on hosts that rebuilt mid-day.

export function getDateDaySeed(now = new Date()) {
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function hashString(value) {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0;
  }
  return hash;
}

function sortedCatalog(apps) {
  return [...(Array.isArray(apps) ? apps : [])].sort((a, b) =>
    String(a?.id ?? "").localeCompare(String(b?.id ?? ""))
  );
}

// appOfDay names the catalog app featured `offsetDays` from now (0 = today).
// Null for an empty catalog, never a crash — the caller hides the card on it.
export function appOfDay(apps, offsetDays = 0, now = new Date()) {
  const catalog = sortedCatalog(apps);
  if (catalog.length === 0) return null;

  const day = new Date(now);
  if (offsetDays) day.setDate(day.getDate() + offsetDays);
  const index = hashString(getDateDaySeed(day)) % catalog.length;
  return catalog[index];
}

// upNextApps lists the apps the card will feature after today — the next
// `count` distinct picks in the rotation, skipping any hash collision with
// today's app or each other. Capped at a week of look-ahead: a catalog as
// small as one app has no "up next", which is the honest answer.
export function upNextApps(apps, featured, count = 2) {
  const seen = new Set();
  if (featured?.id) seen.add(featured.id);

  const out = [];
  for (let i = 1; i <= 7 && out.length < count; i++) {
    const app = appOfDay(apps, i);
    if (app && !seen.has(app.id)) {
      seen.add(app.id);
      out.push(app);
    }
  }
  return out;
}
