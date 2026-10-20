#!/usr/bin/env node
// check.js — validate every app recipe under apps/.
//
// The app contract lives in AGENTS.md and is enforced in Go by
// core/apps/catalog_test.go. This is the fast, fail-fast companion: apps are
// read one at a time in sorted order, and the first hard violation stops the
// run with a non-zero exit.
//
// A few rules predate the current catalog and are reported as warnings so this
// script can still run clean today. `--strict` promotes those warnings to
// errors, which is the mode a new app or a CI gate should use.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from 'yaml';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const APPS_DIR = process.env.YANTR_APPS_DIR
  ? path.resolve(process.env.YANTR_APPS_DIR)
  : path.join(HERE, 'apps');
const STRICT = process.argv.includes('--strict');

const PORT_PROTOCOLS = new Set(['HTTP', 'HTTPS', 'TCP', 'UDP']);
const ACCESS_PROTOCOLS = new Set(['HTTP', 'HTTPS']);
const ENV_CHARSETS = new Set(['alnum', 'hex', 'numeric', 'alpha', 'base64url', 'alnum_symbols']);
const APP_FILES = new Set(['compose.yml', 'logo.svg']);
const HOST_SOCKET = '${HOST_PODMAN_SOCKET}';
const CONTAINER_SOCKET = '/var/run/docker.sock';

// Apps that genuinely publish no browser-reachable surface. Mirrors the
// allow-list in core/apps/catalog_test.go; add to both deliberately.
const NO_ACCESS_APPS = new Set(['mariadb', 'mongodb', 'samba']);

// Labels that name protocol clients or diagnostics endpoints are plumbing
// even when they speak HTTP, so they must stay work ports.
const DIAGNOSTIC_LABELS = new Set(['Metrics / pprof', 'Postgres client', 'gRPC service']);

const warnings = [];

class AppError extends Error {
  constructor(appId, message) {
    super(message);
    this.name = 'AppError';
    this.appId = appId;
  }
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function text(value) {
  return String(value ?? '').replace(/\s+/g, ' ').trim();
}

function fail(appId, message) {
  throw new AppError(appId, message);
}

// A soft issue is a contract violation the existing catalog still carries.
// Default mode records it; --strict makes it stop the run like any other error.
function warn(appId, message) {
  if (STRICT) fail(appId, message);
  warnings.push({ appId, message });
}

// ─── App folder ──────────────────────────────────────────────────────────────

function checkFiles(appId, appDir) {
  const entries = fs.readdirSync(appDir, { withFileTypes: true });

  if (!entries.some((entry) => entry.isFile() && entry.name === 'compose.yml')) {
    fail(appId, 'app folder has no compose.yml');
  }

  for (const entry of entries) {
    if (!entry.isFile()) {
      fail(appId, `app folder contains a directory "${entry.name}"; only compose.yml and logo.svg belong here`);
    }
    if (!APP_FILES.has(entry.name)) {
      fail(appId, `unexpected file "${entry.name}"; app folders contain only compose.yml and logo.svg`);
    }
  }
}

// ─── x-yantr metadata ────────────────────────────────────────────────────────

function checkMetadata(appId, meta) {
  if (!isObject(meta)) {
    fail(appId, 'compose.yml has no x-yantr block');
  }
  if (typeof meta.name !== 'string' || text(meta.name) === '') {
    fail(appId, 'x-yantr.name is required');
  }

  if (!Array.isArray(meta.tags) || meta.tags.length < 3 || meta.tags.length > 5) {
    fail(appId, `x-yantr.tags must contain 3–5 tags (got ${Array.isArray(meta.tags) ? meta.tags.length : 'none'})`);
  }
  for (const tag of meta.tags) {
    if (typeof tag !== 'string' || text(tag) === '') {
      fail(appId, 'x-yantr.tags contains an empty or non-string entry');
    }
  }
  if (new Set(meta.tags.map((tag) => tag.trim())).size !== meta.tags.length) {
    fail(appId, 'x-yantr.tags contains a duplicate');
  }

  const shortDescription = text(meta.short_description);
  if (typeof meta.short_description !== 'string' || shortDescription.length < 50 || shortDescription.length > 100) {
    fail(appId, `x-yantr.short_description must be a 50–100 character string (got ${shortDescription.length})`);
  }

  const description = text(meta.description);
  if (typeof meta.description !== 'string' || description.length < 200 || description.length > 300) {
    fail(appId, `x-yantr.description must be a 200–300 character string (got ${description.length})`);
  }

  if (!Array.isArray(meta.usecases) || meta.usecases.length < 2) {
    fail(appId, 'x-yantr.usecases must contain at least 2 entries');
  }
  for (const usecase of meta.usecases) {
    if (typeof usecase !== 'string' || text(usecase) === '') {
      fail(appId, 'x-yantr.usecases contains an empty or non-string entry');
    }
  }

  if (typeof meta.website !== 'string' || !/^https?:\/\/\S+$/i.test(meta.website.trim())) {
    fail(appId, 'x-yantr.website must be an http(s) URL');
  }

  if (meta.notes !== undefined) {
    if (!Array.isArray(meta.notes)) {
      fail(appId, 'x-yantr.notes must be a list');
    }
    for (const note of meta.notes) {
      if (typeof note !== 'string' || text(note) === '') {
        fail(appId, 'x-yantr.notes contains an empty or non-string entry');
      }
    }
  }
}

// ─── Nostr developers ────────────────────────────────────────────────────────

const BECH32_CHARSET = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';

function bech32Polymod(values) {
  const generator = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];
  let checksum = 1;
  for (const value of values) {
    const top = checksum >> 25;
    checksum = ((checksum & 0x1ffffff) << 5) ^ value;
    for (let i = 0; i < 5; i += 1) {
      if ((top >> i) & 1) checksum ^= generator[i];
    }
  }
  return checksum;
}

