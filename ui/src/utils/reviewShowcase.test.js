import { describe, expect, it } from "vitest";
import {
  SHOWCASE_LINE_BUDGET,
  SHOWCASE_LINE_CLAMP,
  SHOWCASE_MAX_REVIEWS,
  linesFor,
  showcaseReviews,
} from "./reviewShowcase";

function review(id, contentLength) {
  return { id, content: "x".repeat(contentLength) };
}

describe("linesFor", () => {
  it("counts at least one line and never more than the display clamp", () => {
    expect(linesFor("")).toBe(1);
    expect(linesFor(null)).toBe(1);
    expect(linesFor("x".repeat(46))).toBe(1);
    expect(linesFor("x".repeat(47))).toBe(2);
    expect(linesFor("x".repeat(10_000))).toBe(SHOWCASE_LINE_CLAMP);
  });
});

describe("showcaseReviews", () => {
  it("shows reviews in order, newest first, no reordering", () => {
    const reviews = [review("a", 10), review("b", 10), review("c", 10)];
    const picked = showcaseReviews(reviews);
    expect(picked).toContain("a");
    // In-order: nothing picked can skip an earlier pick.
    const first = picked.indexOf("a");
    expect(picked.slice(first + 1).every((id) => reviews.slice(first + 1).some((r) => r.id === id))).toBe(true);
  });

  it("packs one-liners by the handful — up to the cap", () => {
    const reviews = [review("a", 10), review("b", 10), review("c", 10), review("d", 10), review("e", 10)];
    expect(showcaseReviews(reviews)).toHaveLength(SHOWCASE_MAX_REVIEWS);
  });

  it("fits two long reviews and a gap-filling short one inside the budget", () => {
    // 3 lines + 2 header = 5 for a long review, +3 for the short one, +5
    // again for the second long one → 13 of 14. A fourth long review would
    // push past, and the cap is four regardless.
    const reviews = [review("long1", 5_000), review("short", 20), review("long2", 5_000), review("long3", 5_000)];
    expect(showcaseReviews(reviews)).toEqual(["long1", "short", "long2"]);
  });

  it("shows two long reviews when nothing shorter follows", () => {
    const reviews = [review("a", 5_000), review("b", 5_000), review("c", 5_000)];
    expect(showcaseReviews(reviews)).toEqual(["a", "b"]);
  });

  it("always shows the first review, however long, and only one when it eats the budget", () => {
    const huge = [review("huge", 100_000)];
    expect(showcaseReviews(huge)).toEqual(["huge"]);
  });

  it("respects the budget end-to-end: nothing shown can exceed it once clamped", () => {
    const reviews = [review("a", 20), review("b", 200), review("c", 20), review("d", 140)];
    const picked = showcaseReviews(reviews);
    const shown = reviews.filter((r) => picked.includes(r.id));
    const total = shown.reduce((sum, r) => sum + 2 + linesFor(r.content), 0);
    expect(total).toBeLessThanOrEqual(SHOWCASE_LINE_BUDGET);
  });

  it("tolerates junk entries and an empty list", () => {
    expect(showcaseReviews([])).toEqual([]);
    expect(showcaseReviews(undefined)).toEqual([]);
    expect(showcaseReviews([null, { id: "" }, review("ok", 10)])).toEqual(["ok"]);
  });
});
