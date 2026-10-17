// NIP-01 relay transport: the WebSocket fan-out every Nostr read in Yantr
// makes, and the broadcast every Nostr write makes.
//
// It lives here, in the browser, rather than behind the API: reading a comment
// thread, resolving an author's name or a developer profile are all third-party
// queries with no server-side state, and a proxy for them would be an open
// relay for anyone who asked. The backend keeps only the relay *set* (see
// `composables/useNostrRelays.js`), which is configuration.
//
// utils/nostr.js stays pure — filters, tags, sanitising, bech32, signature
// verification — so none of the protocol rules live behind a socket.

// QUERY_TIMEOUT_MS bounds one relay exchange: dial, REQ, EVENTs and EOSE. A
// relay that accepts the socket and then says nothing is the failure this
// exists for, and it matches the bound the backend used for its own relay
// queries.
export const QUERY_TIMEOUT_MS = 8000;

// queryOneRelay runs a single REQ/EVENT/EOSE exchange against one relay and
// resolves with whatever events of the expected kind arrived. It never
// rejects: an unreachable relay means fewer results, not a failed page.
//
// `reached` distinguishes a relay that answered (even with nothing) from one
// that never came up, which is the difference between "no such comment" and
// "we could not ask".
function queryOneRelay(relay, filter, subId, kind, timeoutMs) {
  return new Promise((resolve) => {
    const found = [];
    let settled = false;
    let reached = false;

    const done = () => {
      if (!settled) {
        settled = true;
        try {
          ws.close();
        } catch {
          // Already gone; the events collected so far still count.
        }
        resolve({ events: found, reached });
      }
    };

    let ws;
    try {
      ws = new WebSocket(relay);
    } catch {
      resolve({ events: found, reached });
      return;
    }

    const timer = setTimeout(done, timeoutMs);

    ws.onopen = () => {
      reached = true;
      try {
        ws.send(JSON.stringify(["REQ", subId, filter]));
      } catch {
        clearTimeout(timer);
        done();
      }
    };

    ws.onmessage = (ev) => {
      let msg;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (!Array.isArray(msg) || msg.length === 0) return;

      const [verb, ...rest] = msg;
      if (verb === "EVENT" && rest[0] === subId && rest[1]?.kind === kind) {
        found.push(rest[1]);
      } else if (verb === "EOSE" && rest[0] === subId) {
        // Unsubscribe before leaving, so a relay that keeps subscriptions
        // alive does not hold resources for a socket we drop anyway.
        try {
          ws.send(JSON.stringify(["CLOSE", subId]));
        } catch {
          // Closing anyway; the events already delivered still count.
        }
        clearTimeout(timer);
        done();
      }
      // NOTICE / CLOSED / AUTH frames carry nothing displayable and need no
      // handling.
    };

    ws.onerror = () => {
      clearTimeout(timer);
      done();
    };
    ws.onclose = () => {
      clearTimeout(timer);
      done();
    };
  });
}

// queryRelays asks every relay in parallel and returns the merged events.
//
// Parallel rather than sequential because a miss is the common case for any one
// relay, and a sequence would make the wait grow with the size of the set.
// Merged rather than first-answer-wins because the sets differ: a thread may
// be spread across relays, and coverage is per-relay — one of them holding
// nothing is the normal case, not a failure.
export async function queryRelays(relays, filter, kind, options = {}) {
  const { timeoutMs = QUERY_TIMEOUT_MS, subIdPrefix = "yantr" } = options;
  const subId = `${subIdPrefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

  const perRelay = await Promise.all(
    (Array.isArray(relays) ? relays : []).map((relay) =>
      queryOneRelay(relay, filter, subId, kind, timeoutMs)
    )
  );

  const seen = new Set();
  const events = [];
  for (const { events: fromRelay } of perRelay) {
    for (const event of fromRelay || []) {
      const id = String(event?.id ?? "");
      if (!id || seen.has(id)) continue;
      seen.add(id);
      events.push(event);
    }
  }

  return {
    events,
    reached: perRelay.filter((r) => r.reached).length,
    total: perRelay.length,
  };
}

// publishToRelay broadcasts one signed event and resolves true on a positive
// NIP-20 OK. Auth-gated relays answer with AUTH or a false OK; both are a
// "no" from this relay rather than a failure of the publish.
export function publishToRelay(relay, signedEvent, options = {}) {
  const { timeoutMs = QUERY_TIMEOUT_MS } = options;
  return new Promise((resolve) => {
    let settled = false;
    const done = (ok) => {
      if (!settled) {
        settled = true;
        try {
          ws.close();
        } catch {
          // The verdict is already recorded; a close failure changes nothing.
        }
        resolve(ok);
      }
    };

    let ws;
    try {
      ws = new WebSocket(relay);
    } catch {
      resolve(false);
      return;
    }

    const timer = setTimeout(() => done(false), timeoutMs);

    ws.onopen = () => {
      try {
        ws.send(JSON.stringify(["EVENT", signedEvent]));
      } catch {
        clearTimeout(timer);
        done(false);
      }
    };
    ws.onmessage = (ev) => {
      let msg;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (Array.isArray(msg) && msg[0] === "OK" && msg[1] === signedEvent.id) {
        clearTimeout(timer);
        done(msg[2] === true);
      }
    };
    ws.onerror = () => {
      clearTimeout(timer);
      done(false);
    };
    ws.onclose = () => {
      clearTimeout(timer);
      done(false);
    };
  });
}