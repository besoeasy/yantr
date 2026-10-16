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
  npubToHex,
  parseKind0Profile,
  profileFilter,
  sanitizeCommentContent,
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

describe("sanitizeCommentContent", () => {
  it("keeps ordinary text, emoji and single newlines", () => {
    expect(sanitizeCommentContent("Works great \u{1F44D}\nTwo lines.")).toBe(
      "Works great \u{1F44D}\nTwo lines."
    );
  });

  // Markup stays inert one layer up: interpolation escapes it, so the
  // sanitiser passes it through byte-identical rather than second-guessing
  // the renderer.
  it("leaves markup characters alone for the renderer to escape", () => {
    expect(sanitizeCommentContent('<script>alert("x")</script>')).toBe(
      '<script>alert("x")</script>'
    );
  });

  it("strips control characters but keeps newline and tab", () => {
    expect(sanitizeCommentContent("a\u0001b\u0002c\td\ne")).toBe("abc\td\ne");
  });

  it("strips bidi overrides and zero-width spoof characters", () => {
    expect(sanitizeCommentContent("admin\u202E Steiner")).toBe("admin Steiner");
    expect(sanitizeCommentContent("a\u200Bb\u00ADc\u200Fd")).toBe("abcd");
    expect(sanitizeCommentContent("\uFEFFhello")).toBe("hello");
  });

  it("collapses runs of blank lines", () => {
    expect(sanitizeCommentContent("a\n\n\n\nb")).toBe("a\n\nb");
  });

  it("truncates overlong relay payloads without splitting surrogate pairs", () => {
    const long = "a".repeat(1998) + "\u{1F600}\u{1F600}\u{1F600}";
    const clean = sanitizeCommentContent(long);
    expect([...clean].length).toBe(2000);
    expect(clean.endsWith("\u2026")).toBe(true);
    // No lone surrogates: every code unit belongs to a complete pair.
    expect(/[\uD800-\uDBFF](?![\uDC00-\uDFFF])/.test(clean)).toBe(false);
    expect(/(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/.test(clean)).toBe(false);
  });

  it("returns empty when nothing displayable remains", () => {
    expect(sanitizeCommentContent("   ")).toBe("");
    expect(sanitizeCommentContent("\u200B\u00AD\u200F")).toBe("");
    expect(sanitizeCommentContent(null)).toBe("");
  });
});

describe("npubToHex", () => {
  // Externally-verified vectors shared with core/nip19.
  const FIATJAF_NPUB =
    "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6";
  const FIATJAF_HEX =
    "3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d";
  const JACK_NPROFILE =
    "nprofile1qqsgydql3q4ka27d9wnlrmus4tvkrnc8ftc4h8h5fgyln54gl0a7dgspzemhxue69uhhyetvv9ujuurjd9kkzmpwdejhgw2eqf8";
  const FIATJAF_NPROFILE =
    "nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpz3mhxue69uhhyetvv9ujuerpd46hxtnfduqs6amnwvaz7tmwdaejumr0dsfg70ls";

  it("decodes an npub to the hex relays filter on", () => {
    expect(npubToHex(JACK_NPUB)).toBe(JACK_HEX);
    expect(npubToHex(FIATJAF_NPUB)).toBe(FIATJAF_HEX);
  });

  it("unwraps an nprofile past its relay hints", () => {
    expect(npubToHex(JACK_NPROFILE)).toBe(JACK_HEX);
    expect(npubToHex(FIATJAF_NPROFILE)).toBe(FIATJAF_HEX);
  });

  it("round-trips through the encoder", () => {
    expect(hexToNpub(npubToHex(JACK_NPUB))).toBe(JACK_NPUB);
  });

  it("tolerates case and surrounding whitespace, like the catalog allows", () => {
    expect(npubToHex("  " + JACK_NPUB.toUpperCase() + "  ")).toBe(JACK_HEX);
  });

  it("returns nothing for anything that is not a profile identifier", () => {
    expect(npubToHex("")).toBe("");
    expect(npubToHex("besoeasy")).toBe("");
    // note1 is an event id, nsec1 a private key: neither names an author.
    expect(
      npubToHex(
        "note1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q7k28gn"
      )
    ).toBe("");
    expect(
      npubToHex(
        "nsec1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qr2zmhw"
      )
    ).toBe("");
    // Broken checksum, wrong payload sizes, malformed TLV.
    expect(
      npubToHex(
        "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63q"
      )
    ).toBe("");
    expect(
      npubToHex(
        "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmucs89urk"
      )
    ).toBe("");
    expect(npubToHex("nprofile1qypkzcnre6e3r9")).toBe("");
  });
});

describe("profileFilter", () => {
  it("batches authors into one kind-0 filter", () => {
    const filter = profileFilter([JACK_HEX, JACK_HEX.toUpperCase(), "nope"]);
    expect(filter.kinds).toEqual([0]);
    expect(filter["#t"]).toBeUndefined();
    expect(filter.authors).toEqual([JACK_HEX]);
  });
});

describe("parseKind0Profile", () => {
  const JACK_PROFILE = {
    kind: 0,
    pubkey: JACK_HEX,
    created_at: 1700000000,
    content: JSON.stringify({
      name: "jack",
      display_name: "Jack",
      about: "no state is the best state",
      picture: "https://example.test/j.jpg",
      nip05: "jack@primal.net",
    }),
  };

  it("maps a kind-0 event onto display fields", () => {
    const p = parseKind0Profile(JACK_PROFILE);
    expect(p.pubkey).toBe(JACK_HEX);
    expect(p.name).toBe("jack");
    expect(p.picture).toBe("https://example.test/j.jpg");
    expect(p.nip05).toBe("jack@primal.net");
    expect(p.created_at).toBe(1700000000);
  });

  it("falls back to display_name and sanitises hostile text", () => {
    const p = parseKind0Profile({
      ...JACK_PROFILE,
      content: JSON.stringify({ display_name: "admin\u202e x" }),
    });
    expect(p.name).toBe("admin x");
  });

  it("refuses non-http avatar schemes and oversized URLs", () => {
    for (const picture of [
      "javascript:alert(1)",
      "data:image/png;base64,aaaa",
      "ftp://example.test/j.jpg",
      "https://" + "a".repeat(600),
    ]) {
      const p = parseKind0Profile({
        ...JACK_PROFILE,
        content: JSON.stringify({ name: "j", picture }),
      });
      expect(p.picture).toBe("");
    }
  });

  it("returns null for the wrong kind, a bad pubkey or non-JSON content", () => {
    expect(parseKind0Profile({ ...JACK_PROFILE, kind: 1 })).toBe(null);
    expect(parseKind0Profile({ ...JACK_PROFILE, pubkey: "xyz" })).toBe(null);
    const p = parseKind0Profile({ ...JACK_PROFILE, content: "not json" });
    expect(p.name).toBe("");
    expect(p.picture).toBe("");
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
