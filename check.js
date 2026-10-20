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
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { parse } from 'yaml';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const APPS_DIR = process.env.YANTR_APPS_DIR
  ? path.resolve(process.env.YANTR_APPS_DIR)
  : path.join(HERE, 'apps');

const USAGE = `Usage: node check.js [options]

Options:
  --app <name>     check only this app (repeatable)
  --changed        check apps touched by uncommitted changes
  --since <ref>    check apps changed since a git ref
  --strict         treat warnings as errors
  -h, --help       show this help

YANTR_APPS_DIR overrides the apps directory (default: ./apps).`;

function usageError(message) {
  console.error(`${message}\n\n${USAGE}`);
  process.exit(2);
}

function parseArgs(argv) {
  const options = { apps: [], changed: false, since: null, strict: false };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];

    if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--changed') {
      options.changed = true;
    } else if (arg === '--app' || arg.startsWith('--app=')) {
      const value = arg === '--app' ? argv[index + 1] : arg.slice('--app='.length);
      if (!value || value.startsWith('-')) {
        usageError('--app requires an app name');
      }
      if (arg === '--app') index += 1;
      options.apps.push(value);
    } else if (arg === '--since' || arg.startsWith('--since=')) {
      const value = arg === '--since' ? argv[index + 1] : arg.slice('--since='.length);
      if (!value || value.startsWith('-')) {
        usageError('--since requires a git ref');
      }
      if (arg === '--since') index += 1;
      options.since = value;
    } else if (arg === '--help' || arg === '-h') {
      console.log(USAGE);
      process.exit(0);
    } else {
      usageError(`Unknown option "${arg}"`);
    }
  }

  if (options.changed && options.since !== null) {
    usageError('--changed and --since cannot be combined');
  }
  if (options.apps.length > 0 && (options.changed || options.since !== null)) {
    usageError('--app cannot be combined with --changed or --since');
  }

  return options;
}

const OPTIONS = parseArgs(process.argv.slice(2));
const STRICT = OPTIONS.strict;

const PORT_PROTOCOLS = new Set(['HTTP', 'HTTPS', 'TCP', 'UDP']);
const ACCESS_PROTOCOLS = new Set(['HTTP', 'HTTPS']);
const ENV_CHARSETS = new Set(['alnum', 'hex', 'numeric', 'alpha', 'base64url', 'alnum_symbols']);
const APP_FILES = new Set(['compose.yml', 'logo.svg']);
const HOST_SOCKET = '${HOST_PODMAN_SOCKET}';
const CONTAINER_SOCKET = '/var/run/docker.sock';

// The whole x-yantr vocabulary. Anything else is a typo that the loader
// would silently ignore, so it is reported instead of deploying without it.
const X_YANTR_KEYS = new Set([
  'name',
  'tags',
  'short_description',
  'description',
  'usecases',
  'website',
  'notes',
  'developers',
  'env_generators',
  'ports',
]);

