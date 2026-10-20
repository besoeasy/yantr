import fs from 'fs';
import https from 'https';
import path from 'path';
import nunjucks from 'nunjucks';
import { parse } from 'yaml';
import { fileURLToPath } from 'url';
import { parseFleetEvents, aggregateFleet, emptyFleetStats } from './website/fleet.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const websiteDir = path.join(__dirname, 'website');
const templatesDir = path.join(websiteDir, 'templates');
// The hero one-liner is `curl -fsSL https://yantr.org/install.sh | bash`. Serving
// the installer from the site's own domain keeps the command ~45 characters, which
// fits one line in the hero install box even on phones, where the full jsdelivr
// URL truncated past the visible area.
const installerOutputPath = path.join(websiteDir, 'install.sh');
const appsJsonPath = path.join(websiteDir, 'apps.json');
const appsOutputDir = path.join(websiteDir, 'apps');
const appsDir = path.join(__dirname, 'apps');
const siteUrl = 'https://yantr.org';

const DISPLAY_PROTOCOLS = new Set(['HTTP', 'HTTPS', 'TCP', 'UDP']);

// Mirrors parseDevelopers in core/apps/catalog.go: entries are Nostr public keys,
// deduplicated case-insensitively, with anything that is not one dropped rather
// than rendered as a dead link.
//
// The shape check is not the checksum. core/nip19 verifies bech32 and the Go
// catalog drops what fails before it ever serialises, so by the time this runs
// every entry already passed. Re-verifying here would mean carrying a bech32
// decoder in the site builder to guard a page whose worst outcome is one dead
// njump link — and njump does not notify anyone the way a wrong @mention did.
const NOSTR_KEY_RE = /^n(?:pub|profile)1[02-9ac-hj-np-z]+$/i;

function parseDevelopers(inList) {
  const out = [];
  const seen = new Set();
  if (!Array.isArray(inList)) return out;

  for (const raw of inList) {
    const text = String(raw ?? '').trim();
    if (!NOSTR_KEY_RE.test(text)) continue;

    // bech32 forbids rewriting a key, so the catalog may hold one in any case
    // while the page path is lowercased to keep URLs tidy.
    const folded = text.toLowerCase();
    if (seen.has(folded)) continue;
    seen.add(folded);

    out.push({
      key: text,
      // 63 characters is not a page title.
      shortKey: text.length <= 20 ? text : `${text.slice(0, 11)}…${text.slice(-4)}`,
      // njump resolves any NIP-19 identifier and hands the reader to whichever
      // client they have, which a bare nostr: link cannot do.
      url: `https://njump.me/${text}`,
      platformLabel: 'Nostr',
      initials: (text.replace(/[^\p{L}\p{N}]/gu, '').slice(0, 2) || '?').toUpperCase(),
      ref: folded,
    });
  }
  return out;
}

// developerPagePath is the site path for one developer's page. Kept in one place
// so the app-page credit and the generated page cannot disagree.
function developerPagePath(ref) {
  return `/developer/${ref}/`;
}