function bech32HrpExpand(hrp) {
  const high = [...hrp].map((char) => char.charCodeAt(0) >> 5);
  const low = [...hrp].map((char) => char.charCodeAt(0) & 31);
  return [...high, 0, ...low];
}

// decodeNpub mirrors the part of core/nip19 that x-yantr.developers relies on:
// a bech32 checksum, a 32-byte payload, and nothing else. A typo is an error,
// not a key the catalog silently drops.
function decodeNpub(value) {
  if (typeof value !== 'string') return false;

  const key = value.trim();
  const lower = key.toLowerCase();
  if (lower !== key && lower.toUpperCase() !== key) return false;

  const separator = lower.lastIndexOf('1');
  if (separator < 1 || lower.slice(0, separator) !== 'npub') return false;
  if (separator + 7 > lower.length) return false;

  const data = [...lower.slice(separator + 1)].map((char) => BECH32_CHARSET.indexOf(char));
  if (data.some((valueIndex) => valueIndex === -1)) return false;
  if (bech32Polymod([...bech32HrpExpand('npub'), ...data]) !== 1) return false;

  const words = data.slice(0, -6);
  let accumulator = 0;
  let bits = 0;
  const bytes = [];
  for (const word of words) {
    accumulator = (accumulator << 5) | word;
    bits += 5;
    while (bits >= 8) {
      bits -= 8;
      bytes.push((accumulator >> bits) & 0xff);
    }
  }
  if (bits >= 5 || ((accumulator << (8 - bits)) & 0xff) !== 0) return false;
  return bytes.length === 32;
}

function checkDevelopers(appId, developers) {
  if (developers === undefined) return;
  if (!Array.isArray(developers)) {
    fail(appId, 'x-yantr.developers must be a list of npub keys');
  }
  for (const developer of developers) {
    if (typeof developer !== 'string' || !decodeNpub(developer)) {
      fail(appId, `x-yantr.developers entry ${JSON.stringify(developer)} is not a valid npub key`);
    }
  }
}

// ─── Env generators ──────────────────────────────────────────────────────────

