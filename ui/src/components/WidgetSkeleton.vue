<script>
// The dashboard's loading state.
//
// Twelve of sixteen widgets used to paint nothing and fill in when the data
// arrived, which moved the grid under the reader's eyes. This is one skeleton
// for all of them, because a placeholder that differs per card is just a
// different set of jumps: same chrome, same rhythm, neutral bars standing in
// for the values.
//
// It lives in components/ rather than Widgets/ on purpose — Home.vue globs
// that directory for cards, and a skeleton would otherwise show up on the
// dashboard as a seventeenth widget.
</script>

<script setup>
// No props: a skeleton that mirrors each card's own layout would need sixteen
// variants and would drift the moment a card changed. This is deliberately the
// generic one — header block, a value line, a table of bars — sized to the
// shared minimum so the card it stands in for does not resize when data lands.
</script>

<template>
  <div
    class="yantra-card yantra-card-min flex w-full flex-col overflow-hidden rounded-2xl p-7"
    aria-hidden="true"
  >
    <div class="flex items-center gap-4">
      <span class="h-12 w-12 shrink-0 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800"></span>
      <span class="flex flex-1 flex-col gap-2">
        <span class="block h-3 w-24 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
        <span class="block h-2.5 w-16 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
      </span>
    </div>

    <div class="mt-auto flex flex-col gap-3 pt-6">
      <span class="block h-8 w-48 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800"></span>
      <div class="flex flex-col gap-2">
        <span class="block h-2.5 w-full animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
        <span class="block h-2.5 w-48 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
        <span class="block h-2.5 w-40 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
      </div>
    </div>
  </div>
</template>