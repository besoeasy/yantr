import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fakeIndexedDB } from "./fakeIndexedDB";

// The rules a twenty-hour cache has to keep, pinned where a change to any one
// of them is otherwise invisible until a reader sees a twenty-hour-old page or
// a card that never stops shimmering.
//
// The load-bearing one is the second: a record may only be written when a relay
// actually answered. Everything else in this module exists to make that safe
// to skip a query with, and a cache that stores "the network was down" turns
// an outage into a day of empty pages.

let fake;

beforeEach(() => {
  fake = fakeIndexedDB();
  vi.stubGlobal("indexedDB", fake.api);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.resetModules();
});

// kv.js memoises its handle and schema version at module scope, so every test
// needs its own copy of the module graph.
async function freshCache() {
  vi.resetModules();
  return import("./nostrCache");
}

const ANSWERED = { answered: 1 };

describe("the twenty-hour window", () => {
  it("is twenty hours", async () => {
    const { NOSTR_QUERY_TTL_MS } = await freshCache();
    expect(NOSTR_QUERY_TTL_MS).toBe(20 * 60 * 60 * 1000);
  });

  it("expires a record written at the window's edge", async () => {
    const { writeNostrQuery, readNostrQuery } = await freshCache();
    const now = Date.now();
    vi.spyOn(Date, "now").mockReturnValue(now);

    await writeNostrQuery("app-comments", "immich", { comments: [] }, ANSWERED);
    await expect(readNostrQuery("app-comments", "immich")).resolves.toEqual({ comments: [] });

    // One millisecond before the expiry it is still an answer.
    vi.spyOn(Date, "now").mockReturnValue(now + 20 * 60 * 60 * 1000 - 1);
    await expect(readNostrQuery("app-comments", "immich")).resolves.toEqual({ comments: [] });

    // At it, it is a miss — and the surface goes back to the relays.
    vi.spyOn(Date, "now").mockReturnValue(now + 20 * 60 * 60 * 1000);
    await expect(readNostrQuery("app-comments", "immich")).resolves.toBeNull();
    vi.restoreAllMocks();
  });
});

describe("only an answer is storable", () => {
  it("refuses to write when no relay answered", async () => {
    const { writeNostrQuery, readNostrQuery } = await freshCache();

    expect(await writeNostrQuery("app-comments", "immich", { comments: [] }, { answered: 0 })).toBe(false);
    expect(await writeNostrQuery("app-comments", "immich", { comments: [] }, {})).toBe(false);
    await expect(readNostrQuery("app-comments", "immich")).resolves.toBeNull();
  });

  // Relays answering with nothing is an answer. This is what stops a thread
  // nobody has commented on being re-queried on every visit.
  it("stores an empty answer when relays did answer", async () => {
    const { writeNostrQuery, readNostrQuery } = await freshCache();
    await writeNostrQuery("app-comments", "immich", { comments: [], reached: 3 }, ANSWERED);
    await expect(readNostrQuery("app-comments", "immich")).resolves.toEqual({ comments: [], reached: 3 });
  });
});

describe("surfaces cannot collide", () => {
  it("keeps two scopes apart on the same key", async () => {
    const { writeNostrQuery, readNostrQuery } = await freshCache();
    await writeNostrQuery("app-comments", "same", { from: "thread" }, ANSWERED);
    await writeNostrQuery("review-feed", "same", { from: "feed" }, ANSWERED);

    await expect(readNostrQuery("app-comments", "same")).resolves.toEqual({ from: "thread" });
    await expect(readNostrQuery("review-feed", "same")).resolves.toEqual({ from: "feed" });
  });

  it("reads a key nobody wrote as a miss", async () => {
    const { readNostrQuery } = await freshCache();
    await expect(readNostrQuery("app-comments", "never-written")).resolves.toBeNull();
  });
});

