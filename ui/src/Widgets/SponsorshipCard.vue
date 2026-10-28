<script>
// Dashboard engine: one of the desktop grid's four columns. The card flips
// between the GitHub-sponsor ask and a Bitcoin donation address every 100s,
// so a single tile carries both ways to support the project.
export const meta = { cols: 1 };
</script>
<script setup>
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Heart, Bitcoin } from "@lucide/vue";

const { t } = useI18n();

// --- Bitcoin address fetched from DNS TXT (btc.besoeasy.com) ---
const btcAddress = ref("");
const btcLoading = ref(true);
const btcError = ref(null);

async function fetchBtc() {
  try {
    btcLoading.value = true;
    btcError.value = null;
    const res = await fetch(
      "https://cloudflare-dns.com/dns-query?name=btc.besoeasy.com&type=TXT",
      { headers: { Accept: "application/dns-json" } }
    );
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();
    const answers = Array.isArray(data?.Answer) ? data.Answer : [];
    // TXT data arrives quoted: "\"bc1...\"". Strip surrounding quotes and
    // join any split chunks.
    const raw = answers
      .map((a) => (typeof a?.data === "string" ? a.data : ""))
      .join("")
      .replace(/^"|"$/g, "")
      .replace(/"\s*"/g, "")
      .trim();
    if (!raw) throw new Error("No TXT record returned");
    btcAddress.value = raw;
  } catch (e) {
    btcError.value = e?.message || String(e);
    btcAddress.value = "";
  } finally {
    btcLoading.value = false;
  }
}

// --- 100s flip between the two faces ---
const showBtc = ref(false);
let flipTimer = null;

onMounted(() => {
  fetchBtc();
  flipTimer = setInterval(() => {
    showBtc.value = !showBtc.value;
  }, 100_000);
});

onUnmounted(() => {
  if (flipTimer) clearInterval(flipTimer);
});

const shortBtc = computed(() => {
  const a = btcAddress.value;
  if (!a) return "";
  if (a.length <= 22) return a;
  return `${a.slice(0, 10)}…${a.slice(-8)}`;
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)] sm:border sm:border-zinc-200 dark:sm:border-zinc-800">
    <!-- Header -->
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <div
          class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl transition-transform duration-200 group-hover:scale-105"
          :class="showBtc
            ? 'border-amber-200 bg-amber-50 dark:border-amber-900/50 dark:bg-amber-900/20'
            : 'border-rose-200 bg-rose-50 dark:border-rose-900/50 dark:bg-rose-900/20'"
        >
          <Bitcoin v-if="showBtc" class="h-5 w-5 text-amber-600 dark:text-amber-400" />
          <Heart v-else class="h-5 w-5 fill-current text-rose-500" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ showBtc ? t("sponsorCard.btcTitle") : t("sponsorCard.label") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <span class="text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
              {{ t("home.externalLinks.resources") }}
            </span>
          </div>
        </div>
      </div>
      <!-- Flip indicator -->
      <span class="text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
        {{ showBtc ? "BTC" : "GH" }}
      </span>
    </div>

    <!-- Face A: GitHub sponsors ask -->
    <div v-if="!showBtc" class="relative z-10 mt-auto flex flex-col pt-5">
      <div class="mb-1.5 text-2xl font-black leading-[1.2] tracking-tight yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary">
        {{ t("sponsorCard.title") }}
      </div>
      <div class="line-clamp-3 text-xs leading-relaxed yantra-text-secondary">
        {{ t("sponsorCard.description") }}
      </div>

      <a
        href="https://github.com/sponsors/besoeasy"
        target="_blank"
        rel="noopener noreferrer"
        class="group/cta relative mt-5 flex h-11 w-full items-center justify-between gap-3 overflow-hidden rounded-xl bg-rose-500 px-5 yantra-text-primary transition-[border-color,box-shadow,transform] duration-200 hover:bg-rose-600 active:scale-[0.98]"
      >
        <div class="relative z-10 flex items-center gap-2.5">
          <Heart class="h-4 w-4 shrink-0 fill-current" />
          <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('sponsorCard.cta') }}</span>
        </div>
        <ArrowUpRight class="relative z-10 h-4.5 w-4.5 shrink-0 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
      </a>
    </div>

    <!-- Face B: Bitcoin donation address -->
    <div v-else class="relative z-10 mt-auto flex flex-col pt-5">
      <div class="mb-1.5 text-2xl font-black leading-[1.2] tracking-tight yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary">
        {{ t("sponsorCard.btcHeading") }}
      </div>
      <div class="line-clamp-3 text-xs leading-relaxed yantra-text-secondary">
        {{ t("sponsorCard.btcDescription") }}
      </div>

      <div class="mt-4 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
        <div v-if="btcLoading" class="text-[11px] yantra-text-secondary">{{ t("sponsorCard.btcLoading") }}</div>
        <div v-else-if="btcError" class="text-[11px] text-red-500">{{ btcError }}</div>
        <div v-else class="flex flex-col gap-1">
          <code class="break-all font-mono text-[11px] yantra-text-primary" :title="btcAddress">{{ shortBtc }}</code>
          <button
            type="button"
            class="mt-2 text-[10px] font-bold uppercase tracking-widest text-amber-600 dark:text-amber-400"
            @click="navigator.clipboard?.writeText(btcAddress)"
          >
            {{ t("sponsorCard.btcCopy") }}
          </button>
        </div>
      </div>

      <a
        :href="btcAddress ? `bitcoin:${btcAddress}` : '#'"
        class="group/cta relative mt-4 flex h-11 w-full items-center justify-between gap-3 overflow-hidden rounded-xl bg-amber-500 px-5 text-zinc-900 transition-[border-color,box-shadow,transform] duration-200 active:scale-[0.98]"
      >
        <div class="relative z-10 flex items-center gap-2.5">
          <Bitcoin class="h-4 w-4 shrink-0" />
          <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("sponsorCard.btcCta") }}</span>
        </div>
        <ArrowUpRight class="relative z-10 h-4.5 w-4.5 shrink-0 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
      </a>
    </div>
  </div>
</template>