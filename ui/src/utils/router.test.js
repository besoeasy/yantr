/**
 * router.test.js — the route table under a real vue-router.
 *
 * routes.test.js checks the table as data. This checks that vue-router itself
 * agrees with it: that the legacy paths actually redirect, that the catch-all
 * catches, and that a builder plus its params lands on the path the table
 * declares. Every case navigates for real rather than only resolving, so a
 * route that cannot load its component fails here too.
 *
 * It builds its own router on memory history rather than importing router.js,
 * which calls createWebHashHistory at module load and so needs a DOM.
 */
import { describe, expect, it } from "vitest";
import { createMemoryHistory, createRouter } from "vue-router";
import { ROUTE, appDetail, containerDetail, developerDetail, routes, stackDetail } from "./routes.js";

function makeRouter() {
  return createRouter({ history: createMemoryHistory(), routes });
}

describe("redirects", () => {
  it.each([
    ["/", "/dashboard", ROUTE.DASHBOARD],
    ["/home", "/dashboard", ROUTE.DASHBOARD],
    ["/presence", "/fleet", ROUTE.FLEET],
    ["/stacks", "/dashboard", ROUTE.DASHBOARD],
    ["/developer/npub1abc", "/developers/npub1abc", ROUTE.DEVELOPER_DETAIL],
  ])("%s lands on %s", async (from, path, name) => {
    const router = makeRouter();
    await router.push(from);
    expect(router.currentRoute.value.path).toBe(path);
    expect(router.currentRoute.value.name).toBe(name);
  });
});

describe("navigation", () => {
  it("catches an unknown hash instead of matching nothing", async () => {
    const router = makeRouter();
    await router.push("/nope/at/all");
    expect(router.currentRoute.value.name).toBe(ROUTE.NOT_FOUND);
  });

  it.each([
    ["app-detail", appDetail("immich"), "/apps/immich"],
    ["stack-detail", stackDetail("yantr-immich"), "/stacks/yantr-immich"],
    ["container-detail", containerDetail("9f2c"), "/containers/9f2c"],
    ["developer-detail", developerDetail("npub1abc"), "/developers/npub1abc"],
  ])("resolves %s to %s", async (name, location, path) => {
    const router = makeRouter();
    expect(router.resolve(location).path).toBe(path);
    await router.push(location);
    expect(router.currentRoute.value.name).toBe(name);
    expect(router.currentRoute.value.path).toBe(path);
  });

  it("keeps the storage tab in the query, where the view reads it", async () => {
    const router = makeRouter();
    await router.push({ name: ROUTE.STORAGE, query: { tab: "volumes-unused" } });
    expect(router.currentRoute.value.path).toBe("/storage");
    expect(router.currentRoute.value.query.tab).toBe("volumes-unused");
  });

  it("titles the route it lands on", async () => {
    const router = makeRouter();
    await router.push(stackDetail("yantr-immich"));
    expect(router.currentRoute.value.meta.title).toBe("Stack");
  });
});