function checkEnvGenerators(appId, generators, composeSource) {
  if (generators === undefined) {
    generators = {};
  }
  if (!isObject(generators)) {
    fail(appId, 'x-yantr.env_generators must be a map');
  }

  for (const [name, rule] of Object.entries(generators)) {
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) {
      fail(appId, `x-yantr.env_generators key ${JSON.stringify(name)} is not an environment variable name`);
    }
    if (!isObject(rule)) {
      fail(appId, `x-yantr.env_generators.${name} must be a map`);
    }
    if (!Number.isInteger(rule.length) || rule.length < 1) {
      fail(appId, `x-yantr.env_generators.${name}.length must be a positive integer`);
    }
    if (!ENV_CHARSETS.has(rule.charset)) {
      fail(appId, `x-yantr.env_generators.${name}.charset must be one of ${[...ENV_CHARSETS].join(', ')}`);
    }
  }

  // Every ${VAR} with no default would deploy empty without a generator.
  // HOST_PODMAN_SOCKET is injected by the backend and is never generated.
  const referenced = new Set();
  const missingGenerators = new Set();
  const envRefPattern = /(?<!\$)\$\{([A-Za-z_][A-Za-z0-9_]*)([^}]*)\}/g;
  for (const match of composeSource.matchAll(envRefPattern)) {
    const [, name, rest] = match;
    if (name === 'HOST_PODMAN_SOCKET') continue;

    // ${VAR:-x}, ${VAR-x}, ${VAR:=x} and ${VAR=x} supply a default;
    // ${VAR}, ${VAR:?err}, ${VAR?err} and ${VAR:+x} do not.
    const hasDefault = rest.startsWith(':-') || rest.startsWith('-') || rest.startsWith(':=') || rest.startsWith('=');
    referenced.add(name);
    if (!hasDefault && !Object.hasOwn(generators, name)) {
      missingGenerators.add(name);
    }
  }
  for (const name of missingGenerators) {
    warn(appId, `\${${name}} has no default and no x-yantr.env_generators entry`);
  }

  for (const name of Object.keys(generators)) {
    if (!referenced.has(name)) {
      warn(appId, `x-yantr.env_generators.${name} is not referenced without a default in compose.yml`);
    }
  }
}

// ─── x-yantr ports ───────────────────────────────────────────────────────────

function checkPorts(appId, ports, services) {
  if (ports === undefined) return;
  if (!Array.isArray(ports)) {
    fail(appId, 'x-yantr.ports must be a list');
  }

  let shown = 0;
  for (const [index, port] of ports.entries()) {
    const at = `x-yantr.ports[${index}]`;

    if (!isObject(port)) {
      fail(appId, `${at} must be a map`);
    }
    if (!Number.isInteger(port.port) || port.port < 1 || port.port > 65535) {
      fail(appId, `${at}.port must be an integer between 1 and 65535`);
    }

    const protocol = text(port.protocol).toUpperCase() || 'TCP';
    if (!PORT_PROTOCOLS.has(protocol)) {
      fail(appId, `${at}.protocol must be one of ${[...PORT_PROTOCOLS].join(', ')} (got ${JSON.stringify(port.protocol)})`);
    }

    const label = text(port.label);
    if (typeof port.label !== 'string' || label === '') {
      fail(appId, `${at}.label is required`);
    }

    const service = text(port.service);
    if (typeof port.service !== 'string' || service === '') {
      fail(appId, `${at}.service is required`);
    }
    if (!Object.hasOwn(services, service)) {
      fail(appId, `${at} references service "${service}", which does not exist in compose.yml`);
    }

    if (port.show !== undefined && typeof port.show !== 'boolean') {
      fail(appId, `${at}.show must be true or false`);
    }

    if (port.show === true) {
      shown += 1;
      if (!ACCESS_PROTOCOLS.has(protocol)) {
        fail(appId, `${at} is show: true but ${protocol} has no browser UI to open`);
      }
      if (DIAGNOSTIC_LABELS.has(label)) {
        fail(appId, `${at} label "${label}" is protocol plumbing and must not be show: true`);
      }
    }
  }

  if (ports.length > 0 && shown === 0 && !NO_ACCESS_APPS.has(appId)) {
    fail(appId, `publishes ${ports.length} port(s) but marks none show: true`);
  }
}

// ─── Compose services ────────────────────────────────────────────────────────

// imageTag returns the explicit tag, or null when the reference has none
// (Docker then implies :latest). A registry host:port is not a tag.
function imageTag(image) {
  if (image.includes('@')) return null;
  const lastSlash = image.lastIndexOf('/');
  const lastColon = image.lastIndexOf(':');
  return lastColon > lastSlash ? image.slice(lastColon + 1) : null;
}

function checkImage(appId, serviceName, image) {
  if (typeof image !== 'string' || image.trim() === '') {
    fail(appId, `service "${serviceName}" has no image`);
  }

  if (image.includes('@')) {
    warn(appId, `service "${serviceName}" pins image ${image}; the catalog uses :latest`);
    return;
  }

  const tag = imageTag(image);
  if (tag === null) {
    warn(appId, `service "${serviceName}" image ${image} has no explicit tag; add :latest`);
  } else if (tag !== 'latest') {
    warn(appId, `service "${serviceName}" image ${image} uses :${tag}; the catalog uses :latest`);
  }
}

