// The line-budget that decides how many reviews the Latest Reviews card shows.
//
// One band, a variable number of quotes: the card shows as many as fit, and
// "fit" is measured, not guessed — each review costs its estimated rendered
// line count plus the header it carries, against a budget for the whole card.
// A long review eats the budget and shows nearly alone; one-liners pack in by
// the handful. Newest first always: recency decides the order, length decides
// only the count.

// Approximate rendered characters per quote line at text-sm in a half-row
// band column. Conservative on the low side: underestimating lines packs the
// card too tight, overestimating shows fewer reviews than would fit — the
// latter is the safer miss.
const CHARS_PER_LINE = 46;

// Every review carries an app-chip row and an author line.
const FOOTER_LINES_PER_REVIEW = 2;

// The whole quote region of the band, expressed in quote lines. Calibrated
// against the row the card shares with the uptime/community widgets: roughly
// fourteen text-sm lines after the card's own header and padding.
export const SHOWCASE_LINE_BUDGET = 14;

// Four is the cap for the same reason the developer band caps its app chips:
// past it a showcase becomes a list.
export const SHOWCASE_MAX_REVIEWS = 4;

// Per-review line clamp at display time; also the estimate's ceiling, so what
// is measured cannot drift from what is rendered.
export const SHOWCASE_LINE_CLAMP = 3;

// linesFor estimates the review's rendered line count, clamped to the clamp
// that its blockquote will actually get — the estimate and the display must
// agree or the budget lies.
export function linesFor(text, charsPerLine = CHARS_PER_LINE) {
  const length = String(text ?? "").length;
  return Math.min(SHOWCASE_LINE_CLAMP, Math.max(1, Math.ceil(length / charsPerLine)));
}

// showcaseReviews picks, from reviews ordered newest first, the ids that fit
// the budget: newest complete, then whatever else fits, in order, until the
// budget or the cap runs out. The first review always shows — a band that can
// hold one review must show it, whatever its length; only the "one more?"
// questions are answered by the budget.
export function showcaseReviews(
  reviews,
  budget = SHOWCASE_LINE_BUDGET,
  maxReviews = SHOWCASE_MAX_REVIEWS
) {
  const ids = [];
  let used = 0;

  for (const review of Array.isArray(reviews) ? reviews : []) {
    if (!review || !review.id) continue;

    const cost = FOOTER_LINES_PER_REVIEW + linesFor(review.content);
    if (ids.length === 0) {
      ids.push(review.id);
      used = Math.min(cost, budget);
      continue;
    }
    if (ids.length >= maxReviews || used + cost > budget) break;

    ids.push(review.id);
    used += cost;
  }

  return ids;
}
