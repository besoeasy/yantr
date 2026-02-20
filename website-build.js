import fs from 'fs';
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

function extractYantrLabels(service) {
  const labels = service?.labels || {};
  const yantrData = {};

  if (Array.isArray(labels)) {
    for (const rawLabel of labels) {
      if (typeof rawLabel !== 'string') continue;
      const separatorIndex = rawLabel.indexOf('=');
      if (separatorIndex === -1) continue;

      const key = rawLabel.slice(0, separatorIndex).trim();
      const value = rawLabel.slice(separatorIndex + 1).trim();
      if (!key.startsWith('yantr.')) continue;

      yantrData[key.replace('yantr.', '')] = value;
    }

    return yantrData;
  }

  for (const [key, value] of Object.entries(labels)) {
    if (key.startsWith('yantr.')) {
      const labelName = key.replace('yantr.', '');
      yantrData[labelName] = value;
    }
  }

  return yantrData;
}

function parseComposeFile(appId, composePath) {
  try {
    const composeContent = fs.readFileSync(composePath, 'utf8');
    const composeData = parse(composeContent);

    if (!composeData?.services) {
      console.warn(`⚠️  No services found in ${appId}/compose.yml`);
      return null;
    }

    const firstServiceName = Object.keys(composeData.services)[0];
    const firstService = composeData.services[firstServiceName];

    const yantrLabels = extractYantrLabels(firstService);
    if (!yantrLabels.name) {
      console.warn(`⚠️  No yantr.name label found for ${appId}`);
      return null;
    }

    return {
      id: appId,
      name: yantrLabels.name || appId,
      logo: yantrLabels.logo || null,
      category: yantrLabels.category ? yantrLabels.category.split(',').map((item) => item.trim()) : [],
      port: yantrLabels.port || null,
      description: yantrLabels.description || '',
      website: yantrLabels.website || null,
      dependencies: yantrLabels.dependencies ? yantrLabels.dependencies.split(',').map((item) => item.trim()) : [],
      image: firstService.image || null,
      serviceName: firstServiceName,
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
    const composePath = path.join(appsDir, appId, 'compose.yml');

    if (!fs.existsSync(composePath)) {
      console.warn(`⚠️  Skipping ${appId}: compose.yml not found`);
      stats.skipped++;
      continue;
    }

    const appData = parseComposeFile(appId, composePath);
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
      categories: [...new Set(apps.flatMap((app) => app.category))].sort(),
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
  console.log(`🏷️  Categories: ${output.meta.categories.length}`);
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
  const category = Array.isArray(app?.category) ? app.category : [];
  const dependencies = Array.isArray(app?.dependencies) ? app.dependencies : [];

  return {
    ...app,
    id,
    name,
    category,
    dependencies,
    description: app?.description || 'No description available.',
    logoUrl: getLogoUrl(app),
    appPagePath: `/apps/${id}/`,
    sourceComposeUrl: `https://github.com/besoeasy/yantr/blob/main/apps/${id}/compose.yml`,
    sourceAppFolderUrl: `https://github.com/besoeasy/yantr/tree/main/apps/${id}`,
  };
}

function getRelatedApps(app, allApps, count = 3) {
  const others = allApps.filter(
    (a) => a.id !== app.id && a.category.some((c) => app.category.includes(c))
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

  fs.rmSync(appsOutputDir, { recursive: true, force: true });
  fs.mkdirSync(appsOutputDir, { recursive: true });

  for (const app of apps) {
    const appDir = path.join(appsOutputDir, app.id);
    fs.mkdirSync(appDir, { recursive: true });

    const relatedApps = getRelatedApps(app, apps);

    const html = env.render('app.njk', {
      app,
      relatedApps,
      nowIso: new Date().toISOString(),
      pageTitle: `${app.name} | Yantr App Catalog`,
      pageDescription: app.description,
      pageUrl: `https://yantr.org/apps/${app.id}/`,
      imageUrl: app.logoUrl,
    });

    fs.writeFileSync(path.join(appDir, 'index.html'), html, 'utf8');
  }

  console.log(`✨ Generated ${apps.length} app pages in ${appsOutputDir}`);
}

try {
  buildPages();
} catch (error) {
  console.error('❌ Failed to generate app pages:', error.message);
  process.exit(1);
}
