import { describe, expect, it } from "vitest";
import {
  appsByDeveloper,
  buildIssueReportUrl,
  canFetchProfile,
  developerInitials,
  developerLabel,
  developerProfileUrl,
  developerRouteRef,
  developersOf,
  ISSUE_BASE,
  parseDeveloperRef,
} from "./developer";

describe("parseDeveloperRef", () => {
  it("defaults a bare handle to GitHub", () => {
    expect(parseDeveloperRef("besoeasy")).toEqual({ platform: "github", handle: "besoeasy" });
  });

  it("splits an explicit platform", () => {
    expect(parseDeveloperRef("gitlab:username2")).toEqual({
      platform: "gitlab",
      handle: "username2",
    });
    expect(parseDeveloperRef("nostr:npub1abc")).toEqual({
      platform: "nostr",
      handle: "npub1abc",
    });
  });

  it("tolerates padding and the @ spelling", () => {
    expect(parseDeveloperRef("  GitHub:@Besoeasy  ")).toEqual({
      platform: "github",
      handle: "Besoeasy",
    });
  });

  // The route renders an unknown reference as "no app credits this developer",
  // so anything uninterpretable has to be null rather than a GitHub login.
  it("rejects what it cannot interpret", () => {
    for (const bad of ["", "   ", "github:", "github:@", ":", "mastodon:someone", null, undefined]) {
      expect(parseDeveloperRef(bad)).toBeNull();
    }
  });

  // A handle with a colon in it would otherwise be silently truncated at the
  // first separator, which is how "https://github.com/x" became a developer.
  it("keeps everything after the first colon as the handle", () => {
    expect(parseDeveloperRef("nostr:npub1:a")).toEqual({ platform: "nostr", handle: "npub1:a" });
  });
});

describe("developerProfileUrl", () => {
  it("derives a profile URL per platform", () => {
    expect(developerProfileUrl("github", "besoeasy")).toBe("https://github.com/besoeasy");
    expect(developerProfileUrl("gitlab", "x")).toBe("https://gitlab.com/x");
    expect(developerProfileUrl("codeberg", "x")).toBe("https://codeberg.org/x");
    expect(developerProfileUrl("nostr", "npub1")).toBe("https://njump.me/npub1");
  });

  // An unknown platform must never produce a link to nowhere.
  it("returns empty for an unknown platform", () => {
    expect(developerProfileUrl("mastodon", "x")).toBe("");
  });
});

describe("developerLabel and canFetchProfile", () => {
  it("names the platforms", () => {
    expect(developerLabel("github")).toBe("GitHub");
    expect(developerLabel("nostr")).toBe("Nostr");
  });

  it("only claims a profile API for GitHub", () => {
    expect(canFetchProfile("github")).toBe(true);
    expect(canFetchProfile("gitlab")).toBe(false);
    expect(canFetchProfile("nostr")).toBe(false);
  });
});

describe("developerInitials", () => {
  it("always answers with something", () => {
    expect(developerInitials("besoeasy")).toBe("BE");
    expect(developerInitials("fclairamb")).toBe("FC");
    expect(developerInitials("")).toBe("?");
    expect(developerInitials(undefined)).toBe("?");
  });

  it("ignores punctuation so a nostr key is not just its prefix", () => {
    expect(developerInitials("npub1-abc")).toBe("NP");
  });
});

describe("appsByDeveloper", () => {
  const apps = [
    { id: "a", developers: [{ platform: "github", handle: "besoeasy" }] },
    { id: "b", developers: [{ platform: "gitlab", handle: "someone" }] },
    { id: "c", developers: [] },
    { id: "d" },
  ];

  it("matches on platform and handle", () => {
    expect(appsByDeveloper(apps, "github", "besoeasy").map((a) => a.id)).toEqual(["a"]);
    expect(appsByDeveloper(apps, "gitlab", "someone").map((a) => a.id)).toEqual(["b"]);
    expect(appsByDeveloper(apps, "github", "someone")).toEqual([]);
  });

  // GitHub handles are case-insensitive, so the route must not miss an app
  // because of how the handle was typed.
  it("ignores case on the forge platforms", () => {
    expect(appsByDeveloper(apps, "github", "Besoeasy").map((a) => a.id)).toEqual(["a"]);
  });

  it("survives an apps payload with no developers field", () => {
    expect(appsByDeveloper(undefined, "github", "x")).toEqual([]);
    expect(appsByDeveloper([{ id: "x" }], "github", "x")).toEqual([]);
  });
});