const TAG_RANGE = { min: 3, max: 5 };
const MAX_USECASES = 5;
const MAX_NOTES = 6;

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
// `extra` tags the warning so the summary can group a whole class of them.
function warn(appId, message, extra = {}) {
  if (STRICT) fail(appId, message);
  warnings.push({ appId, message, ...extra });
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

// ─── Logo ────────────────────────────────────────────────────────────────────

// An SVG scales, so its viewBox decides the shape the dashboard renders. A
// missing viewBox/width/height or a non-square pair is warning-level: the logo
// still displays, but stretched or cropped.
function checkLogo(appId, appDir) {
  const logoPath = path.join(appDir, 'logo.svg');
  if (!fs.existsSync(logoPath)) return;

  let source;
  try {
    source = fs.readFileSync(logoPath, 'utf8');
  } catch (error) {
    fail(appId, `cannot read logo.svg: ${error.message}`);
  }

  const root = source.match(/<svg\b[^>]*>/i);
  if (!root) {
    fail(appId, 'logo.svg has no <svg> root element');
  }

  const attribute = (name) => {
    const match = root[0].match(new RegExp(`\\b${name}\\s*=\\s*["']([^"']*)["']`, 'i'));
    return match ? match[1].trim() : null;
  };

  const sizes = [];
  const viewBox = attribute('viewBox');
  if (viewBox !== null) {
    const parts = viewBox.split(/[\s,]+/).map(Number);
    if (parts.length !== 4 || parts.some((part) => !Number.isFinite(part))) {
      warn(appId, `logo.svg viewBox "${viewBox}" is not four numbers; its shape cannot be read`);
    } else {
      sizes.push({ source: 'viewBox', width: parts[2], height: parts[3] });
    }
  }

  const width = attribute('width');
  const height = attribute('height');
  if (width !== null && height !== null) {
    const parsedWidth = parseFloat(width);
    const parsedHeight = parseFloat(height);
    if (Number.isFinite(parsedWidth) && Number.isFinite(parsedHeight)) {
      sizes.push({ source: 'width/height', width: parsedWidth, height: parsedHeight });
    }
  }

  if (sizes.length === 0) {
    warn(appId, 'logo.svg declares no viewBox or width/height; its shape cannot be checked');
    return;
  }

  for (const size of sizes) {
    if (size.width <= 0 || size.height <= 0) {
      warn(appId, `logo.svg ${size.source} is not a positive size (${size.width}×${size.height})`);
    } else if (Math.abs(size.width - size.height) > 0.5) {
      warn(appId, `logo.svg ${size.source} is ${size.width}×${size.height}; logos are square`);
    }
  }
}

// ─── x-yantr metadata ────────────────────────────────────────────────────────

function checkMetadata(appId, meta, tagCounts) {
  if (!isObject(meta)) {
    fail(appId, 'compose.yml has no x-yantr block');
  }
  if (typeof meta.name !== 'string' || text(meta.name) === '') {
    fail(appId, 'x-yantr.name is required');
  }

  for (const key of Object.keys(meta)) {
    if (!X_YANTR_KEYS.has(key)) {
      warn(appId, `x-yantr.${key} is not a known key; it is ignored at deploy time`);
    }
  }

  if (!Array.isArray(meta.tags) || meta.tags.length < TAG_RANGE.min || meta.tags.length > TAG_RANGE.max) {
    fail(appId, `x-yantr.tags must contain ${TAG_RANGE.min}–${TAG_RANGE.max} tags (got ${Array.isArray(meta.tags) ? meta.tags.length : 'none'})`);
  }
  for (const tag of meta.tags) {
    if (typeof tag !== 'string' || text(tag) === '') {
      fail(appId, 'x-yantr.tags contains an empty or non-string entry');
    }
  }
  const uniqueTags = new Set(meta.tags.map((tag) => tag.trim()));
  if (uniqueTags.size !== meta.tags.length) {
    fail(appId, 'x-yantr.tags contains a duplicate');
  }
  // A tag nobody else uses is usually a near-synonym of an existing one
  // ("download" next to "downloads"). It is not fatal, so the catalog can
  // settle its vocabulary over time, but every run surfaces it.
  for (const tag of uniqueTags) {
    if (tagCounts.get(tag) === 1) {
      warn(appId, `tag "${tag}" is used by no other app; prefer a shared tag unless the category is deliberate`, { kind: 'tag', tag });
    }
  }

  const shortDescription = text(meta.short_description);
  if (typeof meta.short_description !== 'string' || shortDescription.length < 50 || shortDescription.length > 100) {
    fail(appId, `x-yantr.short_description must be a 50–100 character string (got ${shortDescription.length})`);
  }

  const description = text(meta.description);
  if (typeof meta.description !== 'string' || description.length < 200 || description.length > 300) {
    fail(appId, `x-yantr.description must be a 200–300 character string (got ${description.length})`);
  }

  if (!Array.isArray(meta.usecases) || meta.usecases.length < 2 || meta.usecases.length > MAX_USECASES) {
    fail(appId, `x-yantr.usecases must contain 2–${MAX_USECASES} entries (got ${Array.isArray(meta.usecases) ? meta.usecases.length : 'none'})`);
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
    if (meta.notes.length > MAX_NOTES) {
      fail(appId, `x-yantr.notes must contain at most ${MAX_NOTES} entries (got ${meta.notes.length})`);
    }
    for (const note of meta.notes) {
      if (typeof note !== 'string' || text(note) === '') {
        fail(appId, 'x-yantr.notes contains an empty or non-string entry');
      }
    }
  }
}

// ─── Description style ───────────────────────────────────────────────────────

// metaSource returns the lines indented under the top-level x-yantr key, so a
// style rule can be checked against the recipe's source and not only its
// parsed value (which no longer remembers whether it was plain or folded).
function metaSource(source) {
  const lines = source.split(/\r?\n/);
  const start = lines.findIndex((line) => /^x-yantr:\s*(?:#.*)?$/.test(line));
  if (start === -1) return '';

  const block = [];
  for (let index = start + 1; index < lines.length; index += 1) {
    const line = lines[index];
    if (line.trim() !== '' && !line.startsWith(' ')) break;
    block.push(line);
  }
  return block.join('\n');
}

// A folded block is required because the description is one paragraph that
// the UI renders as one; a plain scalar or a literal block makes the wrapping
// show up as newlines in some clients.
function checkDescriptionStyle(appId, source) {
  if (!/^\s{2}description:[ \t]*>[-+]?[ \t]*$/m.test(metaSource(source))) {
    fail(appId, 'x-yantr.description must be a folded > block, not a plain string or | block');
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

// decodeNpub returns the canonical lowercase key when value is a valid npub,
// or null. It mirrors the part of core/nip19 that x-yantr.developers relies on:
// a bech32 checksum, a 32-byte payload, and nothing else. A typo is an error,
// not a key the catalog silently drops.
function decodeNpub(value) {
  if (typeof value !== 'string') return null;

  const key = value.trim();
  const lower = key.toLowerCase();
  if (lower !== key && lower.toUpperCase() !== key) return null;

  const separator = lower.lastIndexOf('1');
  if (separator < 1 || lower.slice(0, separator) !== 'npub') return null;
  if (separator + 7 > lower.length) return null;

  const data = [...lower.slice(separator + 1)].map((char) => BECH32_CHARSET.indexOf(char));
  if (data.some((valueIndex) => valueIndex === -1)) return null;
  if (bech32Polymod([...bech32HrpExpand('npub'), ...data]) !== 1) return null;

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
  if (bits >= 5 || ((accumulator << (8 - bits)) & 0xff) !== 0) return null;
  return bytes.length === 32 ? lower : null;
}

function checkDevelopers(appId, developers) {
  if (developers === undefined) return;
  if (!Array.isArray(developers)) {
    fail(appId, 'x-yantr.developers must be a list of npub keys');
  }

  const seen = new Map();
  for (const [index, developer] of developers.entries()) {
    const key = decodeNpub(developer);
    if (key === null) {
      fail(appId, `x-yantr.developers entry ${JSON.stringify(developer)} is not a valid npub key`);
    }
    if (seen.has(key)) {
      fail(appId, `x-yantr.developers entries ${seen.get(key) + 1} and ${index + 1} are the same account; credit it once`);
    }
    seen.set(key, index);
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

  // A variable is referenced when compose mentions it, which is what makes an
  // unused generator detectable. A variable with no generator is fine: the
  // deploy form asks the user for it (a tunnel token, an API key), while a
  // value the catalog must invent is what a generator is for.
  const referenced = new Set();
  const envRefPattern = /(?<!\$)\$\{([A-Za-z_][A-Za-z0-9_]*)([^}]*)\}/g;
  for (const match of composeSource.matchAll(envRefPattern)) {
    const name = match[1];
    if (name === 'HOST_PODMAN_SOCKET') continue;
    referenced.add(name);
  }

  for (const name of Object.keys(generators)) {
    if (!referenced.has(name)) {
      warn(appId, `x-yantr.env_generators.${name} is never referenced in compose.yml`);
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
  const seenPorts = new Map();
  for (const [index, port] of ports.entries()) {
    const at = `x-yantr.ports[${index}]`;

    if (!isObject(port)) {
      fail(appId, `${at} must be a map`);
    }
    if (!Number.isInteger(port.port) || port.port < 1 || port.port > 65535) {
      fail(appId, `${at}.port must be an integer between 1 and 65535`);
    }

    const protocol = text(port.protocol).toUpperCase();
    if (protocol === '') {
      fail(appId, `${at}.protocol is required (${[...PORT_PROTOCOLS].join(', ')})`);
    }
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

    const identity = `${port.port}/${protocol}/${service}`;
    if (seenPorts.has(identity)) {
      fail(appId, `${at} duplicates x-yantr.ports[${seenPorts.get(identity)}] (${identity})`);
    }
    seenPorts.set(identity, index);

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

// The tag is deliberately not judged: `:latest` and a pinned release are both
// legitimate (a database pin is usually a data-compatibility decision). Only a
// missing image is an error.
function checkImage(appId, serviceName, image) {
  if (typeof image !== 'string' || image.trim() === '') {
    fail(appId, `service "${serviceName}" has no image`);
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

// ─── App selection ───────────────────────────────────────────────────────────

// appsPrefix is the repository-relative path of APPS_DIR, with a trailing
// slash, or null when the apps directory is outside this checkout.
function appsPrefix() {
  const relative = path.relative(HERE, APPS_DIR);
  if (relative === '' || path.isAbsolute(relative) || relative.startsWith('..')) {
    return null;
  }
  return `${relative.split(path.sep).join('/')}/`;
}

function gitPaths(args) {
  const output = execFileSync('git', args, {
    cwd: HERE,
    encoding: 'utf8',
    maxBuffer: 16 * 1024 * 1024,
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  return output.split('\0').filter(Boolean);
}

function gitError(error) {
  const stderr = error.stderr ? String(error.stderr).trim() : '';
  return stderr || error.message;
}

// changedAppIds returns the existing app folders touched by uncommitted work
// (--changed) or by everything since a ref (--since). Untracked files count:
// a new app is exactly the case this mode is for.
function changedAppIds(since, knownAppIds) {
  const prefix = appsPrefix();
  if (prefix === null) {
    usageError('--changed and --since require the apps directory to be inside this repository');
  }

  let files;
  try {
    files = since !== null
      ? gitPaths(['diff', '--name-only', '-z', since])
      : gitPaths(['diff', '--name-only', '-z', 'HEAD']);
    files.push(...gitPaths(['ls-files', '--others', '--exclude-standard', '-z']));
  } catch (error) {
    if (since === null) {
      // Unborn branch: there is no HEAD to diff against yet.
      try {
        files = [
          ...gitPaths(['diff', '--name-only', '-z', '--cached']),
          ...gitPaths(['diff', '--name-only', '-z']),
          ...gitPaths(['ls-files', '--others', '--exclude-standard', '-z']),
        ];
      } catch (fallbackError) {
        usageError(`Cannot read git changes: ${gitError(fallbackError)}`);
      }
    } else {
      usageError(`Cannot diff against "${since}": ${gitError(error)}`);
    }
  }

  const touched = new Set();
  for (const file of files) {
    if (!file.startsWith(prefix)) continue;
    const appId = file.slice(prefix.length).split('/')[0];
    if (knownAppIds.has(appId)) touched.add(appId);
  }
  return [...touched].sort();
}

// ─── Catalog-wide tag frequency ──────────────────────────────────────────────

// One-off tags are judged against the whole catalog, so the frequency map is
// built before any single app is checked. A recipe that cannot be read is
// skipped here: it fails its own check with a better message.
function catalogTagCounts(appIds) {
  const counts = new Map();

  for (const appId of appIds) {
    let meta;
    try {
      const compose = parse(fs.readFileSync(path.join(APPS_DIR, appId, 'compose.yml'), 'utf8'));
      meta = isObject(compose) ? compose['x-yantr'] : null;
    } catch {
      continue;
    }
    if (!isObject(meta) || !Array.isArray(meta.tags)) continue;

    const tags = new Set(meta.tags.filter((tag) => typeof tag === 'string').map((tag) => tag.trim()));
    for (const tag of tags) {
      counts.set(tag, (counts.get(tag) ?? 0) + 1);
    }
  }

  return counts;
}

// ─── One app ─────────────────────────────────────────────────────────────────

function checkApp(appId, tagCounts) {
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
  checkMetadata(appId, meta, tagCounts);
  checkDescriptionStyle(appId, source);
  checkServices(appId, compose.services);
  checkPorts(appId, meta.ports, compose.services);
  checkDevelopers(appId, meta.developers);
  checkEnvGenerators(appId, meta.env_generators, source);
  checkLogo(appId, appDir);
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

  const knownAppIds = entries
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();
  const known = new Set(knownAppIds);

  let appIds;
  let fullRun = false;

  if (OPTIONS.apps.length > 0) {
    const unknown = [...new Set(OPTIONS.apps)].filter((appId) => !known.has(appId));
    if (unknown.length > 0) {
      usageError(`Unknown app(s): ${unknown.join(', ')}`);
    }
    appIds = [...new Set(OPTIONS.apps)].sort();
  } else if (OPTIONS.changed || OPTIONS.since !== null) {
    appIds = changedAppIds(OPTIONS.since, known);
    if (appIds.length === 0) {
      console.log(OPTIONS.since !== null
        ? `No apps changed since ${OPTIONS.since}.`
        : 'No apps changed in the working tree.');
      return;
    }
  } else {
    fullRun = true;
    appIds = knownAppIds;

    const strayFiles = entries.filter((entry) => !entry.isDirectory() && !entry.name.startsWith('.'));
    if (strayFiles.length > 0) {
      console.error(`${APPS_DIR} must contain only app folders; found ${strayFiles.map((entry) => entry.name).join(', ')}`);
      process.exit(1);
    }
  }

  if (appIds.length === 0) {
    console.error(`No apps found in ${APPS_DIR}`);
    process.exit(1);
  }

  const width = String(appIds.length).length;
  const scope = fullRun ? `${appIds.length} app(s)` : `${appIds.length} selected app(s)`;
  console.log(`Checking ${scope} in ${APPS_DIR}${STRICT ? ' (strict)' : ''}\n`);

  const tagCounts = catalogTagCounts(knownAppIds);

  for (const [index, appId] of appIds.entries()) {
    const warningsBefore = warnings.length;
    try {
      checkApp(appId, tagCounts);
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
    const general = warnings.filter((warning) => warning.kind !== 'tag');
    const oneOffTags = warnings.filter((warning) => warning.kind === 'tag');

    console.log(`\n${warnings.length} warning(s) — existing catalog debt, not hard failures:`);
    for (const warning of general) {
      console.log(`  ! ${warning.appId}: ${warning.message}`);
    }

    // Tags are one warning each and there are many, so they are listed as a
    // vocabulary to trim rather than one line per app.
    if (oneOffTags.length > 0) {
      console.log(`\n  One-off tags (${oneOffTags.length}) — each used by exactly one app;`);
      console.log('  reuse an existing tag, or keep it when the category is deliberate:');
      const pairs = oneOffTags.map((warning) => `${warning.tag} (${warning.appId})`);
      for (let index = 0; index < pairs.length; index += 3) {
        console.log(`    ${pairs.slice(index, index + 3).join(' · ')}`);
      }
    }

    console.log('\nRun with --strict to treat warnings as errors.');
  }
}

main();
