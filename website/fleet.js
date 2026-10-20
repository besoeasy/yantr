// The fleet aggregation, in one place.
//
// The app has a backend that computes this (core/presence/stats.go); the site
// has none, so it keeps a copy. It keeps exactly one: this module is imported
// by website-build.js, which writes the presence.json snapshot at deploy time,
// and by website/index.html, which refreshes live from the same feed in the
// browser. Those used to be two separate copies of the same forty lines, so the
// page could quietly disagree with the file it had been built from.
//
// The country names are a third copy, in the app's Go table
// (core/presence/countries.go). That one is guarded rather than shared, by
// core/presence/website_mirror_test.go: a Go test cannot import a browser and a
// browser cannot run Go.

// How many OSes and apps the site lists, matching the limits Aggregate applies
// in core/presence/stats.go. Countries are not limited on either side.
export const FLEET_LIST_LIMIT = 12;

export const FLEET_COUNTRIES = {
  AD: 'Andorra',
  AE: 'United Arab Emirates',
  AF: 'Afghanistan',
  AL: 'Albania',
  AM: 'Armenia',
  AR: 'Argentina',
  AT: 'Austria',
  AU: 'Australia',
  AZ: 'Azerbaijan',
  BA: 'Bosnia and Herzegovina',
  BD: 'Bangladesh',
  BE: 'Belgium',
  BG: 'Bulgaria',
  BH: 'Bahrain',
  BO: 'Bolivia',
  BR: 'Brazil',
  BY: 'Belarus',
  CA: 'Canada',
  CH: 'Switzerland',
  CL: 'Chile',
  CN: 'China',
  CO: 'Colombia',
  CR: 'Costa Rica',
  CY: 'Cyprus',
  CZ: 'Czechia',
  DE: 'Germany',
  DK: 'Denmark',
  DO: 'Dominican Republic',
  DZ: 'Algeria',
  EC: 'Ecuador',
  EE: 'Estonia',
  EG: 'Egypt',
  ES: 'Spain',
  FI: 'Finland',
  FR: 'France',
  GB: 'United Kingdom',
  GE: 'Georgia',
  GR: 'Greece',
  GT: 'Guatemala',
  HK: 'Hong Kong',
  HR: 'Croatia',
  HU: 'Hungary',
  ID: 'Indonesia',
  IE: 'Ireland',
  IL: 'Israel',
  IN: 'India',
  IQ: 'Iraq',
  IR: 'Iran',
  IS: 'Iceland',
  IT: 'Italy',
  JO: 'Jordan',
  JP: 'Japan',
  KE: 'Kenya',
  KR: 'South Korea',
  KW: 'Kuwait',
  KZ: 'Kazakhstan',
  LB: 'Lebanon',
  LK: 'Sri Lanka',
  LT: 'Lithuania',
  LU: 'Luxembourg',
  LV: 'Latvia',
  MA: 'Morocco',
  MD: 'Moldova',
  ME: 'Montenegro',
  MK: 'North Macedonia',
  MM: 'Myanmar',
  MX: 'Mexico',
  MY: 'Malaysia',
  NG: 'Nigeria',
  NL: 'Netherlands',
  NO: 'Norway',
  NP: 'Nepal',
  NZ: 'New Zealand',
  OM: 'Oman',
  PA: 'Panama',
  PE: 'Peru',
  PH: 'Philippines',
  PK: 'Pakistan',
  PL: 'Poland',
  PR: 'Puerto Rico',
  PT: 'Portugal',
  PY: 'Paraguay',
  QA: 'Qatar',
  RO: 'Romania',
  RS: 'Serbia',
  RU: 'Russia',
  SA: 'Saudi Arabia',
  SE: 'Sweden',
  SG: 'Singapore',
  SI: 'Slovenia',
  SK: 'Slovakia',
  SV: 'El Salvador',
  TH: 'Thailand',
  TN: 'Tunisia',
  TR: 'Turkey',
  TW: 'Taiwan',
  UA: 'Ukraine',
  UG: 'Uganda',
  US: 'United States',
  UY: 'Uruguay',
  UZ: 'Uzbekistan',
  VE: 'Venezuela',
  VN: 'Vietnam',
  ZA: 'South Africa',
  '??': 'Unknown',
};