describe("developersOf", () => {
  it("returns the declared developers", () => {
    expect(
      developersOf({ developers: [{ platform: "github", handle: "x", url: "u" }] })
    ).toHaveLength(1);
  });

  // An older /api/apps payload, or one served by a host still on the previous
  // build, has no developers field at all.
  it("returns empty for a missing or malformed field", () => {
    expect(developersOf({})).toEqual([]);
    expect(developersOf(null)).toEqual([]);
    expect(developersOf({ developers: "github:x" })).toEqual([]);
    expect(developersOf({ developers: [null, {}, { platform: "github" }] })).toEqual([]);
  });
});

describe("developerRouteRef", () => {
  // GitHub is the default platform, so it gets the short form.
  it("shortens the GitHub route", () => {
    expect(developerRouteRef("github", "besoeasy")).toBe("besoeasy");
  });

  it("qualifies every other platform", () => {
    expect(developerRouteRef("gitlab", "someone")).toBe("gitlab:someone");
    expect(developerRouteRef("nostr", "npub1")).toBe("nostr:npub1");
  });

  // Whatever the chip links to has to parse back to the same developer, or the
  // page it opens is about somebody else.
  it("round-trips through parseDeveloperRef", () => {
    for (const [platform, handle] of [
      ["github", "besoeasy"],
      ["gitlab", "someone"],
      ["codeberg", "x"],
      ["nostr", "npub1abc"],
    ]) {
      expect(parseDeveloperRef(developerRouteRef(platform, handle))).toEqual({
        platform,
        handle,
      });
    }
  });
});

describe("buildIssueReportUrl", () => {
  const app = {
    id: "gupt",
    name: "GUPT",
    developers: [
      { platform: "github", handle: "besoeasy" },
      { platform: "gitlab", handle: "other" },
    ],
  };

  function decode(url) {
    const [base, query] = url.split("?");
    const params = new URLSearchParams(query);
    return { base, title: params.get("title"), body: params.get("body") };
  }

  it("mentions the app's GitHub authors in the body", () => {
    const { base, title, body } = decode(
      buildIssueReportUrl(app, { prefix: "pre-install" })
    );
    expect(base).toBe(ISSUE_BASE);
    expect(title).toBe("[pre-install: GUPT] ");
    expect(body).toContain("App: GUPT (gupt)");
    expect(body).toContain("Author: @besoeasy");
  });

  // An @mention only resolves on GitHub, so a GitLab or Nostr handle must not be
  // presented as one.
  it("does not mention non-GitHub developers", () => {
    const { body } = decode(buildIssueReportUrl(app));
    expect(body).not.toContain("@other");
    expect(body).not.toContain("gitlab");
  });

  it("omits the author line when nobody is credited", () => {
    const { body } = decode(buildIssueReportUrl({ id: "x", name: "X" }));
    expect(body).not.toContain("Author:");
  });

  it("carries extra context lines", () => {
    const { body } = decode(
      buildIssueReportUrl(app, { extraLines: ["Stack: gupt_abc"] })
    );
    expect(body).toContain("Stack: gupt_abc");
    expect(body).toContain("<describe the problem>");
  });

  it("falls back to the bare tracker when there is no app", () => {
    expect(buildIssueReportUrl(null)).toBe(ISSUE_BASE);
  });

  // The whole URL has to survive a round trip through encodeURIComponent, or a
  // copied link arrives with a mangled body.
  it("escapes the body so the link stays intact", () => {
    const url = buildIssueReportUrl(app);
    expect(url).not.toContain("\n");
    expect(decode(url).body).toContain("@besoeasy");
  });
});
