#!/usr/bin/env node
/**
 * Yantr App Validator
 * Checks every app under apps/ alphabetically.
 * Exits immediately on the first broken rule, reporting the app and rule.
 *
 * Usage: node check.js
 */

import path from "path";
import YAML from "yaml";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { readFile, readdir } from "fs/promises";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const APPS_DIR = path.join(__dirname, "apps");

// ── Helpers ────────────────────────────────────────────────────────────────────

function fail(appName, rule, detail) {
  console.error(`\n❌  [${appName}] Rule broken: ${rule}`);
  if (detail) console.error(`   ${detail}`);
  process.exit(1);
}

function warn(appName, rule, detail) {
  console.warn(`⚠️   [${appName}] Warning: ${rule}`);
  if (detail) console.warn(`   ${detail}`);
}

// ── info.json rules ────────────────────────────────────────────────────────────

async function checkInfoJson(appName, infoPath) {
  let raw;
  try {
    raw = await readFile(infoPath, "utf-8");
  } catch {
    fail(appName, "info.json missing", `Expected at ${infoPath}`);
  }

  let info;
  try {
    info = JSON.parse(raw);
  } catch (e) {
    fail(appName, "info.json is not valid JSON", e.message);
  }

  // name
  if (!info.name || typeof info.name !== "string" || !info.name.trim()) {
    fail(appName, 'info.json missing "name"', "Must be a non-empty string with correct capitalisation.");
  }

  // logo — optional IPFS CID; empty or missing is allowed
  if (typeof info.logo === "string" && info.logo.trim()) {
    const logo = info.logo.trim();
    if (logo.includes("://")) {
      warn(appName, '"logo" looks like a URL', `Should be an IPFS CID, got: "${logo}"`);
    } else if (!/^Qm[a-zA-Z0-9]{44}$/.test(logo) && !/^baf[a-zA-Z0-9]+$/.test(logo)) {
      warn(appName, '"logo" does not look like a valid IPFS CID', `Got: "${logo}"`);
    }
  }

  // tags — min 6, lowercase, letters/numbers/hyphens only
  if (!Array.isArray(info.tags) || info.tags.length < 6) {
    fail(appName, 'info.json "tags" must have at least 6 entries', `Found ${info.tags?.length ?? 0}`);
  }
  for (const tag of info.tags) {
    if (typeof tag !== "string" || !/^[a-z0-9-]+$/.test(tag)) {
      fail(appName, `info.json "tags" entry is invalid: "${tag}"`, "Tags must be lowercase letters, numbers, and hyphens only.");
    }
  }

  // short_description — 50–100 chars
  if (typeof info.short_description !== "string" || !info.short_description.trim()) {
    fail(appName, 'info.json missing "short_description"', "Required field.");
  } else {
    const len = info.short_description.trim().length;
    if (len < 50 || len > 100) {
      fail(appName, `info.json "short_description" length out of range (${len} chars)`, "Must be 50–100 characters.");
    }
  }

  // description — 200–300 chars
  if (typeof info.description !== "string" || !info.description.trim()) {
    fail(appName, 'info.json missing "description"', "Required field.");
  } else {
    const len = info.description.trim().length;
    if (len < 200 || len > 300) {
      fail(appName, `info.json "description" length out of range (${len} chars)`, "Must be 200–300 characters.");
    }
  }

  // usecases — min 2
  if (!Array.isArray(info.usecases) || info.usecases.length < 2) {
    fail(appName, 'info.json "usecases" must have at least 2 entries', `Found ${info.usecases?.length ?? 0}`);
  }

  // ports — must be array format, no old "port" string
  if ("port" in info) {
    fail(appName, 'info.json uses deprecated "port" string field', 'Replace with "ports" array: [{ "port": 8080, "protocol": "HTTP", "label": "Web UI" }]');
  }
  if (!Array.isArray(info.ports)) {
    fail(appName, 'info.json missing "ports" array', 'Required field. Use format: [{ "port": 8080, "protocol": "HTTP", "label": "Web UI" }]');
  }
  for (const p of info.ports) {
    if (typeof p.port !== "number") {
      fail(appName, `info.json "ports" entry has invalid "port" value: ${JSON.stringify(p.port)}`, "Must be a number.");
    }
    if (typeof p.protocol !== "string" || !p.protocol.trim()) {
      fail(appName, `info.json "ports" entry missing "protocol" for port ${p.port}`, 'e.g. "HTTP", "TCP", "UDP"');
    }
    if (typeof p.label !== "string" || !p.label.trim()) {
      fail(appName, `info.json "ports" entry missing "label" for port ${p.port}`, 'e.g. "Web UI"');
    }
  }

  // website — must be https://
  if (!info.website || typeof info.website !== "string") {
    fail(appName, 'info.json missing "website"', "Must be a valid https:// URL.");
  } else if (!info.website.startsWith("https://")) {
    fail(appName, `info.json "website" must start with https://`, `Got: "${info.website}"`);
  }

  // dependencies — removed; app metadata should stay self-contained
  if ("dependencies" in info) {
    fail(appName, 'info.json uses removed "dependencies" field', 'Remove the field and describe any manual app-to-app wiring in "notes".');
  }

  // notes — if present must be array of strings
  if ("notes" in info) {
    if (!Array.isArray(info.notes)) {
      fail(appName, 'info.json "notes" must be an array of strings', "Remove the field or use an array.");
    }
    for (const note of info.notes) {
      if (typeof note !== "string") {
        fail(appName, 'info.json "notes" contains a non-string entry', JSON.stringify(note));
      }
    }
  }

  return info;
}

// ── compose.yml rules ──────────────────────────────────────────────────────────

