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
import { readFile, readdir, stat } from "fs/promises";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const APPS_DIR = path.join(__dirname, "..", "apps");

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

// ── SVG helpers ────────────────────────────────────────────────────────────────

function parseSvgDimensions(svgContent) {
  // Try width/height attributes first: width="256" height="256"
  const widthMatch = svgContent.match(/\bwidth="(\d+(?:\.\d+)?)"/);
  const heightMatch = svgContent.match(/\bheight="(\d+(?:\.\d+)?)"/);
  if (widthMatch && heightMatch) {
    return { width: Number(widthMatch[1]), height: Number(heightMatch[1]) };
  }
  // Fall back to viewBox="0 0 W H"
  const viewBoxMatch = svgContent.match(/\bviewBox="[\s]*(\-?[\d.]+)[\s,]+(\-?[\d.]+)[\s,]+(\d+(?:\.\d+)?)[\s,]+(\d+(?:\.\d+)?)\s*"/);
  if (viewBoxMatch) {
    return { width: Number(viewBoxMatch[3]), height: Number(viewBoxMatch[4]) };
  }
  return null;
}

async function checkLogoSvg(appName, appDir) {
  const svgPath = path.join(appDir, "logo.svg");
  let svgStat;
  try {
    svgStat = await stat(svgPath);
    if (!svgStat.isFile()) return null;
  } catch {
    return null; // no logo.svg
  }

  const content = await readFile(svgPath, "utf-8");

  // Must be an SVG
  if (!content.includes("<svg")) {
    fail(appName, 'logo.svg is not a valid SVG file', "File must be an SVG image.");
  }

  const dims = parseSvgDimensions(content);
  if (!dims) {
    fail(appName, 'logo.svg is missing width/height or viewBox', 'Add width and height attributes or a viewBox to the <svg> tag.');
  }

  // Must be square
  if (dims.width !== dims.height) {
    fail(appName, `logo.svg must be square (${dims.width}x${dims.height})`, "Width and height must be equal.");
  }

  // Minimum 256x256
  if (dims.width < 256 || dims.height < 256) {
    fail(appName, `logo.svg is too small (${dims.width}x${dims.height})`, "Minimum dimensions are 256x256.");
  }

  return svgPath;
}

// ── x-yantr block rules ────────────────────────────────────────────────────────

function checkXYantr(appName, meta, hasLogoSvg) {
  // name
  if (!meta.name || typeof meta.name !== "string" || !meta.name.trim()) {
    fail(appName, 'x-yantr missing "name"', "Must be a non-empty string.");
  }

  // logo — IPFS CID (optional); logo.svg in the app folder is auto-detected
  if (typeof meta.logo === "string" && meta.logo.trim()) {
    const logo = meta.logo.trim();
    if (hasLogoSvg) {
      warn(appName, '"logo" field is set but logo.svg exists in the folder', 'Remove the "logo" field — logo.svg is auto-detected.');
    } else if (logo.includes("://")) {
      warn(appName, '"logo" looks like a URL', `Should be an IPFS CID, got: "${logo}"`);
    } else if (!/^Qm[a-zA-Z0-9]{44}$/.test(logo) && !/^baf[a-zA-Z0-9]+$/.test(logo)) {
      warn(appName, '"logo" does not look like a valid IPFS CID', `Got: "${logo}"`);
    }
  }

  // tags — 3–5, lowercase, letters/numbers/hyphens only
  if (!Array.isArray(meta.tags) || meta.tags.length < 3) {
    fail(appName, 'x-yantr "tags" must have at least 3 entries', `Found ${meta.tags?.length ?? 0}`);
  }
  if (meta.tags.length > 5) {
    fail(appName, 'x-yantr "tags" must have at most 5 entries', `Found ${meta.tags.length}`);
  }
  for (const tag of meta.tags) {
    if (typeof tag !== "string" || !/^[a-z0-9-]+$/.test(tag)) {
      fail(appName, `x-yantr "tags" entry is invalid: "${tag}"`, "Tags must be lowercase letters, numbers, and hyphens only.");
    }
  }

  // short_description — 50–100 chars
  if (typeof meta.short_description !== "string" || !meta.short_description.trim()) {
    fail(appName, 'x-yantr missing "short_description"', "Required field.");
  } else {
    const len = meta.short_description.trim().length;
    if (len < 50 || len > 100) {
      fail(appName, `x-yantr "short_description" length out of range (${len} chars)`, "Must be 50–100 characters.");
    }
  }

  // description — 200–300 chars
  if (typeof meta.description !== "string" || !meta.description.trim()) {
    fail(appName, 'x-yantr missing "description"', "Required field.");
  } else {
    const len = meta.description.trim().length;
    if (len < 200 || len > 300) {
      fail(appName, `x-yantr "description" length out of range (${len} chars)`, "Must be 200–300 characters.");
    }
  }

  // usecases — min 2
  if (!Array.isArray(meta.usecases) || meta.usecases.length < 2) {
    fail(appName, 'x-yantr "usecases" must have at least 2 entries', `Found ${meta.usecases?.length ?? 0}`);
  }

  // website — must be https://
  if (!meta.website || typeof meta.website !== "string") {
    fail(appName, 'x-yantr missing "website"', "Must be a valid https:// URL.");
  } else if (!meta.website.startsWith("https://")) {
    fail(appName, `x-yantr "website" must start with https://`, `Got: "${meta.website}"`);
  }

  // notes — if present must be array of strings
  if ("notes" in meta) {
    if (!Array.isArray(meta.notes)) {
      fail(appName, 'x-yantr "notes" must be an array of strings', "Remove the field or use an array.");
    }
    for (const note of meta.notes) {
      if (typeof note !== "string") {
        fail(appName, 'x-yantr "notes" contains a non-string entry', JSON.stringify(note));
      }
    }
  }
}

