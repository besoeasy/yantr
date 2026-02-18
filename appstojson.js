import fs from 'fs';
import path from 'path';
import { parse } from 'yaml';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const appsDir = path.join(__dirname, 'apps');
const outputDir = path.join(__dirname, 'website');
const outputFile = path.join(outputDir, 'apps.json');

/**
 * Extract yantr labels from compose.yml service
 */
function extractYantrLabels(service) {
  const labels = service?.labels || {};
  const yantrData = {};

  for (const [key, value] of Object.entries(labels)) {
    if (key.startsWith('yantr.')) {
      const labelName = key.replace('yantr.', '');
      yantrData[labelName] = value;
    }
  }

  return yantrData;
}

/**
 * Parse compose.yml and extract app information
 */
function parseComposeFile(appId, composePath) {
  try {
    const composeContent = fs.readFileSync(composePath, 'utf8');
    const composeData = parse(composeContent);

    if (!composeData?.services) {
      console.warn(`⚠️  No services found in ${appId}/compose.yml`);
      return null;
    }

    // Get the first service (usually the main app service)
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
      category: yantrLabels.category ? yantrLabels.category.split(',').map(c => c.trim()) : [],
      port: yantrLabels.port || null,
      description: yantrLabels.description || '',
      website: yantrLabels.website || null,
      dependencies: yantrLabels.dependencies ? yantrLabels.dependencies.split(',').map(d => d.trim()) : [],
      image: firstService.image || null,
      serviceName: firstServiceName,
    };
  } catch (error) {
    console.error(`❌ Error parsing ${appId}/compose.yml:`, error.message);
    return null;
  }
}

/**
 * Main function to build apps.json
 */
async function buildAppsJson() {
  console.log('🔨 Building apps.json from apps folder...\n');

  // Create website directory if it doesn't exist
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
    console.log(`✅ Created directory: ${outputDir}\n`);
  }

  // Read all app directories
  const appDirs = fs.readdirSync(appsDir, { withFileTypes: true })
    .filter(dirent => dirent.isDirectory())
    .map(dirent => dirent.name)
    .filter(name => name !== 'node_modules'); // Exclude node_modules if any

  const apps = [];
  const stats = {
    total: appDirs.length,
    success: 0,
    skipped: 0,
    failed: 0,
  };

  // Process each app
  for (const appId of appDirs.sort()) {
    const composePath = path.join(appsDir, appId, 'compose.yml');

    // Skip if compose.yml doesn't exist
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

  // Build final JSON structure
  const output = {
    meta: {
      generatedAt: new Date().toISOString(),
      totalApps: apps.length,
      categories: [...new Set(apps.flatMap(app => app.category))].sort(),
    },
    apps: apps,
  };

  // Write to file
  fs.writeFileSync(outputFile, JSON.stringify(output, null, 2), 'utf8');

  console.log('\n' + '='.repeat(60));
  console.log(`📊 Summary:`);
  console.log(`   Total directories: ${stats.total}`);
  console.log(`   ✅ Successfully processed: ${stats.success}`);
  console.log(`   ⚠️  Skipped: ${stats.skipped}`);
  console.log(`   ❌ Failed: ${stats.failed}`);
  console.log('='.repeat(60));
  console.log(`\n✨ apps.json generated successfully!`);
  console.log(`📁 Output: ${outputFile}`);
  console.log(`📦 Total apps: ${apps.length}`);
  console.log(`🏷️  Categories: ${output.meta.categories.length}`);
}

// Run the script
buildAppsJson().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