async function checkCompose(appName, composePath, info) {
  let raw;
  try {
    raw = await readFile(composePath, "utf-8");
  } catch {
    fail(appName, "compose.yml missing", `Expected at ${composePath}`);
  }

  let compose;
  try {
    compose = YAML.parse(raw);
  } catch (e) {
    fail(appName, "compose.yml is not valid YAML", e.message);
  }

  const services = compose?.services ?? {};
  if (Object.keys(services).length === 0) {
    fail(appName, "compose.yml has no services defined");
  }

  for (const [svcName, svc] of Object.entries(services)) {
    if (!svc || typeof svc !== "object") continue;

    // Required yantr labels
    const labels = normaliseLabels(svc.labels);
    for (const required of ["yantr.app", "yantr.service"]) {
      if (!labels[required]) {
        fail(appName, `compose.yml service "${svcName}" is missing required label: ${required}`);
      }
    }

    if (labels["yantr.app"] && labels["yantr.app"] !== appName) {
      warn(appName, `Service "${svcName}" yantr.app mismatch`, `yantr.app="${labels["yantr.app"]}" but folder is "${appName}"`);
    }

    // Environment must be key-value object, not list
    if (Array.isArray(svc.environment)) {
      fail(
        appName,
        `compose.yml service "${svcName}" uses list format for environment variables`,
        'Use key-value format: "VAR: ${VAR:-default}" not "- VAR=${VAR:-default}"',
      );
    } else if (svc.environment && typeof svc.environment === "object") {
      // Enforce env_generators for required variables
      for (const [key, val] of Object.entries(svc.environment)) {
        if (typeof val === "string") {
          // Look for strictly required variables like ${ADMIN_PASSWORD} (no :- fallback)
          const reqMatch = val.match(/^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$/);
          if (reqMatch) {
            const varName = reqMatch[1];
            // Some common vars are system-provided or external tokens that users must provide manually
            const systemVars = ["TZ", "PUID", "PGID", "TUNNEL_TOKEN", "TAILSCALE_AUTH_KEY", "TELEGRAM_BOT_TOKEN", "NOSTR_NSEC", "AUTHCODE"];
            if (!systemVars.includes(varName) && !info?.env_generators?.[varName] && !info?.envGenerators?.[varName]) {
              fail(
                appName,
                `compose.yml requires secret/variable "${varName}" without a default, but it's missing from info.json env_generators`,
                `Add an env_generators entry for "${varName}" in info.json so Yantr can securely generate it.`
              );
            }
          }
        }
      }
    }

    // Ports must use container-only format — no fixed host:container bindings
    for (const entry of svc.ports ?? []) {
      const spec = typeof entry === "string" ? entry : entry?.target ? String(entry.target) : null;
      if (!spec) continue;

      // Strip protocol suffix (e.g. "53:53/tcp" → "53:53")
      const noProto = spec.split("/")[0];

      // Handle IPv6 bracket notation like "[::]:80:80"
      const stripped = noProto.replace(/^\[.*?\]:/, "");

      if (stripped.includes(":")) {
        warn(
          appName,
          `compose.yml service "${svcName}" uses fixed host port mapping: "${spec}"`,
          'Use container-only format (e.g. "53/tcp") so Docker auto-assigns the host port and multiple instances can run.',
        );
      }
    }
  }

  return compose;
}

function normaliseLabels(raw) {
  if (!raw) return {};
  if (Array.isArray(raw)) {
    const out = {};
    for (const l of raw) {
      if (typeof l !== "string") continue;
      const idx = l.indexOf("=");
      if (idx === -1) {
        out[l] = "";
      } else {
        out[l.slice(0, idx)] = l.slice(idx + 1);
      }
    }
    return out;
  }
  return typeof raw === "object" ? raw : {};
}

// ── Port conflict detection ────────────────────────────────────────────────────

function extractPublishedPorts(compose) {
  const ports = new Set();
  for (const svc of Object.values(compose?.services ?? {})) {
    for (const entry of svc?.ports ?? []) {
      if (typeof entry === "string") {
        // "hostPort:containerPort" or "containerPort" etc.
        const noProto = entry.split("/")[0];
        const parts = noProto.split(":").filter(Boolean);
        if (parts.length >= 2) {
          const host = parts[parts.length - 2];
          if (/^\d+$/.test(host)) ports.add(Number(host));
        }
      } else if (entry && typeof entry === "object" && typeof entry.published === "number") {
        ports.add(entry.published);
      }
    }
  }
  return ports;
}

// ── Main ───────────────────────────────────────────────────────────────────────

const entries = await readdir(APPS_DIR, { withFileTypes: true });
const apps = entries
  .filter((e) => e.isDirectory())
  .map((e) => e.name)
  .sort();

console.log(`🔍  Yantr App Validator — checking ${apps.length} apps alphabetically\n`);

const portMap = new Map(); // port → appName[]
let allPassed = true;

for (const appName of apps) {
  const appDir = path.join(APPS_DIR, appName);
  const infoPath = path.join(appDir, "info.json");
  const composePath = path.join(appDir, "compose.yml");

  const info = await checkInfoJson(appName, infoPath);
  const compose = await checkCompose(appName, composePath, info);

  // Track published ports for conflict detection
  for (const port of extractPublishedPorts(compose)) {
    if (!portMap.has(port)) portMap.set(port, []);
    portMap.get(port).push(appName);
  }
}

// Port conflict check (after all apps so we can report both sides)
for (const [port, owners] of portMap.entries()) {
  if (owners.length > 1) {
    warn("port-conflict", `Port :${port} is used by multiple apps: ${owners.join(", ")}`, "These apps cannot run simultaneously on the same host.");
  }
}

console.log(`\n✅  All ${apps.length} apps passed validation.`);
