import fs from 'fs';
import https from 'https';
import path from 'path';
import nunjucks from 'nunjucks';
import { parse } from 'yaml';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const websiteDir = path.join(__dirname, 'website');
const templatesDir = path.join(websiteDir, 'templates');
const appsJsonPath = path.join(websiteDir, 'apps.json');
const appsOutputDir = path.join(websiteDir, 'apps');
const appsDir = path.join(__dirname, 'apps');
const siteUrl = 'https://yantr.org';

function normaliseLabels(raw) {
  if (!raw) return {};
  if (Array.isArray(raw)) {
    const out = {};
    for (const l of raw) {
      if (typeof l !== 'string') continue;
      const idx = l.indexOf('=');
      out[idx === -1 ? l : l.slice(0, idx)] = idx === -1 ? '' : l.slice(idx + 1);
    }
    return out;
  }
  return typeof raw === 'object' ? raw : {};
}

function parsePortLabels(services) {
  const ports = [];
  const seen = new Set();
  for (const svc of Object.values(services ?? {})) {
    const labels = normaliseLabels(svc?.labels);
    for (const [key, protocol] of Object.entries(labels)) {
      if (!key.startsWith('yantr.port.')) continue;
      const portNum = parseInt(key.replace('yantr.port.', ''), 10);
      if (isNaN(portNum) || seen.has(portNum)) continue;
      seen.add(portNum);
      const serviceLabel = labels[`yantr.service.${portNum}`] || `Port ${portNum}`;
      ports.push({ port: portNum, protocol: protocol.toUpperCase(), label: serviceLabel });
    }
  }
  return ports;
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
    const ports = parsePortLabels(services);

    return {
      id: appId,
      name: meta.name,
      logo: meta.logo || null,
      tags: Array.isArray(meta.tags) ? meta.tags : [],
      ports,
      short_description: meta.short_description || '',
      description: meta.description || meta.short_description || '',
      usecases: Array.isArray(meta.usecases) ? meta.usecases : [],
      website: meta.website || null,
      notes: Array.isArray(meta.notes) ? meta.notes : [],
      image,
      serviceName,
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
}

function getLogoUrl(app) {
  if (!app?.logo) {
    return `https://ui-avatars.com/api/?name=${encodeURIComponent(app?.name || app?.id || 'App')}&background=random&color=fff&bold=true`;
  }

  return app.logo.startsWith('http') ? app.logo : `https://ipfs.io/ipfs/${app.logo}`;
}

function toAppViewModel(app) {
  const id = app?.id || 'unknown-app';
  const name = app?.name || id;
  const tags = Array.isArray(app?.tags) ? app.tags : [];
  const primaryTag = tags[0] || 'self-hosted';
  const summary = app?.short_description || app?.description || 'No description available.';
  const appUrl = `${siteUrl}/apps/${id}/`;

  return {
    ...app,
    id,
    name,
    tags,
    notes: Array.isArray(app?.notes) ? app.notes : [],
    short_description: app?.short_description || '',
    description: app?.description || app?.short_description || 'No description available.',
    usecases: Array.isArray(app?.usecases) ? app.usecases : [],
    logoUrl: getLogoUrl(app),
    summary,
    primaryTag,
    appUrl,
    appPagePath: `/apps/${id}/`,
    sourceComposeUrl: `https://github.com/besoeasy/yantr/blob/main/apps/${id}/compose.yml`,
    sourceAppFolderUrl: `https://github.com/besoeasy/yantr/tree/main/apps/${id}`,
    appSearchIntentTitle: `${name} Docker Compose Setup`,
    appSearchIntentDescription: `Learn how to self-host ${name} with Docker Compose using Yantr. ${summary}`,
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

    const relatedApps = getRelatedApps(app, apps);

    const pageDescription = app.short_description
      ? `Self-host ${app.name} with Docker. ${app.short_description} Deploy it in seconds with Yantr.`
      : `Learn how to self-host ${app.name} on your homelab using Docker. ${app.description}${app.description.endsWith('.') ? '' : '.'} Easy one-click setup with Yantr.`;

    const html = env.render('app.njk', {
      app,
      relatedApps,
      nowIso: generatedAt,
      pageTitle: `Self-Host ${app.name} with Docker | Yantr`,
      pageDescription,
      pageUrl: app.appUrl,
      imageUrl: app.logoUrl,
    });

    fs.writeFileSync(path.join(appDir, 'index.html'), html, 'utf8');
  }

  console.log(`✨ Generated ${apps.length} app pages in ${appsOutputDir}`);
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

try {
  await fetchGitHubData();
  buildPages();
} catch (error) {
  console.error('❌ Failed to generate app pages:', error.message);
  process.exit(1);
}