function parseXyPorts(metaPorts, services) {
  const out = [];
  const seen = new Set();
  if (!Array.isArray(metaPorts)) return out;
  const serviceNames = new Set(Object.keys(services ?? {}));
  for (const p of metaPorts) {
    const portNum = Number(p?.port);
    if (!Number.isInteger(portNum) || portNum < 1 || portNum > 65535) continue;
    const protocol = String(p?.protocol ?? 'TCP').trim().toUpperCase();
    if (!DISPLAY_PROTOCOLS.has(protocol)) continue;
    const label = String(p?.label ?? '').trim() || `Port ${portNum}`;
    const service = String(p?.service ?? '').trim();
    // Drop bogus entries referencing services that don't exist (the old
    // label system hid these via port-only dedup).
    if (service && !serviceNames.has(service)) continue;
    const key = `${portNum}/${protocol}/${service}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ port: portNum, protocol, label, service });
  }
  out.sort((a, b) => a.port - b.port || (a.protocol < b.protocol ? -1 : 1) || (a.service < b.service ? -1 : 1));
  return out;
}

function parseAppFolder(appId, appPath) {
  const composePath = path.join(appPath, 'compose.yml');

  try {
    if (!fs.existsSync(composePath)) {
      console.warn(`⚠️  No compose.yml found for ${appId}`);
      return null;
    }

    const composeData = parse(fs.readFileSync(composePath, 'utf8'));
    const meta = composeData?.['x-yantr'];

    if (!meta?.name) {
      console.warn(`⚠️  No x-yantr.name in compose.yml for ${appId}`);
      return null;
    }

    const services = composeData?.services ?? {};
    const serviceName = Object.keys(services)[0] || null;
    const image = serviceName ? (services[serviceName]?.image || null) : null;
    const ports = parseXyPorts(meta?.ports, services);
    const developers = parseDevelopers(meta?.developers);
    const hasLogo = fs.existsSync(path.join(appPath, 'logo.svg'));

    return {
      id: appId,
      name: meta.name,
      tags: Array.isArray(meta.tags) ? meta.tags : [],
      ports,
      developers,
      short_description: meta.short_description || '',
      description: meta.description || meta.short_description || '',
      usecases: Array.isArray(meta.usecases) ? meta.usecases : [],
      website: meta.website || null,
      notes: Array.isArray(meta.notes) ? meta.notes : [],
      image,
      serviceName,
      hasLogo,
    };
  } catch (error) {
    console.error(`❌ Error parsing ${appId}/compose.yml:`, error.message);
    return null;
  }
}

function buildAppsJson() {
  console.log('🔨 Building apps.json from apps folder...\n');

  if (!fs.existsSync(websiteDir)) {
    fs.mkdirSync(websiteDir, { recursive: true });
    console.log(`✅ Created directory: ${websiteDir}\n`);
  }

  const appDirs = fs
    .readdirSync(appsDir, { withFileTypes: true })
    .filter((dirent) => dirent.isDirectory())
    .map((dirent) => dirent.name)
    .filter((name) => name !== 'node_modules');

  const apps = [];
  const stats = {
    total: appDirs.length,
    success: 0,
    skipped: 0,
    failed: 0,
  };

  for (const appId of appDirs.sort()) {
    const appPath = path.join(appsDir, appId);
    const composePath = path.join(appPath, 'compose.yml');

    if (!fs.existsSync(composePath)) {
      console.warn(`⚠️  Skipping ${appId}: compose.yml not found`);
      stats.skipped++;
      continue;
    }

    const appData = parseAppFolder(appId, appPath);
    if (appData) {
      apps.push(appData);
      console.log(`✅ ${appData.name} (${appId})`);
      stats.success++;
    } else {
      stats.failed++;
    }
  }

  const output = {
    meta: {
      generatedAt: new Date().toISOString(),
      totalApps: apps.length,
      tags: [...new Set(apps.flatMap((app) => app.tags))].sort(),
    },
    apps,
  };

  fs.writeFileSync(appsJsonPath, JSON.stringify(output, null, 2), 'utf8');

  console.log('\n' + '='.repeat(60));
  console.log('📊 Summary:');
  console.log(`   Total directories: ${stats.total}`);
  console.log(`   ✅ Successfully processed: ${stats.success}`);
  console.log(`   ⚠️  Skipped: ${stats.skipped}`);
  console.log(`   ❌ Failed: ${stats.failed}`);
  console.log('='.repeat(60));
  console.log('\n✨ apps.json generated successfully!');
  console.log(`📁 Output: ${appsJsonPath}`);
  console.log(`📦 Total apps: ${apps.length}`);
  console.log(`🏷️  Tags: ${output.meta.tags.length}`);

  // `curl -fsSL https://yantr.org/install.sh | bash` — the hero one-liner needs
  // the installer served by the same Pages site. Copy the root script in so the
  // served copy can never drift from the repo one.
  fs.copyFileSync(path.join(__dirname, 'install.sh'), installerOutputPath);
}

function fallbackLogoUrl(app) {
  return `https://ui-avatars.com/api/?name=${encodeURIComponent(app?.name || app?.id || 'App')}&background=random&color=fff&bold=true`;
}

function getLogoUrl(app) {
  if (app?.hasLogo) {
    return `/apps/${app.id}/logo.svg`;
  }
  return fallbackLogoUrl(app);
}

function toAppViewModel(app) {
  const id = app?.id || 'unknown-app';
  const name = app?.name || id;
  const tags = Array.isArray(app?.tags) ? app.tags : [];
  const primaryTag = tags[0] || 'self-hosted';
  const summary = app?.short_description || app?.description || 'No description available.';
  const appUrl = `${siteUrl}/apps/${id}/`;
  const ports = Array.isArray(app?.ports) ? app.ports : [];
  const primaryPort =
    ports.find((port) => port?.protocol === 'HTTP' || port?.protocol === 'HTTPS') ||
    ports[0] ||
    null;

  return {
    ...app,
    id,
    name,
    tags,
    ports,
    primaryPort,
    notes: Array.isArray(app?.notes) ? app.notes : [],
    developers: Array.isArray(app?.developers) ? app.developers : [],
    hasDevelopers: Array.isArray(app?.developers) && app.developers.length > 0,
    short_description: app?.short_description || '',
    description: app?.description || app?.short_description || 'No description available.',
    usecases: Array.isArray(app?.usecases) ? app.usecases : [],
    hasLogo: Boolean(app?.hasLogo),
    logoUrl: getLogoUrl(app),
    summary,
    primaryTag,
    appUrl,
    appPagePath: `/apps/${id}/`,
    sourceComposeUrl: `https://github.com/besoeasy/yantr/blob/main/apps/${id}/compose.yml`,
    sourceAppFolderUrl: `https://github.com/besoeasy/yantr/tree/main/apps/${id}`,
    appSearchIntentTitle: `${name} Podman Compose Setup`,
    appSearchIntentDescription: `Learn how to self-host ${name} with Podman using Yantr. ${summary}`,
  };
}

function getRelatedApps(app, allApps, count = 3) {
  const others = allApps.filter(
    (a) => a.id !== app.id && a.tags.some((t) => app.tags.includes(t))
  );
  for (let i = others.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [others[i], others[j]] = [others[j], others[i]];
  }
  return others.slice(0, count);
}

function buildPages() {
  buildAppsJson();

  const env = nunjucks.configure(templatesDir, {
    autoescape: true,
    noCache: true,
  });

  const content = fs.readFileSync(appsJsonPath, 'utf8');
  const parsed = JSON.parse(content);
  const apps = (Array.isArray(parsed?.apps) ? parsed.apps : []).map(toAppViewModel);
  const generatedAt = new Date().toISOString();

  fs.rmSync(appsOutputDir, { recursive: true, force: true });
  fs.mkdirSync(appsOutputDir, { recursive: true });

  for (const app of apps) {
    const appDir = path.join(appsOutputDir, app.id);
    fs.mkdirSync(appDir, { recursive: true });

    if (app.hasLogo) {
      const src = path.join(appsDir, app.id, 'logo.svg');
      const dest = path.join(appDir, 'logo.svg');
      fs.copyFileSync(src, dest);
    }

    const relatedApps = getRelatedApps(app, apps);

    const pageDescription = app.short_description
      ? `Self-host ${app.name} with Podman. ${app.short_description} Deploy it in seconds with Yantr.`
      : `Learn how to self-host ${app.name} on your homelab using Podman. ${app.description}${app.description.endsWith('.') ? '' : '.'} Easy one-click setup with Yantr.`;

    const html = env.render('app.njk', {
      app,
      relatedApps,
      nowIso: generatedAt,
      pageTitle: `Self-Host ${app.name} with Podman | Yantr`,
      pageDescription,
      pageUrl: app.appUrl,
      imageUrl: app.hasLogo ? `${siteUrl}${app.logoUrl}` : app.logoUrl,
    });

    fs.writeFileSync(path.join(appDir, 'index.html'), html, 'utf8');
  }

  console.log(`✨ Generated ${apps.length} app pages in ${appsOutputDir}`);

  buildDeveloperPages(env, apps, generatedAt);
}

// buildDeveloperPages writes one page per credited developer, listing the apps
// they maintain in the catalog.
//
// No GitHub call happens here on purpose. The upstream contributors fetch is
// already the only network dependency in this build, and a developer page that
// needs a second one would fail the whole build whenever api.github.com is slow
// or rate-limited. The credit, the profile link and the app list are all in the
// repository already.
function buildDeveloperPages(env, apps, generatedAt) {
  const byRef = new Map();

  for (const app of apps) {
    for (const dev of app.developers || []) {
      if (!byRef.has(dev.ref)) {
        byRef.set(dev.ref, { ...dev, apps: [] });
      }
      byRef.get(dev.ref).apps.push(app);
    }
  }

  const developerDir = path.join(websiteDir, 'developer');
  fs.rmSync(developerDir, { recursive: true, force: true });

  for (const developer of byRef.values()) {
    const dir = path.join(developerDir, developer.ref);
    fs.mkdirSync(dir, { recursive: true });

    const count = developer.apps.length;
    const pageTitle = `Nostr developer ${developer.shortKey} — ${count} self-hosted app${count === 1 ? '' : 's'} on Yantr`;
    const pageDescription = `${count} app${count === 1 ? '' : 's'} credited to one Nostr account (${developer.platformLabel}) in the Yantr catalog: ${developer.apps
      .map((a) => a.name)
      .join(', ')}. One-click self-hosting with Podman.`;

    const html = env.render('developer.njk', {
      developer,
      nowIso: generatedAt,
      pageTitle,
      pageDescription,
      pageUrl: `${siteUrl}${developerPagePath(developer.ref)}`,
    });

    fs.writeFileSync(path.join(dir, 'index.html'), html, 'utf8');
  }

  console.log(`✨ Generated ${byRef.size} developer page(s) in ${developerDir}`);
}

function httpsGet(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'yantr-website-builder' } }, (res) => {
      let body = '';
      res.on('data', (chunk) => (body += chunk));
      res.on('end', () => {
        try { resolve(JSON.parse(body)); } catch (e) { reject(e); }
      });
    }).on('error', reject);
  });
}

