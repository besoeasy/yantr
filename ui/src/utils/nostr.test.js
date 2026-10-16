import { describe, expect, it } from "vitest";
import {
  COMMENT_EXPIRY_SECONDS,
  appCommentTag,
  buildCommentTags,
  commentExpiryTimestamp,
  commentFilter,
  computeEventId,
  expirationOf,
  hasValidId,
  hexToNpub,
  isAppComment,
} from "./nostr";

// Externally-verified vector the backend tests use too.
const JACK_HEX = "82341f882b6eabcd2ba7f1ef90aad961cf074af15b9ef44a09f9d2a8fbfbe6a2";
const JACK_NPUB = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m";

const NOW = 1_800_000_000;

function commentEvent(overrides = {}) {
  return {
    id: "00".repeat(32),
    pubkey: JACK_HEX,
    created_at: NOW - 60,
    kind: 1,
    tags: [
      ["t", "yantr"],
      ["t", "yantr-ollama"],
      ["expiration", String(NOW + COMMENT_EXPIRY_SECONDS)],
    ],
    content: "Runs great on my machine.",
    ...overrides,
  };
}

describe("appCommentTag", () => {
  it("scopes one catalog id under the yantr namespace", () => {
    expect(appCommentTag("ollama")).toBe("yantr-ollama");
    expect(appCommentTag("Ollama")).toBe("yantr-ollama");
    expect(appCommentTag("  open-webui  ")).toBe("yantr-open-webui");
  });
});

describe("commentExpiryTimestamp", () => {
  it("is five years after creation", () => {
    expect(commentExpiryTimestamp(1_000_000_000)).toBe(1_000_000_000 + COMMENT_EXPIRY_SECONDS);
    expect(COMMENT_EXPIRY_SECONDS).toBe(5 * 365 * 24 * 60 * 60);
  });
});

describe("buildCommentTags", () => {
  it("carries both scopes and a five-year NIP-40 expiry", () => {
    const tags = buildCommentTags("ollama", 1_000_000_000);
    expect(tags).toContainEqual(["t", "yantr"]);
    expect(tags).toContainEqual(["t", "yantr-ollama"]);
    expect(tags).toContainEqual(["expiration", String(1_000_000_000 + COMMENT_EXPIRY_SECONDS)]);
  });
});

describe("commentFilter", () => {
  it("queries kind-1 notes on the selective per-app tag", () => {
    const filter = commentFilter("ollama");
    expect(filter.kinds).toEqual([1]);
    expect(filter["#t"]).toEqual(["yantr-ollama"]);
    expect(filter.limit).toBeGreaterThan(0);
  });
});

describe("isAppComment", () => {
  it("accepts a well-formed comment for the app", () => {
    expect(isAppComment(commentEvent(), "ollama", NOW)).toBe(true);
  });

  it("rejects the neighbouring app's comments", () => {
    expect(isAppComment(commentEvent(), "open-webui", NOW)).toBe(false);
  });

  it("requires the yantr scope tag, since the filter alone cannot express AND", () => {
    const event = commentEvent({
      tags: [
        ["t", "yantr-ollama"],
        ["expiration", String(NOW + 1000)],
      ],
    });
    expect(isAppComment(event, "ollama", NOW)).toBe(false);
  });

  it("hides comments past their NIP-40 expiry", () => {
    const event = commentEvent({ tags: [["t", "yantr"], ["t", "yantr-ollama"], ["expiration", String(NOW - 1)]] });
    expect(isAppComment(event, "ollama", NOW)).toBe(false);
  });

  it("keeps comments with no expiry tag", () => {
    const event = commentEvent({ tags: [["t", "yantr"], ["t", "yantr-ollama"]] });
    expect(isAppComment(event, "ollama", NOW)).toBe(true);
  });

  it("rejects the wrong kind, an empty note and a bad pubkey", () => {
    expect(isAppComment(commentEvent({ kind: 7 }), "ollama", NOW)).toBe(false);
    expect(isAppComment(commentEvent({ content: "   " }), "ollama", NOW)).toBe(false);
    expect(isAppComment(commentEvent({ pubkey: "not-a-key" }), "ollama", NOW)).toBe(false);
  });
});

describe("expirationOf", () => {
  it("reads the first well-formed expiration tag", () => {
    expect(expirationOf(commentEvent())).toBe(NOW + COMMENT_EXPIRY_SECONDS);
    expect(expirationOf(commentEvent({ tags: [["t", "yantr"]] }))).toBe(null);
  });
});

describe("hexToNpub", () => {
  it("encodes a pubkey to the identifier readers recognise", () => {
    expect(hexToNpub(JACK_HEX)).toBe(JACK_NPUB);
  });

  it("returns nothing for a value that is not a pubkey", () => {
    expect(hexToNpub("xyz")).toBe("");
    expect(hexToNpub("")).toBe("");
    expect(hexToNpub(null)).toBe("");
  });
});

describe("event id", () => {
  it("is deterministic and 32 bytes of hex", async () => {
    const fields = {
      pubkey: JACK_HEX,
      created_at: 1_700_000_000,
      kind: 1,
      tags: [["t", "yantr"]],
      content: "hello",
    };
    const a = await computeEventId(fields);
    const b = await computeEventId(fields);
    expect(a).toBe(b);
    expect(a).toMatch(/^[0-9a-f]{64}$/);
    expect(await computeEventId({ ...fields, content: "different" })).not.toBe(a);
  });

  it("validates an event against its own fields", async () => {
    const fields = {
      pubkey: JACK_HEX,
      created_at: 1_700_000_000,
      kind: 1,
      tags: [["t", "yantr"]],
      content: "hello",
    };
    const id = await computeEventId(fields);
    expect(await hasValidId({ ...fields, id })).toBe(true);
    expect(await hasValidId({ ...fields, id, content: "tampered" })).toBe(false);
  });
});
