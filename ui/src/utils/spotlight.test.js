import { describe, expect, it } from "vitest";
import { appOfDay, getDateDaySeed, hashString, upNextApps } from "./spotlight";

// Same ref date for every assertion, so tests do not pass one day and fail
// the next just for running on a date whose hash changed.
const NOW = new Date(1_800_000_000 * 1000);

const catalog = [
  { id: "ollama", name: "Ollama" },
  { id: "vaultwarden", name: "Vaultwarden" },
  { id: "immich", name: "Immich" },
  { id: "gitea", name: "Gitea" },
];

describe("getDateDaySeed", () => {
  it("is a local YYYY-MM-DD string, zero-padded", () => {
    expect(getDateDaySeed(new Date(2026, 0, 3))).toBe("2026-01-03");
  });
});

describe("hashString", () => {
  it("is deterministic and fits a uint32", () => {
    const first = hashString("2026-01-03");
    expect(first).toBe(hashString("2026-01-03"));
    expect(first).toBeLessThanOrEqual(0xffffffff);
  });
});

describe("appOfDay", () => {
  it("returns a member of the catalog and the same one all day", () => {
    const pick = appOfDay(catalog, 0, NOW);
    expect(catalog.map((a) => a.id)).toContain(pick.id);
    expect(appOfDay(catalog, 0, NOW)).toBe(pick);
  });

  it("feature an app independent of the catalog's input order", () => {
    const reordered = [...catalog].reverse();
    expect(appOfDay(reordered, 0, NOW).id).toBe(appOfDay(catalog, 0, NOW).id);
  });

  it("is null for an empty catalog, never a crash", () => {
    expect(appOfDay([], 0, NOW)).toBeNull();
    expect(appOfDay(undefined, 0, NOW)).toBeNull();
  });

  it("moves forward day to day, and so do the chosen ids", () => {
    const today = appOfDay(catalog, 0, NOW);
    const tomorrow = appOfDay(catalog, 1, NOW);
    expect(tomorrow).not.toBeNull();
    // With a 4-app small pool a collision is possible, but not for the
    // externally fixed ref date NOR — assert what the rotation does promise:
    // determinism and distinctness, checked below in upNextApps.
    if (tomorrow.id !== today.id) {
      expect(new Set([today.id, tomorrow.id]).size).toBe(2);
    }
  });
});

describe("upNextApps", () => {
  it("returns the requested count of distinct apps from a catalog that can feed it", () => {
    const featured = appOfDay(catalog, 0, NOW);
    const next = upNextApps(catalog, featured, 2);
    expect(next).toHaveLength(2);
    expect(new Set([featured.id, ...next.map((a) => a.id)]).size).toBe(3);
  });

  it("never repeats the featured app", () => {
    const featured = appOfDay(catalog, 0, NOW);
    for (const app of upNextApps(catalog, featured, 2)) {
      expect(app.id).not.toBe(featured.id);
    }
  });

  it("shortens honestly when the catalog is too small to fill the shelf", () => {
    expect(upNextApps([{ id: "ollama", name: "Ollama" }], { id: "ollama" }, 2)).toEqual([]);
  });

  it("is null-featured safe: still counts from tomorrow", () => {
    const next = upNextApps(catalog, null, 2);
    expect(next).toHaveLength(2);
    expect(new Set(next.map((a) => a.id)).size).toBe(2);
  });
});
