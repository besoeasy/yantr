import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// A dashboard widget declares its width by exporting `meta = { cols: N }`
// from the plain <script> block beside its <script setup>. Home.vue clamps N
// to the grid's four columns and maps it to .w-span-N, so 3 and 4 are
// accepted by the engine.
//
// They are not wanted. The grid is `grid-cols-1 md:grid-cols-2
// lg:grid-cols-4` with `grid-flow-dense`, and the widgets are shuffled once
// a day: a card that takes three or four columns is the entire row, so where
// it lands changes every other card's position from one day to the next. And
// a card whose content only reads at full width is a page section wearing a
// card's chrome — the developer band was one, until it was rebuilt to read
// at 632px.
//
// Hence: one or two columns, and if two does not fit the content, the
// content is the thing that changes. See AGENTS.md §8.

const MAX_COLS = 2;

const widgetsDir = join(dirname(fileURLToPath(import.meta.url)), "..", "Widgets");

function readSource(name) {
  return readFileSync(join(widgetsDir, name), "utf8");
}

function readCols(source) {
  // The declaration lives in the plain <script> block, not <script setup>,
  // which is what makes it a module export rather than component state.
  const block = source.match(/<script>([\s\S]*?)<\/script>/);
  if (!block) return null;
  const match = block[1].match(/cols:\s*(\d+)/);
  return match ? Number(match[1]) : null;
}

// The card's surface — background, hairline border, corner radius, hover lift
// — comes from the .yantra-card token, not from a widget restating zinc steps
// or its own `rounded-2xl`. Eleven roots once lost it: a migration stripped
// `border-zinc-200 bg-white dark:bg-black rounded-2xl` without adding the
// token back, and those cards rendered transparent, unbordered and
// square-cornered, which the dark-mode screenshots did not catch.
//
// Every ROOT is checked, not just the first. A widget that renders an error
// or an empty state as a sibling <section> is still a card on the dashboard,
// and two of those had lost the surface while their sibling kept it.
//
// Roots are compared as tokens, never as substrings: "yantra-card" is a
// substring of "yantra-card-min", so text matching passes a root that has
// only the height.
function rootClassLists(source) {
  const template = source.slice(source.indexOf("<template>"));
  const tag = /<(\/?)([a-z][a-z0-9]*)((?:"[^"]*"|[^>"])*?)(\/?)>/g;
  const skip = new Set(["template", "style", "script"]);
  const lists = [];
  let depth = 0;
  let match;
  while ((match = tag.exec(template)) !== null) {
    const [, closing, name, attrs, selfClose] = match;
    if (skip.has(name)) continue;
    if (closing) {
      depth -= 1;
      continue;
    }
    if (depth === 0) {
      const cls = attrs.match(/class="([^"]*)"/);
      lists.push(cls ? cls[1].split(/\s+/) : []);
    }
    if (!selfClose) depth += 1;
  }
  return lists;
}

const widgets = readdirSync(widgetsDir)
  .filter((name) => name.endsWith(".vue"))
  .map((name) => {
    const source = readSource(name);
    return { name, cols: readCols(source), roots: rootClassLists(source) };
  });

/** Widget names whose roots are all missing `token`. */
function missing(token) {
  const names = [];
  for (const widget of widgets) {
    if (widget.roots.some((classes) => !classes.includes(token))) names.push(widget.name);
  }
  return [...new Set(names)];
}

describe("dashboard widgets", () => {
  it("declares a width for every widget", () => {
    const silent = widgets.filter((w) => w.cols === null).map((w) => w.name);
    expect(silent, "widgets without `export const meta = { cols: N }`").toEqual([]);
  });

  it("never spans more than two columns of the desktop grid", () => {
    const wide = widgets.filter((w) => w.cols > MAX_COLS).map((w) => `${w.name} (cols: ${w.cols})`);
    expect(wide, `widgets wider than ${MAX_COLS} columns — see AGENTS.md §8`).toEqual([]);
  });

  it("draws every root from the card token", () => {
    expect(missing("yantra-card"), "roots without `yantra-card` — no background, no border").toEqual([]);
  });

  it("keeps the shared minimum height on every root", () => {
    expect(missing("yantra-card-min"), "roots without `yantra-card-min` — ragged grid rows").toEqual([]);
  });

  // --card-radius is the token now. A root that also carries a `rounded-*`
  // utility wins by source order, so the two would disagree silently.
  it("leaves the corner radius to the token", () => {
    const loud = [];
    for (const widget of widgets) {
      if (widget.roots.some((classes) => classes.some((c) => c.startsWith("rounded-")))) {
        loud.push(widget.name);
      }
    }
    expect(loud, "roots setting their own radius instead of --card-radius").toEqual([]);
  });
});