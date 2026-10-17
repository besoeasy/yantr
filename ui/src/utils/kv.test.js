import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The store's whole job is to be invisible when storage does not work: these
// tests pin the degradation path, which is the one that runs in private windows
// and in any browser where the database refuses to open.
//
// The other half is here too, against a fake that holds kv.js to the two rules
// of the specification it got wrong the first time:
//
//   - a store can only be created inside an upgrade transaction, so
//     createObjectStore throws outside one;
//   - an upgrade only runs when the requested version is higher than the stored
//     one, so reopening at the same version fires success and creates nothing.
//
// Both bugs were silent: a write that reported it had landed while storing
// nothing, and a schema bump that never happened. Nothing else in the project
// could have caught them.

// fakeIndexedDB implements the parts of the API kv.js uses.
function fakeIndexedDB(options = {}) {
  const state = { version: options.version ?? 0, stores: new Map(options.stores ?? []) };

  const nameList = () => ({ contains: (name) => state.stores.has(name) });

  function makeRequest(run) {
    const request = {};
    queueMicrotask(() => {
      request.result = run();
      request.onsuccess?.();
    });
    return request;
  }

  function makeDb(handleVersion, upgrading) {
    return {
      version: handleVersion,
      objectStoreNames: nameList(),
      close() {},
      createObjectStore(name) {
        if (!upgrading) throw new Error("InvalidStateError: outside a versionchange transaction");
        state.stores.set(name, new Map());
        return { name };
      },
      transaction(storeName) {
        const data = state.stores.get(storeName);
        if (!data) throw new Error(`NotFoundError: ${storeName} is not an object store`);
        return {
          objectStore: () => ({
            get: (key) => makeRequest(() => data.get(key)),
            put: (value, key) =>
              makeRequest(() => {
                data.set(key, value);
                return key;
              }),
            delete: (key) =>
              makeRequest(() => {
                data.delete(key);
                return undefined;
              }),
          }),
        };
      },
    };
  }

  const api = {
    open(name, requested) {
      const request = {};
      if (options.blocked) {
        queueMicrotask(() => request.onblocked?.());
        return request;
      }

      const upgrading = requested > state.version;
      queueMicrotask(() => {
        if (upgrading) {
          state.version = requested;
          request.result = makeDb(requested, true);
          request.onupgradeneeded?.();
        } else {
          request.result = makeDb(state.version, false);
        }
        request.onsuccess?.();
      });
      return request;
    },
  };

  return { state, api };
}

// kv.js memoises its open handle and its idea of the schema version at module
// scope, so every test needs its own copy of the module — and must reach it only
// through here. A top-level static import would load kv.js before any test stubs
// storage, and the memo it started with would survive the reset, pointing at
// whichever fake the previous test installed.
async function freshKv() {
  vi.resetModules();
  return import("./kv");
}

let fake;

beforeEach(() => {
  fake = fakeIndexedDB();
  vi.stubGlobal("indexedDB", fake.api);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.resetModules();
});

describe("a record survives being written and read back", () => {
  it("round-trips a value", async () => {
    const { kvSet, kvGet } = await freshKv();
    const record = { v: 1, profile: { name: "jack" }, expires: Date.now() + 60_000 };

    expect(await kvSet("nostr-profiles", "abc", record)).toBe(true);
    await expect(kvGet("nostr-profiles", "abc")).resolves.toEqual(record);
  });

  it("reads a key that was never written as a miss, not an error", async () => {
    const { kvGet } = await freshKv();
    await expect(kvGet("nostr-profiles", "never-written")).resolves.toBe(null);
  });

  it("removes a record", async () => {
    const { kvSet, kvGet, kvRemove } = await freshKv();
    await kvSet("s", "k", { expires: Date.now() + 1000 });
    await kvRemove("s", "k");
    await expect(kvGet("s", "k")).resolves.toBe(null);
  });
});

