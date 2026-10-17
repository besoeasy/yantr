import { describe, expect, it } from "vitest";
import {
  appsByDeveloper,
  buildIssueReportUrl,
  buildYantrIssueReportUrl,
  DEVELOPER_LABEL,
  developerInitials,
  developerProfileUrl,
  developerRouteRef,
  developerShortKey,
  developersOf,
  isNostrKey,
  ISSUE_BASE,
  nostrDmUri,
} from "./developer";

// The same externally-verified keys core/nip19 tests against.
const NPUB = "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6";
const JACK = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m";
const NPROFILE =
  "nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpz3mhxue69uhhyetvv9ujuerpd46hxtnfduqs6amnwvaz7tmwdaejumr0dsfg70ls";

describe("DEVELOPER_LABEL", () => {
  it("names the one kind of developer there is", () => {
    expect(DEVELOPER_LABEL).toBe("Nostr");
  });
});

describe("developerProfileUrl", () => {
  it("routes a key through njump, which hands the reader to their own client", () => {
    expect(developerProfileUrl(NPUB)).toBe(`https://njump.me/${NPUB}`);
    expect(developerProfileUrl(NPROFILE)).toBe(`https://njump.me/${NPROFILE}`);
  });

  // Never a link to nowhere: a dead profile link is worse than no credit.
  it("returns nothing for a value that is not a key", () => {
    expect(developerProfileUrl("besoeasy")).toBe("");
    expect(developerProfileUrl("")).toBe("");
    expect(developerProfileUrl(undefined)).toBe("");
  });
});

describe("isNostrKey", () => {
  it("recognises the two profile identifiers", () => {
    expect(isNostrKey(NPUB)).toBe(true);
    expect(isNostrKey(NPROFILE)).toBe(true);
    expect(isNostrKey(`  ${NPUB}  `)).toBe(true);
  });

  // note1 is an event id and nsec1 a private key. Neither names a person, and
  // drawing a code for one would send a reader somewhere that cannot work.
  it("rejects everything that is not a profile identifier", () => {
    expect(isNostrKey("note1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q7k28gn")).toBe(false);
    expect(isNostrKey("nsec1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qr2zmhw")).toBe(false);
    // The forge logins the field used to accept.
    expect(isNostrKey("besoeasy")).toBe(false);
    expect(isNostrKey("github:besoeasy")).toBe(false);
    expect(isNostrKey("")).toBe(false);
    expect(isNostrKey(null)).toBe(false);
    // bech32 omits the digits 1 and the letters b, i and o, so a key containing
    // them is not a key however plausible it looks.
    expect(isNostrKey("npub1" + "b".repeat(58))).toBe(false);
  });
});

describe("nostrDmUri", () => {
  it("builds the NIP-21 URI a client opens", () => {
    expect(nostrDmUri(NPUB)).toBe(`nostr:${NPUB}`);
    expect(nostrDmUri(NPROFILE)).toBe(`nostr:${NPROFILE}`);
  });

  // bech32 is defined lowercase, NIP-19 exchanges it lowercase, and the catalog
  // is allowed to keep whatever case it was written with.
  it("lowercases the key so the code carries the canonical form", () => {
    expect(nostrDmUri(NPUB.toUpperCase())).toBe(`nostr:${NPUB}`);
    expect(nostrDmUri(`  ${JACK}  `)).toBe(`nostr:${JACK}`);
  });

  it("returns nothing for a value it cannot encode", () => {
    expect(nostrDmUri("besoeasy")).toBe("");
    expect(nostrDmUri("")).toBe("");
    expect(nostrDmUri(undefined)).toBe("");
  });
});

describe("developerInitials", () => {
  // A key has no words in it, so the badge comes from the identifier itself and
  // has to be stable for the same person.
  it("is derived from the identifier and never empty", () => {
    expect(developerInitials(NPUB)).toBe("NP");
    expect(developerInitials(JACK)).toBe("NP");
    expect(developerInitials("")).toBe("?");
    expect(developerInitials(null)).toBe("?");
  });
});

describe("developerShortKey", () => {
  // 63 characters is not a label. What a reader needs is enough to recognise the
  // kind of key it is and to check it against one they already hold.
  it("abbreviates a key and keeps both ends", () => {
    const short = developerShortKey(NPUB);
    expect(short).toBe("npub180cvv0…h6w6");
    expect(short.startsWith("npub1")).toBe(true);
    expect(short.endsWith(NPUB.slice(-4))).toBe(true);
    expect(short.length).toBeLessThan(NPUB.length);
  });

  it("leaves a short value alone", () => {
    expect(developerShortKey("npub1abc")).toBe("npub1abc");
    expect(developerShortKey("")).toBe("");
    expect(developerShortKey(undefined)).toBe("");
  });
});

describe("developerRouteRef", () => {
  it("is the key itself, since there is no platform prefix to carry", () => {
    expect(developerRouteRef(NPUB)).toBe(NPUB);
    expect(developerRouteRef(`  ${NPUB} `)).toBe(NPUB);
  });
});

