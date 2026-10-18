import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

// Every class a component asks for must exist in the curated utility layer
// (src/assets/main.css). There is no Tailwind build to catch the ones that
// do not: an undefined class is not an error, it is silence — a shelf
// divider that never appears, a hover that never fires, a control stretched
// to full width. That silence cost three rounds of pixel-reviewing before
// this check existed, so it is checked here instead.

const SRC = fileURLToPath(new URL("..", import.meta.url));
const CSS = readFileSync(join(SRC, "assets/main.css"), "utf8");

// CSS escapes the characters that would otherwise end a selector. This
// mirrors that: `dark:lg:border-zinc-800` is written `.dark\:lg\:border-zinc-800`.
function cssEscape(token) {
  return token.replace(/([:/.%#[\]!$])/g, "\\$1");
}

const DEFINED = new Set();
for (const match of CSS.matchAll(/\.((?:\\.|[A-Za-z0-9_#.[\]%()-])+)/g)) {
  // A CSS escape is either `\x` for one character or `\32xl` style hex for a
  // whole one — the class `2xl:grid-cols-5` is written `.\32xl\:grid-cols-5`.
  DEFINED.add(match[1].replace(/\\([0-9a-fA-F]{1,6})\s?/g, (_, hex) =>
    String.fromCodePoint(parseInt(hex, 16))
  ).replace(/\\(.)/g, "$1"));
}

function vueFiles(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) {
      if (["node_modules", "dist"].includes(entry)) continue;
      vueFiles(path, out);
    } else if (path.endsWith(".vue")) {
      out.push(path);
    }
  }
  return out;
}

// A class token: what a browser would treat as one name. Anything else is a
// Vue expression (`isActive(item)`, `item.isUrgent`, the `b` branch of a
// ternary) and is not this test's business. Parentheses are allowed because
// the `bg-(--token)` custom-property form is a real class.
const CLASS_LIKE = /^[A-Za-z0-9][A-Za-z0-9_:/.\-[\]#%()]*$/;

function looksLikeClass(token) {
  if (token.length < 2) return false; // single letters are ternary branches
  if (!CLASS_LIKE.test(token)) return false;
  if (token.includes("${")) return false;
  if (/^[A-Za-z_$][\w$]*\.[\w$]/.test(token)) return false; // dotted JS
  // A call, not the custom-property form: `isActive(` or `.trim(`.
  if (/\(|\)$/.test(token) && !token.includes("--(")) return false;
  return true;
}

function classesIn(source) {
  const found = new Set();

  // Static attributes only: `class="…"`, not `:class="…"` (the lookbehind
  // rejects the binding colon, and any preceding word character).
  for (const [, attr] of source.matchAll(/(?<![\w:-])class="([^"]*)"/g)) {
    for (const token of attr.split(/\s+/)) {
      const clean = token.trim();
      if (looksLikeClass(clean)) found.add(clean);
    }
  }

  // :class bindings resolve at runtime, so only their literal string parts can
  // be checked — and a template literal's interpolated part cannot, so those
  // tokens are left out rather than guessed at.
  for (const [, expr] of source.matchAll(/:class="([^"]*)"/g)) {
    for (const [, literal] of expr.matchAll(/'([^']*)'|"([^"]*)"/g)) {
      const text = literal[1] ?? literal[2] ?? "";
      for (const token of text.split(/\s+/)) {
        if (looksLikeClass(token.trim())) found.add(token.trim());
      }
    }
  }

  return found;
}

function stylesIn(source) {
  // A component may carry its own styles in a <style> block — .gh-cta and
  // .report-cta are component classes, not utilities — so those count as
  // defined for that component only.
  const defined = new Set();
  for (const [, block] of source.matchAll(/<style[^>]*>([\s\S]*?)<\/style>/g)) {
    for (const match of block.matchAll(/\.((?:\\.|[A-Za-z0-9_#.[\]%()-])+)/g)) {
      defined.add(match[1].replace(/\\([0-9a-fA-F]{1,6})\s?/g, (_, hex) =>
        String.fromCodePoint(parseInt(hex, 16))
      ).replace(/\\(.)/g, "$1"));
    }
  }
  return defined;
}

const FILES = vueFiles(SRC);
const UNDEFINED = [];
for (const path of FILES) {
  const source = readFileSync(path, "utf8");
  const local = stylesIn(source);
  for (const token of classesIn(source)) {
    if (!DEFINED.has(cssEscape(token)) && !DEFINED.has(token) && !local.has(cssEscape(token)) && !local.has(token)) {
      UNDEFINED.push(`${path.slice(SRC.length)}: ${token}`);
    }
  }
}

describe("curated utility layer", () => {
  it("is not empty and is the file the app actually loads", () => {
    expect(CSS.length).toBeGreaterThan(10_000);
    expect(FILES.length).toBeGreaterThan(10);
  });

  it("defines every class the components use", () => {
    // Printed as one message so a batch of missing classes is one failure to
    // fix, not thirty.
    expect(UNDEFINED, `classes with no rule in main.css:\n${UNDEFINED.join("\n")}`).toEqual([]);
  });
});