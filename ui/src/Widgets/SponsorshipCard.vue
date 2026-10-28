<script>
// Dashboard engine: two columns. GitHub Sponsors and the Bitcoin donation
// address live on one card now, so the two ways to support Yantr stay next to
// each other instead of being shuffled apart in the daily grid.
export const meta = { cols: 2 };
</script>
<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Heart, Bitcoin, Copy, Wallet, AlertCircle } from "@lucide/vue";
import NostrQrCode from "../components/NostrQrCode.vue";

const { t } = useI18n();

// --- Bitcoin address fetched from DNS TXT (btc.besoeasy.com) ---
const btcAddress = ref("");
const btcLoading = ref(true);
const btcError = ref(null);
const copied = ref(false);

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

onMounted(fetchBtc);

const shortBtc = computed(() => {
  const a = btcAddress.value;
  if (!a) return "";
  if (a.length <= 22) return a;
  return `${a.slice(0, 10)}…${a.slice(-8)}`;
});

async function copyAddress() {
  if (!btcAddress.value) return;
  try {
    await navigator.clipboard?.writeText(btcAddress.value);
    copied.value = true;
    setTimeout(() => { copied.value = false; }, 2000);
  } catch {
    // clipboard unavailable — the full address stays readable via title
  }
}
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)] sm:border sm:border-zinc-200 dark:sm:border-zinc-800">
    <!-- GitHub Sponsors -->
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-rose-200 bg-rose-50 transition-transform duration-200 group-hover:scale-105 dark:border-rose-900/50 dark:bg-rose-900/20">
          <Heart class="h-5 w-5 fill-current text-rose-500" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("sponsorCard.label") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <span class="text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
              {{ t("home.externalLinks.resources") }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <div class="relative z-10 mt-auto flex flex-col pt-5">
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

    <!-- Divider -->
    <div class="mt-6 mb-6 border-t border-zinc-100 dark:border-zinc-900"></div>

    <!-- Bitcoin donation -->
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-amber-200 bg-amber-50 transition-transform duration-200 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20">
          <Bitcoin class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("sponsorCard.btcBadge") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <span class="text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
              {{ t("home.externalLinks.resources") }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-5 flex flex-1 items-center justify-center">
      <div v-if="btcLoading" class="h-32 w-32 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800"></div>
      <div v-else-if="btcError" class="flex h-32 w-32 items-center justify-center rounded-xl border-red-200 bg-red-50 p-3 dark:border-red-900/50 dark:bg-red-900/20">
        <AlertCircle class="h-5 w-5 text-red-500" />
      </div>
      <div v-else class="rounded-xl bg-white p-2 dark:bg-white">
        <NostrQrCode :value="`bitcoin:${btcAddress}`" :size="116" :label="btcAddress" />
      </div>
    </div>

    <div class="mt-5">
      <div v-if="btcLoading" class="h-3 w-40 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></div>
      <div v-else-if="btcError" class="text-[11px] text-red-500">{{ btcError }}</div>
      <code v-else class="block font-mono text-[10px] yantra-text-secondary" :title="btcAddress">{{ shortBtc }}</code>
    </div>

    <div class="mt-4 grid grid-cols-2 gap-2">
      <button
        type="button"
        class="flex h-10 items-center justify-center gap-2 rounded-xl border border-zinc-200 text-[10px] font-bold uppercase tracking-widest yantra-text-primary transition-colors dark:border-zinc-800 dark:bg-zinc-900/50 active:scale-[0.98]"
        :disabled="btcLoading || !!btcError"
        @click="copyAddress"
      >
        <Copy class="h-3.5 w-3.5" v-if="!copied" />
        <Bitcoin v-else-if="copied" class="h-3.5 w-3.5 text-emerald-500" />
        {{ copied ? t("sponsorCard.btcCopied") : t("sponsorCard.btcCopy") }}
      </button>
      <a
        :href="btcAddress ? `bitcoin:${btcAddress}` : '#'"
        class="flex h-10 items-center justify-center gap-2 rounded-xl bg-amber-500 text-[10px] font-bold uppercase tracking-widest text-zinc-900 active:scale-[0.98]"
      >
        <Wallet class="h-3.5 w-3.5" />
        {{ t("sponsorCard.btcWallet") }}
      </a>
    </div>
  </div>
</template>