describe("author names", () => {
  const HEX = "a".repeat(64);
  const OTHER = "b".repeat(64);

  it("round-trips a name for one public key", async () => {
    const { writeCachedProfile, readCachedProfiles } = await freshCache();
    await writeCachedProfile(HEX, { name: "besoeasy" }, ANSWERED);
    await expect(readCachedProfiles([HEX])).resolves.toEqual({ [HEX]: { name: "besoeasy" } });
  });

  it("answers for the keys it has and stays silent about the rest", async () => {
    const { writeCachedProfile, readCachedProfiles } = await freshCache();
    await writeCachedProfile(HEX, { name: "besoeasy" }, ANSWERED);
    await expect(readCachedProfiles([HEX, OTHER])).resolves.toEqual({ [HEX]: { name: "besoeasy" } });
  });

  // A miss is stored as null, which readCachedProfiles reports as "no name" —
  // the same as not cached — so the only thing that distinguishes the two is
  // how long the record survives. That is the point of the shorter window.
  it("remembers that an account has no profile — for an hour, not twenty", async () => {
    const { writeCachedProfile, NOSTR_MISS_TTL_MS, NOSTR_QUERY_TTL_MS } = await freshCache();
    const now = Date.now();
    vi.spyOn(Date, "now").mockReturnValue(now);

    await writeCachedProfile(HEX, null, ANSWERED);
    await writeCachedProfile(OTHER, { name: "besoeasy" }, ANSWERED);

    const stored = fake.state.stores.get("nostr-queries");
    expect(stored.get(`author-profile:${HEX}`).expires - now).toBe(NOSTR_MISS_TTL_MS);
    expect(stored.get(`author-profile:${OTHER}`).expires - now).toBe(NOSTR_QUERY_TTL_MS);
    vi.restoreAllMocks();
  });

  it("ignores anything that is not a public key", async () => {
    const { writeCachedProfile, readCachedProfiles } = await freshCache();
    await writeCachedProfile("not-a-key", { name: "nope" }, ANSWERED);
    await expect(readCachedProfiles(["not-a-key"])).resolves.toEqual({});
  });
});

describe("a failure is remembered, briefly", () => {
  it("stores an unanswered query for ten minutes and no longer", async () => {
    const { writeNostrFailure, readNostrQuery, NOSTR_SCOPE, NOSTR_FAILURE_TTL_MS } =
      await freshCache();
    const now = Date.now();
    vi.spyOn(Date, "now").mockReturnValue(now);

    // No `answered` here on purpose: the absence of an answer is the content.
    expect(await writeNostrFailure(NOSTR_SCOPE.reviewFeed, "latest")).toBe(true);

    const record = fake.state.stores.get("nostr-queries").get("review-feed:latest");
    expect(record.expires - now).toBe(NOSTR_FAILURE_TTL_MS);
    await expect(readNostrQuery(NOSTR_SCOPE.reviewFeed, "latest")).resolves.toMatchObject({
      failed: true,
    });

    // Ten minutes later it is a miss again, so a host whose network came back
    // is not left in the dark.
    vi.spyOn(Date, "now").mockReturnValue(now + NOSTR_FAILURE_TTL_MS + 1);
    await expect(readNostrQuery(NOSTR_SCOPE.reviewFeed, "latest")).resolves.toBeNull();
    vi.restoreAllMocks();
  });

  // A real answer must not be shadowed by an older failure: the reader gets
  // the reviews, not the memory of a network that was down.
  it("is replaced by a later real answer", async () => {
    const { writeNostrFailure, writeNostrQuery, readNostrQuery, NOSTR_SCOPE } =
      await freshCache();

    await writeNostrFailure(NOSTR_SCOPE.reviewFeed, "latest");
    await writeNostrQuery(NOSTR_SCOPE.reviewFeed, "latest", { candidates: [{ id: "a" }] }, ANSWERED);

    await expect(readNostrQuery(NOSTR_SCOPE.reviewFeed, "latest")).resolves.toEqual({
      candidates: [{ id: "a" }],
    });
  });
});

describe("when storage says no", () => {
  it("reads as a miss and writes as not landed, rather than throwing", async () => {
    vi.stubGlobal("indexedDB", undefined);
    const { writeNostrQuery, readNostrQuery } = await freshCache();

    expect(await writeNostrQuery("app-comments", "immich", { comments: [] }, ANSWERED)).toBe(false);
    await expect(readNostrQuery("app-comments", "immich")).resolves.toBeNull();
  });
});

describe("a record written by another build", () => {
  it("is ignored rather than misread", async () => {
    fake = fakeIndexedDB({ stores: new Map([["nostr-queries", new Map([
      ["app-comments:immich", { v: 0, data: { comments: [{ content: "stale" }] }, expires: Date.now() + 60_000 }],
    ])]]) });
    vi.stubGlobal("indexedDB", fake.api);

    const { readNostrQuery } = await freshCache();
    await expect(readNostrQuery("app-comments", "immich")).resolves.toBeNull();
  });
});