// ── compose.yml rules ──────────────────────────────────────────────────────────

async function checkCompose(appName, composePath) {
  let raw;
  try {
    raw = await readFile(composePath, "utf-8");
  } catch {
    fail(appName, "compose.yml missing", `Expected at ${composePath}`);
  }

  let compose;
  let doc;
  try {
    compose = YAML.parse(raw);
    doc = YAML.parseDocument(raw);
  } catch (e) {
    fail(appName, "compose.yml is not valid YAML", e.message);
  }

  const meta = compose?.["x-yantr"];
  if (!meta || typeof meta !== "object" || !meta.name) {
    fail(appName, "compose.yml missing x-yantr block", 'Add an "x-yantr:" top-level key with name, tags, description, etc.');
  }

  // Enforce compact flow sequences for tags, usecases, and notes
  if (doc && doc.contents && doc.contents.items) {
    const xYantrItem = doc.contents.items.find(item => item.key && item.key.value === 'x-yantr');
    if (xYantrItem && xYantrItem.value && xYantrItem.value.items) {
      for (const yantrProp of xYantrItem.value.items) {
        if (['tags', 'usecases', 'notes'].includes(yantrProp.key.value) && yantrProp.value && yantrProp.value.items) {
          if (!yantrProp.value.flow) {
            fail(appName, `x-yantr "${yantrProp.key.value}" must use compact flow sequence`, `Use [item1, item2] format instead of block sequence (- item1 \\n - item2).`);
          }
        }
      }
    }
  }

  const appDir = path.dirname(composePath);
  const hasLogoSvg = !!(await checkLogoSvg(appName, appDir));

  checkXYantr(appName, meta, hasLogoSvg);

  const services = compose?.services ?? {};
  if (Object.keys(services).length === 0) {
    fail(appName, "compose.yml has no services defined");
  }

  for (const [svcName, svc] of Object.entries(services)) {
    if (!svc || typeof svc !== "object") continue;
    if (Array.isArray(svc.labels)) {
      fail(
        appName,
        `compose.yml service "${svcName}" defines labels as an array`,
        'Define labels as a map (key-value dictionary). Example:\n      labels:\n        yantr.service.80: "Web UI"\n        yantr.port.80: "HTTP"'
      );
    }

    const labels = normaliseLabels(svc.labels);

    if (!labels["yantr.app"] || labels["yantr.app"] !== appName) {
      fail(
        appName,
        `compose.yml service "${svcName}" is missing or has incorrect yantr.app label`,
        `Add exactly: yantr.app: "${appName}"`
      );
    }

    // yantr.service.{PORT} is required for each port — skip for services with no exposed ports
    const hasExposedPorts = Array.isArray(svc.ports) && svc.ports.length > 0;
    if (hasExposedPorts) {
      const hasPortServiceLabel = Object.keys(labels).some((k) => /^yantr\.service\.\d+$/.test(k));
      if (!hasPortServiceLabel) {
        fail(
          appName,
          `compose.yml service "${svcName}" is missing a service label`,
          'Add a per-port label like yantr.service.8080: "Web UI" for each port'
        );
      }

      // at least one yantr.port.N label required
      const portLabels = Object.keys(labels).filter((k) => k.startsWith("yantr.port."));
      if (portLabels.length === 0) {
        fail(
          appName,
          `compose.yml service "${svcName}" has no port labels`,
          'Add at least one "yantr.port.{N}: PROTOCOL" label, e.g. yantr.port.8080: "HTTP"'
        );
      }

      // validate each yantr.port.N label
      for (const key of portLabels) {
        const portNum = key.replace("yantr.port.", "");
        if (!/^\d+$/.test(portNum)) {
          fail(appName, `compose.yml service "${svcName}" has invalid port label key: ${key}`, "Port must be a number, e.g. yantr.port.8080");
        }
        const protocol = labels[key];
        if (!["HTTP", "HTTPS", "TCP", "UDP"].includes(protocol?.toUpperCase())) {
          fail(
            appName,
            `compose.yml service "${svcName}" has invalid protocol "${protocol}" on label ${key}`,
            'Allowed values: HTTP, HTTPS, TCP, UDP'
          );
        }
      }
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
      const envGenerators = meta.env_generators ?? {};
      for (const [, val] of Object.entries(svc.environment)) {
        if (typeof val === "string") {
          const reqMatch = val.match(/^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$/);
          if (reqMatch) {
            const varName = reqMatch[1];
            const systemVars = ["TZ", "PUID", "PGID", "TUNNEL_TOKEN", "TAILSCALE_AUTH_KEY", "TELEGRAM_BOT_TOKEN", "NOSTR_NSEC", "AUTHCODE"];
            if (!systemVars.includes(varName) && !envGenerators[varName]) {
              fail(
                appName,
                `compose.yml requires secret/variable "${varName}" without a default, but it's missing from x-yantr env_generators`,
                `Add an env_generators entry for "${varName}" in the x-yantr block so Yantr can securely generate it.`
              );
            }
          }
        }
      }
    }

    // Ports must use container-only format
    for (const entry of svc.ports ?? []) {
      const spec = typeof entry === "string" ? entry : entry?.target ? String(entry.target) : null;
      if (!spec) continue;
      const noProto = spec.split("/")[0];
      const stripped = noProto.replace(/^\[.*?\]:/, "");
      if (stripped.includes(":")) {
        warn(
          appName,
          `compose.yml service "${svcName}" uses fixed host port mapping: "${spec}"`,
          'Use container-only format (e.g. "8080") so Docker auto-assigns the host port.',
        );
      }
    }
  }

  return compose;
}

function normaliseLabels(raw) {
  if (!raw) return {};
  return typeof raw === "object" ? raw : {};
}

// ── Port conflict detection ────────────────────────────────────────────────────

function extractPublishedPorts(compose) {
  const ports = new Set();
  for (const svc of Object.values(compose?.services ?? {})) {
    for (const entry of svc?.ports ?? []) {
      if (typeof entry === "string") {
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

const portMap = new Map();

for (const appName of apps) {
  const appDir = path.join(APPS_DIR, appName);
  const composePath = path.join(appDir, "compose.yml");

  const compose = await checkCompose(appName, composePath);

  for (const port of extractPublishedPorts(compose)) {
    if (!portMap.has(port)) portMap.set(port, []);
    portMap.get(port).push(appName);
  }
}

// Port conflict check
for (const [port, owners] of portMap.entries()) {
  if (owners.length > 1) {
    warn("port-conflict", `Port :${port} is used by multiple apps: ${owners.join(", ")}`, "These apps cannot run simultaneously on the same host.");
  }
}

console.log(`\n✅  All ${apps.length} apps passed validation.`);