async function fetchAllContributors() {
  const allContributors = [];
  let page = 1;
  while (true) {
    const data = await httpsGet(`https://api.github.com/repos/besoeasy/yantr/contributors?per_page=100&page=${page}`);
    if (!Array.isArray(data) || data.length === 0) break;
    allContributors.push(...data);
    if (data.length < 100) break;
    page++;
  }
  return allContributors;
}

function normalizeContributors(contributors) {
  return contributors
    .filter((contributor) => contributor?.type === 'User' && contributor?.login && !contributor.login.endsWith('[bot]'))
    .map((contributor) => ({
      login: contributor.login,
      contributions: contributor.contributions || 0,
      avatar_url: contributor.avatar_url || '',
      url: contributor.html_url || contributor.url || '',
    }));
}

async function fetchGitHubData() {
  const githubDataPath = path.join(websiteDir, 'github-data.json');

  if (fs.existsSync(githubDataPath)) {
    console.log('\n🐙 GitHub data already present — skipping fetch.');
    return;
  }

  console.log('\n🐙 Fetching GitHub data...');

  try {
    const [repoData, allContributors] = await Promise.all([
      httpsGet('https://api.github.com/repos/besoeasy/yantr'),
      fetchAllContributors(),
    ]);

    const output = {
      fetchedAt: new Date().toISOString(),
      stars: repoData.stargazers_count || 0,
      forks: repoData.forks_count || 0,
      contributors: normalizeContributors(allContributors),
    };

    fs.writeFileSync(githubDataPath, JSON.stringify(output, null, 2), 'utf8');
    console.log(`✅ GitHub data saved (${output.stars} stars, ${output.contributors.length} contributors)`);
  } catch (error) {
    console.warn(`⚠️  Could not fetch GitHub data: ${error.message}`);
    // Write empty fallback so the page doesn't break
    if (!fs.existsSync(githubDataPath)) {
      fs.writeFileSync(githubDataPath, JSON.stringify({ fetchedAt: new Date().toISOString(), stars: 0, forks: 0, contributors: [] }, null, 2), 'utf8');
    }
  }
}

