import { describe, expect, it } from "vitest";
import { finalizeEvent, generateSecretKey, getPublicKey } from "nostr-tools";
import {
  COMMENT_EXPIRY_SECONDS,
  appCommentTag,
  authorCommentFilter,
  bech32Encode,
  buildCommentTags,
  commentAppIds,
  commentExpiryTimestamp,
  commentFilter,
  expirationOf,
  hexToNpub,
  isAppComment,
  isAuthorYantrComment,
  isAuthentic,
  isLightningAddress,
  isYantrComment,
  lnurlEncode,
  newestKind0Profiles,
  npubToHex,
  parseKind0Profile,
  profileFilter,
  sanitizeCommentContent,
  yantrFeedFilter,
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

// ─── bech32 / LNURL ────────────────────────────────────────────────────────

// Fixtures computed with an independent BIP-173 implementation; the same
// reference that verifies JACK_NPUB below (its npub round-trip is asserted
// first, so the LNURL vectors are only as good as a checked encoder).
const LNURL_FIXTURES = [
  ["user@example.com", "lnurl1dp68gurn8ghj7etcv9khqmr99e3k7mf09emk2mrv944kummhdchkcmn4wfk8qtm4wdjhydu9pn6"],
  ["jack@cash.app", "lnurl1dp68gurn8ghj7cmpwd5zuctswqhjuam9d3kz66mwdamkutmvde6hymrs9a4xzcmtv8vsm3"],
  ["portuguese@example.com", "lnurl1dp68gurn8ghj7etcv9khqmr99e3k7mf09emk2mrv944kummhdchkcmn4wfk8qtmsdae8gat8w4jhxegamp8hq"],
];

describe("bech32Encode", () => {
  it("agrees with hexToNpub on the externally-verified npub vector", () => {
    const bytes = new Uint8Array(32);
    for (let i = 0; i < 32; i++) bytes[i] = parseInt(JACK_HEX.slice(i * 2, i * 2 + 2), 16);
    expect(bech32Encode("npub", bytes)).toBe(JACK_NPUB);
  });
});

describe("lnurlEncode", () => {
  it("matches fixtures computed independently of this code", () => {
    for (const [lud, expected] of LNURL_FIXTURES) {
      expect(lnurlEncode(lud)).toBe(expected);
      expect(lnurlEncode(lud.toUpperCase())).toBe(expected);
    }
  });

  it("rejects anything a wallet lookup could be abused with, returning no QR payload", () => {
    expect(lnurlEncode("")).toBe("");
    expect(lnurlEncode("not-an-address")).toBe("");
    expect(lnurlEncode("user@")).toBe("");
    // No query syntax, no slashes, no whitespace smuggling into the URL.
    expect(lnurlEncode("a@b.com/?x")).toBe("");
    expect(lnurlEncode("a@b.co m")).toBe("");
  });
});

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

describe("authorCommentFilter", () => {
  it("pins authorship and queries the bare yantr scope", () => {
    const filter = authorCommentFilter(JACK_NPUB);
    expect(filter).toBeNull(); // bech32, not hex — relays speak hex
    const hex = npubToHex(JACK_NPUB);
    const filterByHex = authorCommentFilter(hex);
    expect(filterByHex.kinds).toEqual([1]);
    expect(filterByHex.authors).toEqual([JACK_HEX]);
    expect(filterByHex["#t"]).toEqual(["yantr"]);
    expect(filterByHex.limit).toBeGreaterThan(0);
  });

  it("is null for a value that is not a pubkey, never an author-less filter", () => {
    expect(authorCommentFilter("")).toBeNull();
    expect(authorCommentFilter("not-a-key")).toBeNull();
    expect(authorCommentFilter("z".repeat(64))).toBeNull();
  });
});

describe("commentAppIds", () => {
  it("extracts per-app tags, skipping the bare scope", () => {
    expect(commentAppIds(commentEvent())).toEqual(["ollama"]);
    expect(
      commentAppIds(commentEvent({ tags: [["t", "yantr"], ["t", "yantr-ollama"], ["t", "yantr-open-webui"]] }))
    ).toEqual(["ollama", "open-webui"]);
  });

  it("is empty for an event with no per-app tag at all", () => {
    expect(commentAppIds(commentEvent({ tags: [["t", "yantr"]] }))).toEqual([]);
    expect(commentAppIds(commentEvent({ tags: [] }))).toEqual([]);
    expect(commentAppIds(null)).toEqual([]);
  });
});

describe("isAuthorYantrComment", () => {
  it("accepts a well-formed comment by the account", () => {
    expect(isAuthorYantrComment(commentEvent(), JACK_HEX, NOW)).toBe(true);
  });

  it("accepts a comment tagged for any app, not just one", () => {
    expect(isAuthorYantrComment(commentEvent(), JACK_HEX, NOW)).toBe(true);
    expect(isAuthorYantrComment(commentEvent({ tags: [["t", "yantr"], ["t", "yantr-open-webui"], ["expiration", String(NOW + COMMENT_EXPIRY_SECONDS)]] }), JACK_HEX, NOW)).toBe(
      true
    );
  });

  it("rejects another account's note", () => {
    const other = generateSecretKey();
    expect(isAuthorYantrComment(commentEvent({ pubkey: getPublicKey(other) }), JACK_HEX, NOW)).toBe(false);
  });

  it("rejects notes under the bare scope only — no per-app tag, no app comment", () => {
    expect(
      isAuthorYantrComment(commentEvent({ tags: [["t", "yantr"], ["expiration", String(NOW + COMMENT_EXPIRY_SECONDS)]] }), JACK_HEX, NOW)
    ).toBe(false);
  });

  it("rejects unsigned-shaped, untimely and expired events like isAppComment does", () => {
    expect(isAuthorYantrComment(null, JACK_HEX, NOW)).toBe(false);
    expect(isAuthorYantrComment(commentEvent({ created_at: NOW + 3600 }), JACK_HEX, NOW)).toBe(false);
    expect(isAuthorYantrComment(commentEvent({ tags: [["t", "yantr"], ["t", "yantr-ollama"], ["expiration", String(NOW - 1)]] }), JACK_HEX, NOW)).toBe(false);
    expect(isAuthorYantrComment(commentEvent({ content: "   " }), JACK_HEX, NOW)).toBe(false);
  });
});

describe("isYantrComment", () => {
  it("accepts any account's well-formed yantr comment — authorship is not a question here", () => {
    const other = generateSecretKey();
    expect(isYantrComment(commentEvent(), NOW)).toBe(true);
    expect(isYantrComment(commentEvent({ pubkey: getPublicKey(other) }), NOW)).toBe(true);
  });

  it("rejects a note under the bare scope only — the feed-wide REQ catches per-app notes", () => {
    expect(isYantrComment(commentEvent({ tags: [["t", "yantr"], ["expiration", String(NOW + COMMENT_EXPIRY_SECONDS)]] }), NOW)).toBe(false);
  });

  it("applies the same freshness, expiry and content rules as isAppComment does", () => {
    expect(isYantrComment(null, NOW)).toBe(false);
    expect(isYantrComment(commentEvent({ created_at: NOW + 3600 }), NOW)).toBe(false);
    expect(isYantrComment(commentEvent({ tags: [["t", "yantr"], ["t", "yantr-ollama"], ["expiration", String(NOW - 1)]] }), NOW)).toBe(false);
    expect(isYantrComment(commentEvent({ content: "   " }), NOW)).toBe(false);
  });
});

describe("yantrFeedFilter", () => {
  it("queries kind-1 notes on the bare yantr scope, any author", () => {
    const filter = yantrFeedFilter();
    expect(filter.kinds).toEqual([1]);
    expect(filter["#t"]).toEqual(["yantr"]);
    expect(filter).not.toHaveProperty("authors");
    expect(filter.limit).toBeGreaterThan(0);
  });

  it("clamps the limit into what relays answer sanely", () => {
    expect(yantrFeedFilter(0).limit).toBe(200);
    expect(yantrFeedFilter(-5).limit).toBe(200);
    expect(yantrFeedFilter(50).limit).toBe(50);
    expect(yantrFeedFilter(10_000).limit).toBe(500);
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

  // The regression this parser had: the payment fields were never read, so a
  // profile that published them showed nothing and the donation QR was dead.
  it("reads the payment fields a real profile publishes — verbatim from a live kind-0", () => {
    // Content of the account credited across the catalog, as the relays
    // served it (kind-0 89f8e11d…, 2026-08-14): NIP-05 and a NIP-57 lud16,
    // no structured on-chain field.
    const live = parseKind0Profile({
      ...JACK_PROFILE,
      content:
        '{"name":"besoeasy","display_name":"Aman Singh Kattal","about":"Building and optimizing financial systems.","website":"https://besoeasy.com","nip05":"besoeasy@zaps.lol","picture":"https://besoeasy.com/me.png","banner":"https://picsum.photos/800/600","lud16":"besoeasy@wallet.yakihonne.com"}',
    });
    expect(live.lud16).toBe("besoeasy@wallet.yakihonne.com");
    expect(live.nip05).toBe("besoeasy@zaps.lol");
    expect(live.bitcoin).toBe("");
  });

  it("accepts the lud06 spelling when no lud16 is published", () => {
    const p = parseKind0Profile({
      ...JACK_PROFILE,
      content: JSON.stringify({ name: "j", lud06: "Jack@Example.Test" }),
    });
    expect(p.lud16).toBe("jack@example.test");
  });

  it("refuses a Lightning address that would encode into a dead lookup", () => {
    for (const lud of ["not-an-address", "user@", "user@host", "user@host/path", "user @host.com"]) {
      const p = parseKind0Profile({ ...JACK_PROFILE, content: JSON.stringify({ lud16: lud }) });
      expect(p.lud16, lud).toBe("");
    }
  });

  it("reads an on-chain address from the metadata field or the profile text", () => {
    const ADDRESS = "bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqzk5jj0";
    expect(
      parseKind0Profile({ ...JACK_PROFILE, content: JSON.stringify({ bitcoin: ADDRESS }) }).bitcoin
    ).toBe(ADDRESS);
    expect(
      parseKind0Profile({ ...JACK_PROFILE, content: JSON.stringify({ about: `sats: ${ADDRESS}` }) })
        .bitcoin
    ).toBe(ADDRESS);
  });

  it("publishes no Bitcoin address for one that fails its checksum", () => {
    // One character changed from a valid BIP-86 vector: shape is perfect,
    // checksum is not, and a QR must not be built on it.
    const broken = "bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqzk5jj1";
    const p = parseKind0Profile({ ...JACK_PROFILE, content: JSON.stringify({ bitcoin: broken }) });
    expect(p.bitcoin).toBe("");
  });
});

describe("isLightningAddress", () => {
  it("normalises a valid address and rejects everything else", () => {
    expect(isLightningAddress("Besoeasy@Wallet.Yakihonne.COM")).toBe("besoeasy@wallet.yakihonne.com");
    expect(isLightningAddress("besoeasy@wallet.yakihonne.com")).toBe("besoeasy@wallet.yakihonne.com");
    expect(isLightningAddress("besoeasy@wallet")).toBe("");
    expect(isLightningAddress("besoeasy@.com")).toBe("");
    expect(isLightningAddress("")).toBe("");
  });
});

describe("newestKind0Profiles", () => {
  // Real signatures. This merge decides which name, picture and NIP-05 the UI
  // prints beside a comment or on a developer page, so an unsigned fixture
  // would be testing nothing at all.
  it("keeps the newest event per account", () => {
    const sk = generateSecretKey();
    const older = finalizeEvent(
      { kind: 0, created_at: 1700000000, tags: [], content: JSON.stringify({ name: "old" }) },
      sk
    );
    const newer = finalizeEvent(
      { kind: 0, created_at: 1700009999, tags: [], content: JSON.stringify({ name: "new" }) },
      sk
    );

    // Deliberately out of order: a relay may answer in any order, so "newest
    // wins" has to be decided by created_at rather than by arrival.
    const merged = newestKind0Profiles([newer, older]);
    expect(Object.keys(merged)).toEqual([older.pubkey]);
    expect(merged[older.pubkey].name).toBe("new");
  });

  it("drops an event whose signature does not verify", () => {
    const sk = generateSecretKey();
    const event = finalizeEvent(
      { kind: 0, created_at: 1700000000, tags: [], content: JSON.stringify({ name: "jack" }) },
      sk
    );
    expect(Object.keys(newestKind0Profiles([event]))).toEqual([event.pubkey]);

    // A relay impersonating an account is the whole threat: keep the victim's
    // pubkey, rewrite the content — but the signature no longer covers it.
    const forged = { ...event, content: JSON.stringify({ name: "admin" }) };
    expect(newestKind0Profiles([forged])).toEqual({});
  });

  it("ignores events of another kind and anything that is not an event", () => {
    const sk = generateSecretKey();
    const note = finalizeEvent(
      { kind: 1, created_at: 1700000000, tags: [], content: "hello" },
      sk
    );
    expect(newestKind0Profiles([note])).toEqual({});
    expect(newestKind0Profiles([null, undefined, {}, 42])).toEqual({});
    expect(newestKind0Profiles(undefined)).toEqual({});
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

describe("isAuthentic", () => {
  // Events are signed the way a NIP-07 signer signs them, so the verifier
  // faces genuine schnorr signatures rather than mocks.
  function signed(seed, overrides = {}) {
    const sk = new Uint8Array(32).fill(seed);
    return finalizeEvent(
      {
        kind: 1,
        pubkey: getPublicKey(sk),
        created_at: 1700000000,
        tags: [["t", "yantr"]],
        content: "hello",
        ...overrides,
      },
      sk
    );
  }

  // Rebuilt field-by-field on purpose: spreading an event copies nostr-tools'
  // verification cache symbol, which would make a changed event look verified.
  const vary = (event, changes) => ({
    id: event.id,
    pubkey: event.pubkey,
    created_at: event.created_at,
    kind: event.kind,
    tags: event.tags,
    content: event.content,
    sig: event.sig,
    ...changes,
  });

  it("accepts an event whose signature matches its id and pubkey", () => {
    expect(isAuthentic(signed(7))).toBe(true);
  });

  // The attack the signature exists to stop: a relay keeps the victim's
  // pubkey, rewrites the content, and recomputes the id to match.
  it("rejects impersonation with a recomputed id", () => {
    const event = signed(7);
    const forged = signed(7, { content: "trust me, this is the author" });
    const tampered = vary(event, { content: "trust me, this is the author", id: forged.id });
    expect(isAuthentic(tampered)).toBe(false);
  });

  it("rejects edited content, a swapped pubkey and a forged id", () => {
    const event = signed(7);
    expect(isAuthentic(vary(event, { content: "tampered" }))).toBe(false);
    expect(isAuthentic(vary(event, { pubkey: getPublicKey(new Uint8Array(32).fill(9)) }))).toBe(false);
    expect(isAuthentic(vary(event, { id: "00".repeat(32) }))).toBe(false);
  });

  it("rejects malformed hex, wrong sizes and missing fields", () => {
    expect(isAuthentic(null)).toBe(false);
    expect(isAuthentic({})).toBe(false);
    expect(isAuthentic({ id: "zz", sig: "zz", pubkey: "zz" })).toBe(false);
    // 31-byte pubkey, 63-byte id, 64-byte signature.
    expect(isAuthentic({ pubkey: "aa".repeat(31), id: "bb".repeat(32), sig: "cc".repeat(64) })).toBe(false);
    expect(isAuthentic({ pubkey: "aa".repeat(32), id: "bb".repeat(31), sig: "cc".repeat(64) })).toBe(false);
    expect(isAuthentic({ pubkey: "aa".repeat(32), id: "bb".repeat(32), sig: "cc".repeat(63) })).toBe(false);
  });
});
