#!/usr/bin/env node
/**
 * propose-developers.js — propose `x-yantr.developers` entries from the image
 * namespace in each app's compose file.
 *
 * Usage:
 *   node scripts/propose-developers.js            # report only, writes nothing
 *   node scripts/propose-developers.js --write    # apply the high-confidence set
 *
 * Why this exists and why it is not a bulk edit:
 *
 * A `developers` entry becomes a profile link and, on GitHub, an @mention in
 * every issue filed about the app. A wrong one subscribes a stranger to bug
 * reports, so the field cannot be machine-generated and committed — it has to be
 * proposed and read. This script proposes; `git diff` is the review; the commit
 * is yours.
 *
 * The signal it uses is the image namespace. `ghcr.io/<owner>/...` is a GitHub
 * account by construction, which makes it high confidence. A Docker Hub
 * namespace is only a hint — the publisher there may have no GitHub account, or
 * a different name — so those are proposed for review instead, and only when the
 * app's own website corroborates the owner.
 *
 * It only ever adds one line (right after `website:`), never rewrites a file.
 * The catalog's compose files are hand-formatted — folded descriptions, flow
 * sequences with deliberate wrapping — and re-serialising them with a YAML
 * library would reformat all 168 of them.
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import YAML from 'yaml';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const appsDir = path.join(root, 'apps');

// Registries that mean "this namespace is a GitHub account".
const GITHUB_REGISTRIES = new Set(['ghcr.io']);

// Organisations that publish images on behalf of the upstream project. They are
// the upstream author for the purposes of this field, which is why they are
// credited rather than skipped — but they are named here so the report can say
// "an organisation, not a person" and let you decide.
const KNOWN_ORGS = new Set(['linuxserver', 'immich-app', 'wg-easy', 'itskovacs', 'egzola']);

// Packagers, not authors. LinuxServer builds images of other people's software,
// so `ghcr.io/linuxserver/plex` does not make LinuxServer the author of Plex.
// These are pulled out of the high-confidence set on purpose: the field means
// the upstream author, and for a packaged app that is the project itself.
const PACKAGER_ORGS = new Set(['linuxserver']);

// A bare `postgres:16` or `alpine` is the Docker Hub "library" namespace: the
// upstream authors are the project maintainers, not a GitHub handle the image
// name identifies. There is no correct value to propose, so propose none.
const UNCREDITED_NAMESPACES = new Set(['library', 'docker.io', 'quay.io', 'ghcr.io', 'lscr.io', 'gcr.io']);

function readCompose(file) {
  try {
    return YAML.parse(fs.readFileSync(file, 'utf8'));
  } catch (error) {
    return { __error: error.message };
  }
}

// imageNamespace reduces an image reference to its publishing account, or null
// when the reference carries none.
function imageNamespace(image) {
  if (typeof image !== 'string' || !image.trim()) return null;

  const ref = image.trim().replace(/^docker:\/\//, '');
  const parts = ref.split('/');

  // A single segment (`alpine`) is a Docker Hub official image; two segments
  // without a dot in the first (`lissy93/domain-locker`) are a Docker Hub
  // account. Three or more, or a first segment containing a dot, name a
  // registry first.
  let registry = 'docker.io';
  let segments = parts;
  if (parts.length >= 2 && (/[.:]/.test(parts[0]) || parts[0] === 'localhost')) {
    registry = parts[0];
    segments = parts.slice(1);
  }

  if (segments.length < 2) return null; // official image, no account
  const owner = segments[0];

  if (registry === 'lscr.io' && owner === 'linuxserver') {
    // LinuxServer publishes to lscr.io and ghcr.io from the same GitHub org.
    return { registry: 'ghcr.io', owner };
  }
  return { registry, owner };
}

// githubOwnerFromWebsite reads the owner out of a github.com URL in the app's
// own metadata. Used to corroborate a Docker Hub namespace, which on its own is
// only a hint.
function githubOwnerFromWebsite(...urls) {
  for (const url of urls) {
    if (typeof url !== 'string') continue;
    const match = url.match(/github\.com\/([A-Za-z0-9][A-Za-z0-9-]*)/);
    if (match) return match[1];
  }
  return null;
}

function analyse(folder) {
  const file = path.join(appsDir, folder, 'compose.yml');
  const compose = readCompose(file);

  if (compose?.__error) {
    return { folder, tier: 'error', reason: `unparseable compose: ${compose.__error}` };
  }

  const meta = compose?.['x-yantr'] ?? {};
  if (Array.isArray(meta.developers) && meta.developers.length > 0) {
    return { folder, tier: 'has-field', reason: `already declares ${meta.developers.join(', ')}` };
  }

  const images = Object.values(compose?.services ?? {})
    .map((service) => service?.image)
    .filter(Boolean);

  if (images.length === 0) {
    return { folder, tier: 'skip', reason: 'no image in the compose file' };
  }

  const namespaces = [...new Set(images.map(imageNamespace).filter(Boolean).map((n) => `${n.registry}/${n.owner}`))];

  if (namespaces.length === 0) {
    return { folder, tier: 'skip', reason: `official images only (${images.join(', ')})` };
  }
  if (namespaces.length > 1) {
    // Genuinely ambiguous: the app bundles images from several accounts and the
    // field can only name one author. A human decides whose project this is.
    return { folder, tier: 'review', reason: `several namespaces: ${namespaces.join(', ')}` };
  }

  const [registry, owner] = namespaces[0].split('/');
  const websiteOwner = githubOwnerFromWebsite(meta.website);
  const orgNote = KNOWN_ORGS.has(owner) ? ' (organisation, not a person)' : '';

  // Packagers are checked before the registry, because LinuxServer publishes to
  // both ghcr.io and Docker Hub and the answer is the same either way.
  if (PACKAGER_ORGS.has(owner)) {
    return {
      folder,
      tier: 'review',
      handle: owner,
      reason: `${registry}/${owner} packages third-party software; the author is the project itself`,
    };
  }

  if (GITHUB_REGISTRIES.has(registry)) {
    return {
      folder,
      tier: 'high',
      handle: owner,
      reason: `${registry}/${owner} is a GitHub account${orgNote}`,
    };
  }

  if (!UNCREDITED_NAMESPACES.has(owner)) {
    if (websiteOwner && websiteOwner.toLowerCase() === owner.toLowerCase()) {
      return {
        folder,
        tier: 'review',
        handle: owner,
        reason: `Docker Hub namespace ${owner} matches the project's github.com owner`,
      };
    }
    if (websiteOwner) {
      return {
        folder,
        tier: 'review',
        handle: websiteOwner,
        reason: `website is github.com/${websiteOwner}, but the image is from ${registry}/${owner}`,
      };
    }
  }

  return {
    folder,
    tier: 'skip',
    reason: `nothing ties ${registry}/${owner} to a GitHub account`,
  };
}

// applyDeveloper inserts the developers line directly after the website line,
// reusing that line's own indentation and preserving the rest of the file
// byte-for-byte.
function applyDeveloper(folder, handle) {
  const file = path.join(appsDir, folder, 'compose.yml');
  const text = fs.readFileSync(file, 'utf8');
  const lines = text.split('\n');

  if (lines.some((line) => /^\s*developers:/.test(line))) {
    return { applied: false, reason: 'developers already present' };
  }

  const index = lines.findIndex((line) => /^(\s*)website:(\s|$)/.test(line));
  if (index === -1) {
    return { applied: false, reason: 'no single-line website: to anchor to' };
  }

  const indent = lines[index].match(/^(\s*)/)[1];
  lines.splice(index + 1, 0, `${indent}developers: ["github:${handle}"]`);
  fs.writeFileSync(file, lines.join('\n'), 'utf8');
  return { applied: true };
}

const write = process.argv.includes('--write');
const tiers = { high: [], review: [], skip: [], 'has-field': [], error: [] };

for (const folder of fs.readdirSync(appsDir).sort()) {
  if (!fs.statSync(path.join(appsDir, folder)).isDirectory()) continue;
  if (!fs.existsSync(path.join(appsDir, folder, 'compose.yml'))) continue;
  const result = analyse(folder);
  tiers[result.tier].push(result);
}

function section(title, entries, note) {
  console.log(`\n${title} — ${entries.length}`);
  if (note) console.log(`  ${note}`);
  for (const entry of entries) {
    const line = entry.handle ? `developers: ["github:${entry.handle}"]` : '';
    console.log(`  ${entry.folder.padEnd(24)} ${line.padEnd(38)} # ${entry.reason}`);
  }
}

section('HIGH CONFIDENCE (ghcr.io namespace = a GitHub account)', tiers.high);
section(
  'NEEDS REVIEW (Docker Hub namespace, or several accounts)',
  tiers.review,
  'A Docker Hub namespace is a hint, not proof. Check each one before adding.'
);
section('ALREADY CREDITED', tiers['has-field']);
section('SKIPPED (no account the name identifies)', tiers.skip);
if (tiers.error.length) section('ERRORS', tiers.error);

if (!write) {
  console.log(
    `\nTotals: ${tiers.high.length} high confidence, ${tiers.review.length} to review, ` +
      `${tiers.skip.length} with no match, ${tiers['has-field'].length} already credited.`
  );
  console.log('Report only — nothing was written. Re-run with --write to apply the');
  console.log('high-confidence set, then read `git diff` before committing.');
  process.exit(tiers.error.length ? 1 : 0);
}

let applied = 0;
const refused = [];
for (const entry of tiers.high) {
  const result = applyDeveloper(entry.folder, entry.handle);
  if (result.applied) applied += 1;
  else refused.push(`${entry.folder} (${result.reason})`);
}

console.log(`\nWrote developers into ${applied} app(s). Review the diff before committing:`);
console.log('  git diff --stat apps/');
if (refused.length) console.log(`\nRefused ${refused.length}: ${refused.join('; ')}`);
console.log(`\nLeft alone: ${tiers.review.length} needing review, ${tiers.skip.length} with no confident match.`);
process.exit(tiers.error.length ? 1 : 0);
