// A tiny IndexedDB key-value store, for the small amount of data worth keeping
// across visits: reference data a reader would otherwise re-fetch from a third
// party on every page load. A developer's kind-0 is the first user.
//
// Nothing that matters for correctness is stored here. Every read is treated as
// a miss when the record is missing, expired, from an older shape or unreadable,
// so the worst case of a broken store is an extra relay query — never a wrong
// page.
//
// Every operation is guarded, because storage is not reliably available:
//
//   - private windows and "block all cookies" can refuse to open a database
//     outright, so `indexedDB` may be missing or may throw on open;
//   - an `open()` that never fires `success` is a real failure mode, so every
//     request carries a deadline rather than waiting forever;
//   - any of it failing resolves as a miss, and clears the memo so a later call
//     can try again rather than inheriting one failed open forever.

// DB_NAME is one database and DB_VERSION one schema. The version is not fixed:
// a store is created by bumping it, which is the only way to create one.
const DB_NAME = "yantr";
const DB_VERSION = 1;

// OPEN_TIMEOUT_MS bounds opening the database, OP_TIMEOUT_MS bounds one read or
// write, CREATE_ATTEMPTS bounds the store-creation loop. All are short on
// purpose: the caller is a page that would rather show a profile half a second
// late than show nothing at all.
const OPEN_TIMEOUT_MS = 1500;
const OP_TIMEOUT_MS = 1500;
const CREATE_ATTEMPTS = 3;

let dbPromise = null;

// knownVersion is read back from every handle that opens, never assumed. An
// upgrade only runs when the requested version is *higher* than the database's,
// so a page that has never seen the database before — or one reloaded after
// another page already created a store — must ask for the version that is
// actually there, plus one, or the upgrade silently never fires.
let knownVersion = DB_VERSION;

// openDatabase opens (or reuses) the one handle, resolving null when the
// database cannot be opened at all.
//
// `createStore` runs inside the upgrade transaction, which is the only place a
// store can be created. It is passed in rather than hardcoded so the set of
// stores is not this module's business: a caller asks for a store by name, and
// the version is bumped until that store exists.
function openDatabase(createStore, version = knownVersion) {
  if (dbPromise) return dbPromise;

  dbPromise = new Promise((resolve) => {
    let settled = false;
    const done = (value) => {
      if (settled) return;
      settled = true;
      // A failed open is not memoised: the next caller gets a fresh attempt,
      // since storage can be revoked and restored while a page is open.
      if (!value) dbPromise = null;
      resolve(value);
    };

    try {
      const idb = globalThis.indexedDB;
      if (!idb) return done(null);

      const request = idb.open(DB_NAME, version);
      const timer = setTimeout(() => done(null), OPEN_TIMEOUT_MS);

      request.onupgradeneeded = () => {
        if (!createStore) return;
        try {
          createStore(request.result);
        } catch {
          // A store that cannot be created is reported by the operations that
          // need it, not here.
        }
      };
      request.onsuccess = () => {
        clearTimeout(timer);
        const db = request.result;
        knownVersion = db.version;
        // Another tab upgrading the schema must not leave this one holding a
        // connection it can no longer use.
        db.onversionchange = () => {
          db.close();
          done(null);
        };
        done(db);
      };
      request.onerror = () => {
        clearTimeout(timer);
        done(null);
      };
      request.onblocked = () => {
        clearTimeout(timer);
        done(null);
      };
    } catch {
      done(null);
    }
  });

  return dbPromise;
}

// storeDatabase returns an open handle that is known to contain storeName,
// creating the store if this is the first anything has asked for it.
//
// The loop is bounded and re-checks each time, so two tabs racing to add two
// different stores both settle: the loser's upgrade is blocked, degrades to a
// miss, and its next attempt finds the store the winner created.
async function storeDatabase(storeName) {
  for (let attempt = 0; attempt < CREATE_ATTEMPTS; attempt++) {
    const db = await openDatabase();
    if (!db) return null;
    if (db.objectStoreNames.contains(storeName)) return db;

    // Close this handle before reopening at a higher version: an open connection
    // blocks the very upgrade that would create the store. The new version is
    // derived from the handle just closed, not from the constant at the top —
    // asking for the same version again would open successfully and create
    // nothing, which is exactly the bug that made this store silently missing.
    db.close();
    const nextVersion = db.version + 1;
    dbPromise = null;

    const upgraded = await openDatabase((handle) => {
      if (!handle.objectStoreNames.contains(storeName)) {
        handle.createObjectStore(storeName);
      }
    }, nextVersion);
    if (!upgraded) return null;
  }
  return null;
}

// withStore runs one operation against one object store, in one transaction, and
// resolves to null if any part of that is unavailable.
async function withStore(storeName, mode, run) {
  const db = await storeDatabase(storeName);
  if (!db) return null;

  try {
    const tx = db.transaction(storeName, mode);
    const store = tx.objectStore(storeName);
    const request = run(store);
    if (!request) return null;

    let settled = false;
    return await new Promise((resolve) => {
      const finish = (value) => {
        if (settled) return;
        settled = true;
        resolve(value);
      };
      const timer = setTimeout(() => finish(null), OP_TIMEOUT_MS);
      request.onsuccess = () => {
        clearTimeout(timer);
        finish(request.result ?? null);
      };
      request.onerror = () => {
        clearTimeout(timer);
        finish(null);
      };
    });
  } catch {
    return null;
  }
}

// kvGet reads one record, or null when there is nothing to read.
export function kvGet(storeName, key) {
  return withStore(storeName, "readonly", (store) => store.get(key));
}

// kvSet writes one record. Returns whether the write was accepted, so a caller
// can tell "cached" from "not cached" without guessing.
export async function kvSet(storeName, key, value) {
  const written = await withStore(storeName, "readwrite", (store) => store.put(value, key));
  return written !== null;
}

// kvRemove deletes one record. A record that was not there is not an error.
export function kvRemove(storeName, key) {
  return withStore(storeName, "readwrite", (store) => store.delete(key));
}

// kvIsFresh is the cache's own rule, kept next to the store so a caller cannot
// invent a second one: a record counts only while it carries an expiry that has
// not passed. A record with no readable expiry is not fresh — that is how a
// cache bug turns into a page showing a name from last year with no sign of it.
export function kvIsFresh(entry, now = Date.now()) {
  if (!entry || typeof entry !== "object") return false;
  return typeof entry.expires === "number" && entry.expires > now;
}