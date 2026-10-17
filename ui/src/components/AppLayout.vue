<script setup>
/**
 * AppLayout — the inside of a page.
 *
 * App.vue owns the frame: the sidebar, the mobile header, and the `md:pl-20`
 * offset on <main>. This owns what sits inside that frame and was previously
 * copied into every view: the page background, text colour, font, and one
 * container width with one padding scale.
 *
 * Before this, eight views each opened with a near-identical
 * `min-h-screen bg-white dark:bg-[#0A0A0A] … font-sans` wrapper and then picked
 * their own max-width — max-w-400 on the dashboards, max-w-7xl on the detail
 * pages, max-w-6xl on the developer profile — from three different padding
 * scales. The widths are still three deliberate choices, but they are now named
 * here instead of restated per view.
 */
const WIDTHS = {
  /** The dashboards, which use the whole column. */
  wide: "max-w-400",
  /** Detail pages, which read better at a narrower measure. */
  reading: "max-w-7xl",
  /** The developer profile, whose two-column header wants a tighter measure. */
  narrow: "max-w-6xl",
  /** No container at all. For a view that brings its own body. */
  bare: "",
};

const GAPS = {
  none: "",
  tight: "space-y-6",
  loose: "space-y-8",
};

const props = defineProps({
  width: {
    type: String,
    default: "reading",
    // Mirrors WIDTHS' keys. defineProps() is hoisted out of setup(), so it
    // cannot reference the map itself.
    validator: (value) => ["wide", "reading", "narrow", "bare"].includes(value),
  },
  /** Vertical rhythm between this page's sections. Ignored when width is bare. */
  gap: {
    type: String,
    default: "loose",
    validator: (value) => ["none", "tight", "loose"].includes(value),
  },
});
</script>

<template>
  <div
    class="min-h-screen bg-white font-sans text-gray-900 selection:bg-blue-500/30 dark:bg-[#0A0A0A] dark:text-zinc-100"
  >
    <!-- `bare` adds nothing at all — not even a wrapper div. A page that uses it
         is bringing its own body, and it may be relying on this root being the
         flex container: the catalog puts `flex lg:flex-row` on this element via
         class fallthrough, so an extra div in between would stack its filter
         rail on top of its grid instead of beside it. That is exactly what a
         first version of this component did. -->
    <slot v-if="props.width === 'bare'" />
    <div
      v-else
      class="mx-auto px-4 py-6 sm:px-6 sm:py-8 lg:px-8"
      :class="[WIDTHS[props.width], GAPS[props.gap]]"
    >
      <slot />
    </div>
  </div>
</template>