function checkVolumes(appId, serviceName, volumes) {
  if (volumes === undefined) return;
  if (!Array.isArray(volumes)) {
    fail(appId, `service "${serviceName}" volumes must be a list`);
  }

  for (const [index, volume] of volumes.entries()) {
    const at = `service "${serviceName}" volumes[${index}]`;

    if (typeof volume === 'string') {
      // Anonymous volume: a container path with no source.
      if (!volume.includes(':')) continue;

      const source = volume.slice(0, volume.indexOf(':'));
      const remainder = volume.slice(volume.indexOf(':') + 1);
      const target = remainder.split(':')[0];

      if (source === HOST_SOCKET) {
        if (target !== CONTAINER_SOCKET) {
          fail(appId, `${at} mounts ${HOST_SOCKET} at ${target}; the container side must stay ${CONTAINER_SOCKET}`);
        }
        continue;
      }
      if (source.includes('.sock')) {
        fail(appId, `${at} mounts host socket ${source}; the host side must be ${HOST_SOCKET}`);
      }
      if (source.startsWith('/') || source.startsWith('.') || source.startsWith('~')) {
        fail(appId, `${at} is a bind mount (${source}); use a named volume`);
      }
      if (target === CONTAINER_SOCKET) {
        fail(appId, `${at} mounts ${source} at ${CONTAINER_SOCKET}; the host side must be ${HOST_SOCKET}`);
      }
      continue;
    }

    if (!isObject(volume)) {
      fail(appId, `${at} must be a string or a map`);
    }
    if (volume.type === 'bind') {
      fail(appId, `${at} is a bind mount; use a named volume`);
    }

    const source = volume.source;
    if (typeof source === 'string') {
      if (source === HOST_SOCKET) {
        if (volume.target !== CONTAINER_SOCKET) {
          fail(appId, `${at} mounts ${HOST_SOCKET} at ${volume.target}; the container side must stay ${CONTAINER_SOCKET}`);
        }
        continue;
      }
      if (source.includes('.sock')) {
        fail(appId, `${at} mounts host socket ${source}; the host side must be ${HOST_SOCKET}`);
      }
      if (source.startsWith('/') || source.startsWith('.') || source.startsWith('~')) {
        fail(appId, `${at} is a bind mount (${source}); use a named volume`);
      }
    }
    if (volume.target === CONTAINER_SOCKET && source !== HOST_SOCKET) {
      fail(appId, `${at} mounts ${JSON.stringify(source)} at ${CONTAINER_SOCKET}; the host side must be ${HOST_SOCKET}`);
    }
  }
}

function checkComposePorts(appId, serviceName, ports) {
  if (ports === undefined) return;
  if (!Array.isArray(ports)) {
    fail(appId, `service "${serviceName}" ports must be a list`);
  }

  for (const [index, entry] of ports.entries()) {
    const at = `service "${serviceName}" ports[${index}]`;

    if (typeof entry === 'number') {
      // Compose accepts this, but AGENTS.md asks for the quoted short form.
      warn(appId, `${at} is an unquoted port number; write "${entry}"`);
      continue;
    }

    if (typeof entry === 'string') {
      const value = entry.trim();
      if (/^\d+(?:-\d+)?(?:\/(?:tcp|udp))?$/i.test(value)) {
        continue;
      }
      if (/^\d+(?:-\d+)?:\d+(?:-\d+)?(?:\/(?:tcp|udp))?$/i.test(value)) {
        fail(appId, `${at} is a fixed host mapping "${entry}"; use auto assignment like "8080"`);
      }
      fail(appId, `${at} is not a container port; write auto assignment like "8080"`);
    }

    if (isObject(entry)) {
      if (entry.published !== undefined) {
        fail(appId, `${at} sets published; use auto assignment only`);
      }
      if (!Number.isInteger(entry.target) || entry.target < 1 || entry.target > 65535) {
        fail(appId, `${at}.target must be an integer between 1 and 65535`);
      }
      continue;
    }

    fail(appId, `${at} must be a string or a map`);
  }
}

