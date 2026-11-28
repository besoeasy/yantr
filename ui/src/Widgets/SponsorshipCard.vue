<script>
// Two columns. One donate card, two clearly separated ways to help: Bitcoin
// (self-custodial, address verified from the project's own DNS) and GitHub
// Sponsors. Single accent, one primary action, honest loading / error / copied
// states. No QR plate at this size and no Open-Wallet deep link — copying the
// full address is the lowest-friction, most verifiable path.
export const meta = { cols: 2 };
</script>
<script setup>
import { ref, onMounted, onUnmounted } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Heart, Bitcoin, Copy, Check, AlertCircle } from "@lucide/vue";
import { copyText } from "../utils/clipboard";
import { fetchBitcoinAddress } from "../utils/dnsBitcoin";

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
    btcAddress.value = await fetchBitcoinAddress();
  } catch (e) {
    btcError.value = e?.message || String(e);
    btcAddress.value = "";
  } finally {
    btcLoading.value = false;
  }
}

onMounted(fetchBtc);

async function copyAddress() {
  if (!btcAddress.value) return;
  // copyText falls back to the textarea path on a plain-HTTP page, where
  // navigator.clipboard is undefined. Always copies the FULL address.
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
      <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-400">
        <Heart class="h-5 w-5 fill-current" />
      </div>
    </div>

    <!-- Stacked panels: a widget is never guaranteed full width, so don't
         split inside a card that may itself be narrow. -->
    <div class="mt-6 grid flex-1 grid-cols-1 gap-4">
      <!-- Bitcoin -->
      <section class="flex min-w-0 flex-col rounded-2xl border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center gap-2">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-900/50 dark:bg-amber-900/20">
            <Bitcoin class="h-4 w-4 text-amber-600 dark:text-amber-400" />
          </div>
          <div class="min-w-0">
            <p class="text-[11px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("sponsorCard.bitcoinLabel") }}</p>
            <p class="mt-0.5 text-[11px] yantra-text-secondary">{{ t("sponsorCard.bitcoinHint") }}</p>
          </div>
        </div>

        <!-- Full address, always. Never truncated; wraps instead. -->
        <div class="mt-4 min-w-0 rounded-xl border border-zinc-200 bg-white px-3.5 py-2.5 dark:border-zinc-800 dark:bg-zinc-900" aria-live="polite">
          <div v-if="btcLoading" class="h-3 w-40 max-w-full animate-pulse rounded-full bg-zinc-200 dark:bg-zinc-700"></div>
          <div v-else-if="btcError" role="alert" class="flex flex-wrap items-center gap-2 text-[11px] text-red-500">
            <AlertCircle class="h-4 w-4 shrink-0" />
            <span :title="btcError">{{ t("sponsorCard.btcError") }}</span>
            <button
              type="button"
              class="inline-flex h-7 items-center justify-center rounded-lg border border-zinc-200 bg-white px-2.5 text-[10px] font-bold uppercase tracking-widest yantra-text-primary transition-colors hover:border-zinc-300 active:scale-[0.98] dark:border-zinc-800 dark:bg-zinc-900 dark:hover:border-zinc-700"
              @click="fetchBtc"
            >
              {{ t("sponsorCard.retry") }}
            </button>
          </div>
          <code v-else class="block w-full break-all font-mono text-[11px] leading-relaxed yantra-text-primary">{{ btcAddress }}</code>
        </div>

        <button
          type="button"
          class="mt-4 flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-amber-500 text-[10px] font-bold uppercase tracking-widest text-zinc-900 transition-opacity hover:opacity-90 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="btcLoading || !!btcError"
          @click="copyAddress"
        >
          <Check v-if="copied" class="h-3.5 w-3.5 text-emerald-500" />
          <Copy v-else class="h-3.5 w-3.5" />
          <span aria-live="polite">{{ copied ? t("sponsorCard.btcCopied") : t("sponsorCard.btcCopy") }}</span>
        </button>
      </section>

      <!-- GitHub Sponsors -->
      <section class="flex min-w-0 flex-col rounded-2xl border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center gap-2">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
            <Heart class="h-4 w-4 yantra-text-secondary" />
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
          class="group/cta mt-4 flex h-10 w-full items-center justify-between gap-3 rounded-xl border border-zinc-200 bg-white px-4 text-[10px] font-bold uppercase tracking-widest yantra-text-primary transition-colors hover:border-zinc-300 active:scale-[0.98] dark:border-zinc-800 dark:bg-zinc-900 dark:hover:border-zinc-700"
        >
          <div class="flex min-w-0 items-center gap-2.5">
            <Heart class="h-4 w-4 shrink-0 fill-current text-amber-500" />
            <span class="truncate">{{ t("sponsorCard.cta") }}</span>
          </div>
          <ArrowUpRight class="h-4 w-4 shrink-0 opacity-70 transition-transform duration-200 group-hover/cta:opacity-100" />
        </a>
      </section>
    </div>
  </div>
</template>