describe("appsByDeveloper", () => {
  const apps = [
    { id: "a", developers: [{ key: NPUB, url: `https://njump.me/${NPUB}` }] },
    { id: "b", developers: [{ key: JACK, url: `https://njump.me/${JACK}` }] },
    { id: "c", developers: [{ key: NPUB }, { key: JACK }] },
    { id: "d" },
  ];

  it("filters to the apps crediting that key", () => {
    expect(appsByDeveloper(apps, NPUB).map((a) => a.id)).toEqual(["a", "c"]);
    expect(appsByDeveloper(apps, JACK).map((a) => a.id)).toEqual(["b", "c"]);
    expect(appsByDeveloper(apps, "npub1gwnvhmyryy8wqgfttcgk7adkvfxrynvy556qex4dvf94yptc2r6q0z3p66")).toEqual([]);
  });

  // Two spellings of one account are one account, so they must not each match a
  // differently-cased entry in the catalog.
  it("compares case-insensitively, like the catalog does", () => {
    expect(appsByDeveloper(apps, NPUB.toUpperCase()).map((a) => a.id)).toEqual(["a", "c"]);
  });

  it("survives a missing or malformed list", () => {
    expect(appsByDeveloper(undefined, NPUB)).toEqual([]);
    expect(appsByDeveloper([{ id: "x" }], NPUB)).toEqual([]);
    expect(appsByDeveloper(apps, "")).toEqual([]);
    expect(appsByDeveloper(apps, null)).toEqual([]);
  });
});

describe("developersOf", () => {
  it("keeps only entries that carry a key", () => {
    expect(developersOf({ developers: [{ key: "npub1x" }] })).toEqual([{ key: "npub1x" }]);
  });

  // The catalog serialises a non-null array, but a cached payload from an older
  // version or a hand-rolled /api/apps response may not have the field at all.
  it("returns an empty list rather than throwing", () => {
    expect(developersOf({})).toEqual([]);
    expect(developersOf(null)).toEqual([]);
    expect(developersOf({ developers: "npub1x" })).toEqual([]);
    expect(developersOf({ developers: [null, {}, { url: "u" }] })).toEqual([]);
  });
});

describe("buildIssueReportUrl", () => {
  it("names the app and its key in the body", () => {
    const url = buildIssueReportUrl({ id: "gupt", name: "GUPT", developers: [{ key: NPUB }] });
    expect(url.startsWith(ISSUE_BASE)).toBe(true);

    const body = decodeURIComponent(url.split("body=")[1]);
    expect(body).toContain("App: GUPT (gupt)");
    expect(body).toContain(`Author Nostr key: ${NPUB}`);
  });

  // A filed report used to @mention a GitHub login, which subscribed the author
  // automatically. A GitHub issue cannot mention a Nostr account, so the key is
  // information now — and there is no @ left in the body to do anything.
  it("carries no mention, because a Nostr key cannot be pinged", () => {
    const url = buildIssueReportUrl({ id: "gupt", name: "GUPT", developers: [{ key: NPUB }] });
    const body = decodeURIComponent(url.split("body=")[1]);
    expect(body).not.toContain("@");
  });

  it("still works with no app and with extra context lines", () => {
    expect(buildIssueReportUrl(null)).toBe(ISSUE_BASE);
    expect(buildIssueReportUrl(undefined)).toBe(ISSUE_BASE);

    const url = buildIssueReportUrl(
      { id: "gupt", name: "GUPT" },
      { prefix: "app", extraLines: ["Stack: 7"] }
    );
    expect(decodeURIComponent(url)).toContain("[app: GUPT]");
    expect(decodeURIComponent(url.split("body=")[1])).toContain("Stack: 7");
  });
});
describe("buildYantrIssueReportUrl", () => {
  it("fills in the facts a maintainer would otherwise have to ask for", () => {
    const url = buildYantrIssueReportUrl({
      version: "20261017",
      build: "2026-10-17",
      host: "Fedora · linux/amd64 · 24 cores",
      podman: "5.4.1",
      stacks: "2",
    });

    expect(url.startsWith(ISSUE_BASE)).toBe(true);
    const body = decodeURIComponent(url.split("body=")[1]);
    expect(body).toContain("Yantr: 20261017");
    expect(body).toContain("Build: 2026-10-17");
    expect(body).toContain("Host: Fedora · linux/amd64 · 24 cores");
    expect(body).toContain("Podman: 5.4.1");
    expect(body).toContain("Stacks running: 2");
    expect(body).toContain("<describe what happened>");
  });

  // A card that fills itself in must not carry the machine it came from. These
  // are the fields a report has no business having, and a future caller must not
  // be able to add them by passing something extra.
  it("carries no hostname, address or user", () => {
    const body = decodeURIComponent(
      buildYantrIssueReportUrl({
        version: "20261017",
        host: "Fedora",
        hostname: "nas-in-the-attic",
        ip: "203.0.113.9",
        user: "alice",
      }).split("body=")[1]
    );

    expect(body).not.toContain("nas-in-the-attic");
    expect(body).not.toContain("203.0.113.9");
    expect(body).not.toContain("alice");
  });

  it("skips what it does not know instead of printing empty fields", () => {
    const body = decodeURIComponent(
      buildYantrIssueReportUrl({ version: "20261017", host: "  ", podman: null }).split("body=")[1]
    );

    expect(body).toContain("Yantr: 20261017");
    expect(body).not.toContain("Host:");
    expect(body).not.toContain("Podman:");
    expect(body).not.toContain("Build:");
  });

  // A zero is a real answer — no stacks running is a fact worth having in a bug
  // report — so only genuinely absent values are dropped.
  it("keeps a zero", () => {
    const body = decodeURIComponent(
      buildYantrIssueReportUrl({ version: "20261017", stacks: "0" }).split("body=")[1]
    );
    expect(body).toContain("Stacks running: 0");
  });

  it("still produces a usable report with nothing to say", () => {
    const url = buildYantrIssueReportUrl();
    expect(url.startsWith(ISSUE_BASE)).toBe(true);
    expect(decodeURIComponent(url.split("body=")[1])).toContain("<describe what happened>");
  });
});
