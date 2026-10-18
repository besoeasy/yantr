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

function readCols(source) {
  // The declaration lives in the plain <script> block, not <script setup>,
  // which is what makes it a module export rather than component state.
  const block = source.match(/<script>([\s\S]*?)<\/script>/);
  if (!block) return null;
  const match = block[1].match(/cols:\s*(\d+)/);
  return match ? Number(match[1]) : null;
}

const widgets = readdirSync(widgetsDir)
  .filter((name) => name.endsWith(".vue"))
  .map((name) => ({
    name,
    cols: readCols(readFileSync(join(widgetsDir, name), "utf8")),
  }));

describe("dashboard widgets", () => {
  it("declares a width for every widget", () => {
    const silent = widgets.filter((w) => w.cols === null).map((w) => w.name);
    expect(silent, "widgets without `export const meta = { cols: N }`").toEqual([]);
  });

  it("never spans more than two columns of the desktop grid", () => {
    const wide = widgets.filter((w) => w.cols > MAX_COLS).map((w) => `${w.name} (cols: ${w.cols})`);
    expect(wide, `widgets wider than ${MAX_COLS} columns — see AGENTS.md §8`).toEqual([]);
  });
});