describe("stores are created on demand", () => {
  it("creates the store the caller asked for, then writes into it", async () => {
    const { kvSet, kvGet } = await freshKv();

    expect(await kvSet("later-store", "k", { expires: Date.now() + 1000 })).toBe(true);
    expect(fake.state.stores.has("later-store")).toBe(true);
    await expect(kvGet("later-store", "k")).resolves.toMatchObject({ expires: expect.any(Number) });
  });

  it("bumps the schema version rather than reopening at the one it has", async () => {
    const { kvSet } = await freshKv();
    await kvSet("first", "k", { expires: 1 });
    const afterFirst = fake.state.version;

    await kvSet("second", "k", { expires: 1 });

    expect(fake.state.version).toBeGreaterThan(afterFirst);
    expect(fake.state.stores.has("first")).toBe(true);
    expect(fake.state.stores.has("second")).toBe(true);
  });

  // A page reloaded after another page created the store starts at the constant
  // version again. Used as-is, its upgrade would never fire.
  it("opens a store that already exists without trying to create it again", async () => {
    fake = fakeIndexedDB({ version: 4, stores: new Map([["existing", new Map([["k", { expires: 7 }]])]]) });
    vi.stubGlobal("indexedDB", fake.api);

    const { kvGet } = await freshKv();
    await expect(kvGet("existing", "k")).resolves.toEqual({ expires: 7 });
    expect(fake.state.version).toBe(4);
  });
});

describe("without IndexedDB", () => {
  it("reads as a miss rather than throwing", async () => {
    vi.stubGlobal("indexedDB", undefined);
    const { kvGet } = await freshKv();
    await expect(kvGet("store", "key")).resolves.toBe(null);
  });

  it("reports a write as not landing", async () => {
    vi.stubGlobal("indexedDB", undefined);
    const { kvSet } = await freshKv();
    await expect(kvSet("store", "key", { v: 1 })).resolves.toBe(false);
  });

  it("removes as a miss rather than throwing", async () => {
    vi.stubGlobal("indexedDB", undefined);
    const { kvRemove } = await freshKv();
    await expect(kvRemove("store", "key")).resolves.toBe(null);
  });

  it("survives a host whose indexedDB throws on use", async () => {
    vi.stubGlobal("indexedDB", {
      open() {
        throw new Error("storage is disabled");
      },
    });
    const { kvGet, kvSet } = await freshKv();
    await expect(kvGet("store", "key")).resolves.toBe(null);
    await expect(kvSet("store", "key", {})).resolves.toBe(false);
  });

  // Another tab holding an older version blocks the upgrade. The page must still
  // render; it just has no cache this time.
  it("degrades to a miss when an upgrade is blocked", async () => {
    vi.stubGlobal("indexedDB", fakeIndexedDB({ blocked: true }).api);
    const { kvSet, kvGet } = await freshKv();
    await expect(kvSet("s", "k", { expires: 1 })).resolves.toBe(false);
    await expect(kvGet("s", "k")).resolves.toBe(null);
  });
});

describe("kvIsFresh", () => {
  const NOW = 1_800_000_000_000;

  it("accepts a record whose expiry has not passed", async () => {
    const { kvIsFresh } = await freshKv();
    expect(kvIsFresh({ expires: NOW + 1 }, NOW)).toBe(true);
  });

  it("rejects a record at or past its expiry", async () => {
    const { kvIsFresh } = await freshKv();
    expect(kvIsFresh({ expires: NOW }, NOW)).toBe(false);
    expect(kvIsFresh({ expires: NOW - 1 }, NOW)).toBe(false);
  });

  // The failure that matters: a record with no usable expiry must read as a
  // miss. Treating it as fresh is how a cache bug becomes a page showing a name
  // from a year ago with no sign of it.
  it("rejects anything without an expiry it can read", async () => {
    const { kvIsFresh } = await freshKv();
    for (const entry of [null, undefined, {}, 42, "record", { expires: "later" }, []]) {
      expect(kvIsFresh(entry, NOW)).toBe(false);
    }
  });
});