function checkLabels(appId, serviceName, labels) {
  if (labels === undefined) return;

  const names = Array.isArray(labels)
    ? labels.map((label) => (typeof label === 'string' ? label.split('=')[0] : ''))
    : isObject(labels)
      ? Object.keys(labels)
      : [];

  if (!Array.isArray(labels) && !isObject(labels)) {
    fail(appId, `service "${serviceName}" labels must be a list or a map`);
  }

  for (const name of names) {
    if (name.startsWith('yantr.')) {
      fail(appId, `service "${serviceName}" sets Yantr label "${name}"; identity labels are injected by the backend`);
    }
  }
}

function checkServices(appId, services) {
  if (!isObject(services) || Object.keys(services).length === 0) {
    fail(appId, 'compose.yml has no services');
  }

  for (const [serviceName, service] of Object.entries(services)) {
    if (!isObject(service)) {
      fail(appId, `service "${serviceName}" must be a map`);
    }
    if (service.build !== undefined) {
      fail(appId, `service "${serviceName}" declares build:; the catalog ships prebuilt images only`);
    }

    checkImage(appId, serviceName, service.image);
    checkVolumes(appId, serviceName, service.volumes);
    checkComposePorts(appId, serviceName, service.ports);
    checkLabels(appId, serviceName, service.labels);
  }
}

// ─── One app ─────────────────────────────────────────────────────────────────

function checkApp(appId) {
  const appDir = path.join(APPS_DIR, appId);
  const composePath = path.join(appDir, 'compose.yml');

  checkFiles(appId, appDir);

  let source;
  try {
    source = fs.readFileSync(composePath, 'utf8');
  } catch (error) {
    fail(appId, `cannot read compose.yml: ${error.message}`);
  }

  let compose;
  try {
    compose = parse(source);
  } catch (error) {
    fail(appId, `compose.yml is not valid YAML: ${error.message}`);
  }
  if (!isObject(compose)) {
    fail(appId, 'compose.yml must contain a top-level map');
  }

  const meta = compose['x-yantr'];
  checkMetadata(appId, meta);
  checkServices(appId, compose.services);
  checkPorts(appId, meta.ports, compose.services);
  checkDevelopers(appId, meta.developers);
  checkEnvGenerators(appId, meta.env_generators, source);
}

// ─── Runner ──────────────────────────────────────────────────────────────────

function displayComposePath(appId) {
  const composePath = path.join(APPS_DIR, appId, 'compose.yml');
  const relative = path.relative(process.cwd(), composePath);
  return relative !== '' && !relative.startsWith('..') ? relative : composePath;
}

function formatError(error) {
  if (error instanceof AppError) {
    return `app "${error.appId}" failed:\n  ${error.message}\n  ${displayComposePath(error.appId)}`;
  }
  return error?.stack || String(error);
}

function main() {
  let entries;
  try {
    entries = fs.readdirSync(APPS_DIR, { withFileTypes: true });
  } catch (error) {
    console.error(`Cannot read apps directory at ${APPS_DIR}: ${error.message}`);
    process.exit(1);
  }

  const appIds = entries
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();
  const strayFiles = entries.filter((entry) => !entry.isDirectory() && !entry.name.startsWith('.'));

  if (strayFiles.length > 0) {
    console.error(`${APPS_DIR} must contain only app folders; found ${strayFiles.map((entry) => entry.name).join(', ')}`);
    process.exit(1);
  }
  if (appIds.length === 0) {
    console.error(`No apps found in ${APPS_DIR}`);
    process.exit(1);
  }

  const width = String(appIds.length).length;
  console.log(`Checking ${appIds.length} app(s) in ${APPS_DIR}${STRICT ? ' (strict)' : ''}\n`);

  for (const [index, appId] of appIds.entries()) {
    const warningsBefore = warnings.length;
    try {
      checkApp(appId);
    } catch (error) {
      console.error(`\n✖ ${formatError(error)}`);
      process.exit(1);
    }
    const added = warnings.length - warningsBefore;
    const suffix = added > 0 ? ` — ${added} warning${added === 1 ? '' : 's'}` : '';
    console.log(`[${String(index + 1).padStart(width)}/${appIds.length}] ${appId} ✓${suffix}`);
  }

  console.log(`\nAll ${appIds.length} app(s) passed.`);

  if (warnings.length > 0) {
    console.log(`\n${warnings.length} warning(s) — existing catalog debt, not hard failures:`);
    for (const warning of warnings) {
      console.log(`  ! ${warning.appId}: ${warning.message}`);
    }
    console.log('\nRun with --strict to treat warnings as errors.');
  }
}

main();
