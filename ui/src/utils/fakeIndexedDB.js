// The fake IndexedDB kv.js is written against, extracted so the cache built on
// top of it is held to the same standards. Not a test file: no `it` lives here.
export function fakeIndexedDB(options = {}) {
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