// ── Fleet presence snapshot ──────────────────────────────────────
// Same feed the app UI shows on its Fleet page (core/presence/stats.go):
// anonymous presence pings aggregated from the public ntfy topic.
// The snapshot is written to website/presence.json so yantr.org renders
// instantly; the page then refreshes live from ntfy in the browser.
//
// The aggregation itself is in website/fleet.js, which the page imports too.
const presenceJsonPath = path.join(websiteDir, 'presence.json');
const PRESENCE_TOPIC = (
  process.env.YANTR_PRESENCE_TOPIC ||
  'https://ntfy.sh/yantr'
).replace(/\/+$/, '');

function httpsGetText(url, timeoutMs = 15000) {
  return new Promise((resolve, reject) => {
    const req = https.get(
      url,
      { headers: { 'User-Agent': 'yantr-website-builder', Accept: 'application/x-ndjson' }, timeout: timeoutMs },
      (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          httpsGetText(res.headers.location, timeoutMs).then(resolve, reject);
          res.resume();
          return;
        }
        let body = '';
        res.on('data', (chunk) => (body += chunk));
        res.on('end', () => {
          if (res.statusCode >= 400) reject(new Error(`ntfy status ${res.statusCode}`));
          else resolve(body);
        });
      }
    );
    req.on('error', reject);
    req.on('timeout', () => req.destroy(new Error('ntfy fetch timed out')));
  });
}