// Same names as LookupCountry in core/presence/countries.go, and held to that
// table by core/presence/website_mirror_test.go.
//
// A code the table does not cover yet — one the feed has not seen before —
// falls back to the platform's own region names, which the app has no
// equivalent of: the app prints such a code raw while the site names it. That
// asymmetry is deliberate and one-directional, and the table above is what both
// sides share.
export function fleetCountryName(code) {
  const upper = String(code || '').trim().toUpperCase();
  if (!upper) return 'Unknown';
  if (FLEET_COUNTRIES[upper]) return FLEET_COUNTRIES[upper];
  try {
    const name = new Intl.DisplayNames(['en'], { type: 'region' }).of(upper);
    if (name && name !== upper) return name;
  } catch {
    /* no region data — fall through to the code */
  }
  return upper;
}

// Parses an ntfy NDJSON poll into events. Mirrors the scan loop in
// fetchAndAggregate plus ParseEvent in core/presence/stats.go.
export function parseFleetEvents(ndjson) {
  const events = [];
  for (const line of String(ndjson || '').split('\n')) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    let msg;
    try {
      msg = JSON.parse(trimmed);
    } catch {
      continue;
    }
    if (msg.event && msg.event !== 'message') continue;
    let evt;
    try {
      evt = JSON.parse(msg.message);
    } catch {
      continue;
    }
    if (!evt || !evt.event) continue;
    if (!evt.ts && msg.time) evt.ts = msg.time;
    if (!evt.node && msg.id) evt.node = msg.id;
    if (evt.country) evt.country = String(evt.country).trim().toUpperCase();
    events.push(evt);
  }
  return events;
}

function sortedList(counts, limit, toEntry) {
  const out = Object.entries(counts).map(([key, count]) => toEntry(key, count));
  out.sort((a, b) => (b.count - a.count) || String(a.name).localeCompare(String(b.name)));
  return limit > 0 ? out.slice(0, limit) : out;
}

// Mirrors Aggregate in core/presence/stats.go: latest presence wins per node.
//
// appNames maps an app id to the name to show for it; an id it does not cover
// keeps the id as its name. Both callers pass a name map, never catalog
// entries: the website-build snapshot and the page each reduce their catalog to
// one before calling.
export function aggregateFleet(events, appNames) {
  const nodes = new Map();
  for (const evt of events) {
    if (evt.event === 'presence') {
      const id =
        evt.node ||
        `fp:${evt.country || ''}|${evt.os || ''}|${evt.arch || ''}|${evt.cores || 0}|${evt.ram_gb || 0}|${evt.v || ''}`;
      const prev = nodes.get(id);
      if (!prev || (evt.ts || 0) >= prev.ts) nodes.set(id, { evt, ts: evt.ts || 0 });
    }
  }

  const countryCounts = {};
  const osCounts = {};
  const appCounts = {};
  const ramValues = [];
  for (const { evt } of nodes.values()) {
    const country = (evt.country || '??').toUpperCase();
    countryCounts[country] = (countryCounts[country] || 0) + 1;
    const os = evt.os || 'unknown';
    osCounts[os] = (osCounts[os] || 0) + 1;
    if (evt.ram_gb > 0) ramValues.push(evt.ram_gb);
    for (const app of evt.apps || []) {
      if (app) appCounts[app] = (appCounts[app] || 0) + 1;
    }
  }

  const ram = { min: 0, max: 0, avg: 0, unit: 'GB', count: ramValues.length };
  if (ramValues.length > 0) {
    ram.min = Math.min(...ramValues);
    ram.max = Math.max(...ramValues);
    ram.avg = ramValues.reduce((s, v) => s + v, 0) / ramValues.length;
  }

  return {
    nodes: nodes.size,
    events: events.length,
    countries: sortedList(countryCounts, 0, (code, count) => ({
      code,
      name: fleetCountryName(code),
      count,
    })),
    os: sortedList(osCounts, FLEET_LIST_LIMIT, (name, count) => ({ name, count })),
    ram,
    apps: sortedList(appCounts, FLEET_LIST_LIMIT, (id, count) => ({
      id,
      name: (appNames && appNames[id]) || id,
      count,
    })),
    window: 'ntfy cache',
    fetchedAt: new Date().toISOString(),
  };
}

// The shape the Fleet section renders when the feed could not be read at all.
export function emptyFleetStats() {
  return {
    nodes: 0,
    events: 0,
    countries: [],
    os: [],
    ram: { min: 0, max: 0, avg: 0, unit: 'GB', count: 0 },
    apps: [],
    window: 'ntfy cache',
    fetchedAt: new Date().toISOString(),
  };
}
