import { afterEach, describe, expect, it, vi } from "vitest";
import { publishToRelay, queryRelays } from "./relay";

// A relay that answers however the test tells it to. Everything about the
// fan-out under test — what counts as reached, what merges, when a silent relay
// gives up — is decided by this fake rather than by a network, which is the
// whole reason the transport is its own module.
class FakeSocket {
  constructor(url) {
    this.url = url;
    this.sent = [];
    FakeSocket.instances.push(this);
    queueMicrotask(() => {
      if (this.behaviour === "refuse") {
        this.onerror?.();
        return;
      }
      // Every non-refusing socket opens. "silent" is the one that opens and
      // then never answers, which is what the timeout exists for.
      this.onopen?.();
    });
  }

  send(raw) {
    this.sent.push(JSON.parse(raw));
    const msg = JSON.parse(raw);
    if (msg[0] !== "REQ" || this.behaviour === "silent") return;
    const [, subId] = msg;
    queueMicrotask(() => {
      for (const event of this.events || []) {
        this.onmessage?.({ data: JSON.stringify(["EVENT", subId, event]) });
      }
      if (this.behaviour !== "no_eose") {
        this.onmessage?.({ data: JSON.stringify(["EOSE", subId]) });
      }
    });
  }

  close() {
    this.closed = true;
  }
}
FakeSocket.instances = [];

function installRelay(behaviours) {
  let i = 0;
  vi.stubGlobal(
    "WebSocket",
    class extends FakeSocket {
      constructor(url) {
        super(url);
        this.behaviour = behaviours[i++] || "answer";
      }
    }
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  FakeSocket.instances.length = 0;
});

const note = (id) => ({ id, pubkey: "aa".repeat(32), kind: 1, created_at: 1, tags: [], content: id });

describe("queryRelays", () => {
  it("merges what every relay answered and counts who reached", async () => {
    installRelay(["answer", "answer"]);
    // The fake reads events off the socket, so give each one something to hold.
    vi.stubGlobal(
      "WebSocket",
      class extends FakeSocket {
        constructor(url) {
          super(url);
          this.behaviour = "answer";
          this.events = [note(url.includes("one") ? "e1" : "e2")];
        }
      }
    );

    const { events, reached, total } = await queryRelays(
      ["wss://one", "wss://two"],
      { kinds: [1] },
      1
    );

    expect(total).toBe(2);
    expect(reached).toBe(2);
    expect(events.map((e) => e.id).sort()).toEqual(["e1", "e2"]);
  });

  it("keeps one copy of an event two relays both hold", async () => {
    vi.stubGlobal(
      "WebSocket",
      class extends FakeSocket {
        constructor() {
          super("wss://relay");
          this.behaviour = "answer";
          this.events = [note("shared")];
        }
      }
    );

    const { events } = await queryRelays(["wss://a", "wss://b"], { kinds: [1] }, 1);
    expect(events).toHaveLength(1);
  });

  it("treats a relay that refuses as not reached, without failing the read", async () => {
    installRelay(["refuse", "answer"]);

    const { events, reached, total } = await queryRelays(["wss://down", "wss://up"], {}, 1);

    expect(total).toBe(2);
    expect(reached).toBe(1);
    expect(events).toEqual([]);
  });

  it("only accepts events of the kind that was asked for", async () => {
    vi.stubGlobal(
      "WebSocket",
      class extends FakeSocket {
        constructor() {
          super("wss://relay");
          this.behaviour = "answer";
          this.events = [note("wanted"), { ...note("other"), kind: 0 }];
        }
      }
    );

    const { events } = await queryRelays(["wss://relay"], { kinds: [1] }, 1);
    expect(events.map((e) => e.id)).toEqual(["wanted"]);
  });

  it("gives up on a relay that opens and then says nothing", async () => {
    installRelay(["silent"]);

    const started = Date.now();
    const { reached } = await queryRelays(["wss://mute"], { kinds: [1] }, 1, {
      timeoutMs: 40,
    });

    expect(reached).toBe(1); // it did come up, it just never answered
    expect(Date.now() - started).toBeLessThan(2000);
    expect(FakeSocket.instances[0].closed).toBe(true);
  });

  it("closes the subscription before leaving, so a relay lets it go", async () => {
    vi.stubGlobal(
      "WebSocket",
      class extends FakeSocket {
        constructor() {
          super("wss://relay");
          this.behaviour = "answer";
        }
      }
    );

    await queryRelays(["wss://relay"], { kinds: [1] }, 1);
    expect(FakeSocket.instances[0].sent.map((m) => m[0])).toContain("CLOSE");
  });

  it("answers with an empty set when no relays are configured", async () => {
    const { events, reached, total } = await queryRelays([], {}, 1);
    expect({ events, reached, total }).toEqual({ events: [], reached: 0, total: 0 });
  });
});

describe("publishToRelay", () => {
  it("is true only on a positive NIP-20 OK", async () => {
    const signed = { id: "ab".repeat(32) };
    vi.stubGlobal(
      "WebSocket",
      class extends FakeSocket {
        constructor() {
          super("wss://relay");
          this.behaviour = "answer";
          queueMicrotask(() => this.onopen?.());
          const self = this;
          queueMicrotask(() =>
            self.onmessage?.({
              data: JSON.stringify(["OK", signed.id, true, ""]),
            })
          );
        }
      }
    );

    expect(await publishToRelay("wss://relay", signed)).toBe(true);
  });

  it("is false when the relay refuses the event", async () => {
    const signed = { id: "cd".repeat(32) };
    vi.stubGlobal(
      "WebSocket",
      class extends FakeSocket {
        constructor() {
          super("wss://relay");
          this.behaviour = "answer";
          queueMicrotask(() => this.onopen?.());
          const self = this;
          queueMicrotask(() =>
            self.onmessage?.({
              data: JSON.stringify(["OK", signed.id, false, "blocked: not authenticated"]),
            })
          );
        }
      }
    );

    expect(await publishToRelay("wss://relay", signed)).toBe(false);
  });

  it("is false, not a rejection, when the relay cannot be reached", async () => {
    installRelay(["refuse"]);
    expect(await publishToRelay("wss://down", { id: "ef".repeat(32) })).toBe(false);
  });
});