<script>
// Two columns, no QR plate: Bitcoin support and GitHub sponsorship sit side by
// side so the card reads as two clear ways to help, not a scan target.
export const meta = { cols: 2 };
</script>
<script setup>
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Heart, Bitcoin, Copy, Check, Wallet, AlertCircle } from "@lucide/vue";
import { copyText } from "../utils/clipboard";

const { t } = useI18n();

// --- Bitcoin address fetched from DNS TXT (btc.besoeasy.com) ---
const btcAddress = ref("");
const btcLoading = ref(true);
const btcError = ref(null);
const copied = ref(false);
let copiedTimer = null;

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
  if (a.length <= 34) return a;
  return `${a.slice(0, 18)}…${a.slice(-12)}`;
});

async function copyAddress() {
  if (!btcAddress.value) return;
  // copyText falls back to the textarea path on a plain-HTTP page, where
  // navigator.clipboard is undefined.
  if (!(await copyText(btcAddress.value))) return;
  copied.value = true;
  if (copiedTimer) clearTimeout(copiedTimer);
  copiedTimer = setTimeout(() => { copied.value = false; }, 2000);
}

onUnmounted(() => {
  if (copiedTimer) clearTimeout(copiedTimer);
});
</script>

<template>
  <div class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-amber-500 to-transparent"
    ></div>

    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary">{{ t("sponsorCard.title") }}</h3>
        <p class="mt-1.5 text-[11px] font-medium yantra-text-secondary">{{ t("sponsorCard.subtitle") }}</p>
      </div>
      <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-rose-200 bg-rose-50 text-rose-500 dark:border-rose-900/50 dark:bg-rose-900/20">
        <Heart class="h-5 w-5 fill-current" />
      </div>
    </div>

    <div class="mt-6 grid flex-1 grid-cols-1 gap-4 md:grid-cols-2">
      <section class="flex min-w-0 flex-col rounded-2xl border border-zinc-200 bg-zinc-50/70 p-4 dark:border-zinc-800 dark:bg-zinc-900/40">
        <div class="flex items-center gap-2">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-900/50 dark:bg-amber-900/20">
            <Bitcoin class="h-4 w-4 text-amber-600 dark:text-amber-400" />
          </div>
          <div class="min-w-0">
            <p class="text-[11px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("sponsorCard.bitcoinLabel") }}</p>
            <p class="mt-0.5 text-[11px] yantra-text-secondary">{{ t("sponsorCard.bitcoinHint") }}</p>
          </div>
        </div>

        <div class="mt-4 flex min-h-[3.25rem] items-center rounded-xl border border-zinc-200 bg-white px-3.5 py-2.5 dark:border-zinc-800 dark:bg-zinc-950/60">
          <div v-if="btcLoading" class="h-3 w-40 animate-pulse rounded-full bg-zinc-200 dark:bg-zinc-700"></div>
          <div v-else-if="btcError" class="flex items-center gap-2 text-[11px] text-red-500">
            <AlertCircle class="h-4 w-4 shrink-0" />
            <span class="truncate" :title="btcError">{{ btcError }}</span>
          </div>
          <code v-else class="block truncate font-mono text-[11px] yantra-text-primary" :title="btcAddress">{{ shortBtc }}</code>
        </div>

        <div class="mt-4 grid grid-cols-2 gap-2">
          <button
            type="button"
            class="flex h-10 items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-white text-[10px] font-bold uppercase tracking-widest yantra-text-primary transition-colors hover:border-zinc-300 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-800 dark:bg-zinc-950/60 dark:hover:border-zinc-700 active:scale-[0.98]"
            :disabled="btcLoading || !!btcError"
            @click="copyAddress"
          >
            <Check v-if="copied" class="h-3.5 w-3.5 text-emerald-500" />
            <Copy v-else class="h-3.5 w-3.5" />
            <span>{{ copied ? t("sponsorCard.btcCopied") : t("sponsorCard.btcCopy") }}</span>
          </button>
          <a
            :href="btcAddress ? `bitcoin:${btcAddress}` : undefined"
            :aria-disabled="btcLoading || !!btcError ? 'true' : undefined"
            :tabindex="btcLoading || !!btcError ? -1 : undefined"
            class="flex h-10 items-center justify-center gap-2 rounded-xl bg-amber-500 text-[10px] font-bold uppercase tracking-widest text-zinc-900 transition-opacity hover:bg-amber-400 active:scale-[0.98]"
            :class="btcLoading || !!btcError ? 'pointer-events-none opacity-50' : ''"
          >
            <Wallet class="h-3.5 w-3.5" />
            <span>{{ t("sponsorCard.btcWallet") }}</span>
          </a>
        </div>
      </section>

      <section class="flex min-w-0 flex-col rounded-2xl border border-zinc-200 bg-zinc-50/70 p-4 dark:border-zinc-800 dark:bg-zinc-900/40">
        <div class="flex items-center gap-2">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-rose-200 bg-rose-50 dark:border-rose-900/50 dark:bg-rose-900/20">
            <Heart class="h-4 w-4 text-rose-500" />
          </div>
          <div class="min-w-0">
            <p class="text-[11px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("sponsorCard.githubLabel") }}</p>
            <p class="mt-0.5 text-[11px] yantra-text-secondary">{{ t("sponsorCard.githubHint") }}</p>
          </div>
        </div>

        <p class="mt-4 flex-1 text-sm leading-relaxed yantra-text-secondary">
          {{ t("sponsorCard.githubBody") }}
        </p>

        <a
          href="https://github.com/sponsors/besoeasy"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta mt-4 flex h-10 w-full items-center justify-between gap-3 rounded-xl bg-rose-500 px-4 text-zinc-900 transition-colors hover:bg-rose-400 active:scale-[0.98]"
        >
          <div class="flex min-w-0 items-center gap-2.5">
            <Heart class="h-4 w-4 shrink-0 fill-current" />
            <span class="truncate text-[10px] font-bold uppercase tracking-widest">{{ t("sponsorCard.cta") }}</span>
          </div>
          <ArrowUpRight class="h-4 w-4 shrink-0 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-0.5 group-hover/cta:translate-x-0.5 group-hover/cta:opacity-100" />
        </a>
      </section>
    </div>
  </div>
</template>
