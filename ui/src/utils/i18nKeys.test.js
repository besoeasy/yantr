import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

// Keys defined in en.json must be referenced somewhere in the source. Dead
// keys are worse than no keys: they get faithfully translated into all seven
// locales forever, and every redesign leaves another orphan block behind
// (the last sweep removed 186 of them, whole removed features among them).
// This is the same guard cssClasses.test.js runs for the utility layer.

const SRC = fileURLToPath(new URL("..", import.meta.url));
const en = JSON.parse(readFileSync(join(SRC, "i18n/locales/en.json"), "utf8"));

function walk(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) {
      if (["node_modules", "dist", "i18n"].includes(entry)) continue;
      walk(path, out);
    } else if (/\.(vue|js)$/.test(path) && !/\.test\.js$/.test(path)) {
      out.push(path);
    }
  }
  return out;
}

const used = new Set();
for (const file of walk(SRC)) {
  const text = readFileSync(file, "utf8");
  // Static keys: t('...') and t("..."), including $t and i18n.global.t.
  for (const m of text.matchAll(/(?:^|[^\w.$])t\(\s*(["'])([^"'\n]+)\1/g)) {
    used.add(m[2].trim());
  }
  // Backtick keys without interpolation are static too.
  for (const m of text.matchAll(/(?:^|[^\w.$])t\(\s*`([^`$]+)`/g)) {
    used.add(m[1].trim());
  }
}

// Keys that reach t() through variables, resolved by hand against their call
// sites; the scan above cannot follow them:
// - utils/routes.js NAV_ITEMS labelKey strings, consumed in App.vue as
//   t(item.labelKey);
// - views/ContainerDetail.vue lifecycle labels map (start/stop/restart pairs);
// - Widgets/BiggestUnusedCard.vue confirm ternary;
// - views/DeveloperProfile.vue calls usePaymentRails(profile, "developer.payments"),
//   and the composable interpolates `${prefix}.lightning` & friends.
const HAND_REFERENCED = [
  "nav.dashboard", "nav.apps", "nav.storage", "nav.logs",
  "quickMetrics.biggestUnused.confirmImage", "quickMetrics.biggestUnused.confirmVolume",
  "container.starting", "container.startFailed", "container.stopping",
  "container.stopFailed", "container.restarting", "container.restartFailed",
  "developer.payments.lightning", "developer.payments.onchain",
  "developer.payments.qrLabel", "developer.payments.onchainQrLabel",
];
for (const key of HAND_REFERENCED) used.add(key);

function flatten(obj, prefix = "", out = []) {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === "object" && !Array.isArray(v)) flatten(v, key, out);
    else out.push(key);
  }
  return out;
}

describe("i18n key usage", () => {
  it("defines no key that no component or composable references", () => {
    const unused = flatten(en).filter((key) => !used.has(key));
    // The message is the sweep instructions: delete the keys, then the guard
    // passes again — never add to this list to silence it.
    expect(
      unused,
      `unused keys in en.json (${unused.length}): delete them from every locale`,
    ).toEqual([]);
  });

  it("defines no key en.json lacks — en is the source of truth", () => {
    const enKeys = new Set(flatten(en));
    const localesDir = join(SRC, "i18n/locales");
    for (const file of readdirSync(localesDir)) {
      if (file === "en.json") continue;
      const tree = JSON.parse(readFileSync(join(localesDir, file), "utf8"));
      const missing = flatten(tree).filter((key) => !enKeys.has(key));
      expect(
        missing,
        `${file} defines keys en.json does not — en is the source of truth`,
      ).toEqual([]);
      // Coverage the other way is checked by reading: a locale missing a key
      // falls back to en at runtime, which is visible but survivable; a locale
      // inventing its own keys silently drifts the surface apart.
    }
  });
});