async function fetchPresenceSnapshot() {
  console.log('\n📡 Fetching fleet presence snapshot...');
  let appNames = {};
  try {
    const catalog = JSON.parse(fs.readFileSync(appsJsonPath, 'utf8'));
    for (const app of catalog?.apps || []) {
      if (app?.id) appNames[app.id] = app.name || app.id;
    }
  } catch (error) {
    console.warn(`⚠️  Could not read apps.json for presence names: ${error.message}`);
  }

  try {
    const ndjson = await httpsGetText(`${PRESENCE_TOPIC}/json?poll=1&since=all`);
    const stats = aggregateFleet(parseFleetEvents(ndjson), appNames);
    fs.writeFileSync(presenceJsonPath, JSON.stringify({ fetchedAt: stats.fetchedAt, stats }, null, 2), 'utf8');
    console.log(`✅ Presence snapshot saved (${stats.nodes} nodes, ${stats.events} events)`);
  } catch (error) {
    console.warn(`⚠️  Could not fetch presence: ${error.message}`);
    // Snapshot fallback so the Fleet section still renders its empty state.
    if (!fs.existsSync(presenceJsonPath)) {
      const stats = emptyFleetStats();
      fs.writeFileSync(presenceJsonPath, JSON.stringify({ fetchedAt: stats.fetchedAt, stats }, null, 2), 'utf8');
    }
  }
}

try {
  await fetchGitHubData();
  buildPages();
  await fetchPresenceSnapshot();
} catch (error) {
  console.error('❌ Failed to generate app pages:', error.message);
  process.exit(1);
}
