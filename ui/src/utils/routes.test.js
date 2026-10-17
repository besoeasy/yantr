/**
 * routes.test.js — the guard the route table never had.
 *
 * The reason this file exists: `/apps/:appname` declared a param called
 * `appname` while AppDetail compared the route against `app.id`, and nothing
 * anywhere failed until a user opened the page. A table of pure data plus a
 * builder per route makes that mismatch checkable, so these assertions are
 * about names, params and ownership rather than about rendering.
 */
import { describe, expect, it } from "vitest";
import {
  NAV,
  ROUTE,
  appDetail,
  apps,
  containerDetail,
  dashboard,
  developerDetail,
  fleet,
  logs,
  routes,
  stackDetail,
  storageTab,
} from "./routes.js";

// The `:param` tokens a path declares, in the order they appear.
function declaredParams(path) {
  return [...path.matchAll(/:([A-Za-z0-9_]+)/g)].map((match) => match[1]);
}

const named = routes.filter((route) => route.name);
const byName = new Map(named.map((route) => [route.name, route]));

describe("route table", () => {
  it("declares no duplicate names", () => {
    const names = named.map((route) => route.name);
    expect(new Set(names).size).toBe(names.length);
  });

  it("declares no duplicate paths", () => {
    const paths = routes.map((route) => route.path);
    expect(new Set(paths).size).toBe(paths.length);
  });

  it("declares every name in ROUTE, so no constant dangles", () => {
    for (const name of Object.values(ROUTE)) {
      expect(byName.has(name), `${name} is not declared in routes`).toBe(true);
    }
  });

  it("titles every page, so history entries and tabs differ", () => {
    for (const route of named) {
      expect(route.meta?.title, `${route.name} has no meta.title`).toBeTruthy();
    }
  });

  it("only points meta.nav at a real sidebar item", () => {
    const navIds = new Set(Object.values(NAV));
    for (const route of named) {
      const nav = route.meta?.nav;
      if (nav === undefined) continue;
      expect(navIds.has(nav), `${route.name} names unknown nav "${nav}"`).toBe(true);
    }
  });

  it("gives every page an owning sidebar item", () => {
    for (const route of named) {
      if (route.name === ROUTE.NOT_FOUND) continue;
      expect(route.meta?.nav, `${route.name} has no meta.nav`).toBeTruthy();
    }
  });

  it("keeps the 404 out of the sidebar", () => {
    expect(byName.get(ROUTE.NOT_FOUND).meta?.nav).toBeUndefined();
  });

  it("ends with the catch-all", () => {
    expect(routes.at(-1).name).toBe(ROUTE.NOT_FOUND);
  });

  it("resolves a renamed path by name, not by URL", () => {
    // The test names already encode the contract: nothing in the app writes
    // "/apps/:appId" or "/fleet" as a string, so changing the path is safe.
    expect(byName.get(ROUTE.APP_DETAIL).path).toBe("/apps/:appId");
    expect(byName.get(ROUTE.CONTAINER_DETAIL).path).toBe("/containers/:containerId");
    expect(byName.get(ROUTE.DEVELOPER_DETAIL).path).toBe("/developers/:npub");
  });

  it("still answers the paths that were renamed", () => {
    // A hash URL is bookmarkable and users do bookmark their own dashboard, so
    // these must redirect rather than fall through to the 404.
    const redirecting = new Set(
      routes.filter((route) => route.redirect).map((route) => route.path),
    );
    expect(redirecting.has("/")).toBe(true);
    expect(redirecting.has("/home")).toBe(true);
    expect(redirecting.has("/presence")).toBe(true);
    expect(redirecting.has("/developer/:npub")).toBe(true);
  });
});

describe("builders", () => {
  const parameterised = [
    { build: appDetail, name: ROUTE.APP_DETAIL, param: "appId", value: "immich" },
    { build: stackDetail, name: ROUTE.STACK_DETAIL, param: "projectId", value: "yantr-immich" },
    { build: containerDetail, name: ROUTE.CONTAINER_DETAIL, param: "containerId", value: "9f2c" },
    { build: developerDetail, name: ROUTE.DEVELOPER_DETAIL, param: "npub", value: "npub1abc" },
  ];

  it.each(parameterised)(
    "$name supplies exactly the params its path declares",
    ({ build, name, param, value }) => {
      const route = byName.get(name);
      expect(route, `${name} is not declared in routes`).toBeTruthy();

      // This is the assertion that would have caught /apps/:appname: the path
      // asked for one name, the caller supplied another.
      expect(declaredParams(route.path)).toEqual([param]);

      const location = build(value);
      expect(location.name).toBe(name);
      expect(location.params).toEqual({ [param]: value });
    },
  );

  it.each(parameterised)("$name navigates by name, never by path", ({ build, value }) => {
    const location = build(value);
    expect(location).not.toHaveProperty("path");
  });

  const plain = [
    { label: "dashboard", build: dashboard, name: ROUTE.DASHBOARD },
    { label: "apps", build: apps, name: ROUTE.APPS },
    { label: "logs", build: logs, name: ROUTE.LOGS },
    { label: "fleet", build: fleet, name: ROUTE.FLEET },
  ];

  it.each(plain)("$label targets a declared pathless route", ({ build, name }) => {
    expect(byName.has(name), `${name} is not declared in routes`).toBe(true);
    expect(declaredParams(byName.get(name).path)).toEqual([]);
    expect(build()).toEqual({ name });
  });

  it("only puts a tab in the query when one is given", () => {
    expect(storageTab()).toEqual({ name: ROUTE.STORAGE });
    expect(storageTab("volumes-unused")).toEqual({
      name: ROUTE.STORAGE,
      query: { tab: "volumes-unused" },
    });
  });
});
