<script>
// Dashboard engine: two columns. The card is split into a full-space live QR
// plate on the left half and identity, address, wallet actions and GitHub
// sponsorship on the right half. Width honours the two-column ceiling every
// widget lives by (AGENTS.md §8); see Home.vue.
export const meta = { cols: 2 };
</script>
<script setup>
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Heart, Bitcoin, Copy, Check, Wallet, AlertCircle } from "@lucide/vue";
import NostrQrCode from "../components/NostrQrCode.vue";

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
  if (a.length <= 22) return a;
  return `${a.slice(0, 10)}…${a.slice(-8)}`;
});

async function copyAddress() {
  if (!btcAddress.value) return;
  try {
    await navigator.clipboard?.writeText(btcAddress.value);
    copied.value = true;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => { copied.value = false; }, 2000);
  } catch {
    // clipboard unavailable — the full address stays readable via title
  }
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

    <div class="grid h-full w-full flex-1 grid-cols-1 md:grid-cols-2 gap-6 items-center">
      <!-- Left Column: Full-space QR plate -->
      <div class="flex h-full w-full items-center justify-center">
        <div
          v-if="btcLoading"
          class="aspect-square w-full max-w-64 animate-pulse rounded-2xl bg-zinc-100 dark:bg-zinc-800"
        ></div>

        <div
          v-else-if="btcError"
          class="flex aspect-square w-full max-w-64 flex-col items-center justify-center gap-2 rounded-2xl border border-red-200 bg-red-50 p-4 text-center dark:border-red-900/50 dark:bg-red-900/20"
        >
          <AlertCircle class="h-6 w-6 text-red-500" />
          <span class="text-xs text-red-500">{{ btcError }}</span>
        </div>

        <div
          v-else
          class="flex aspect-square w-full max-w-64 items-center justify-center rounded-2xl border border-zinc-200 bg-white p-3.5 shadow-sm dark:border-zinc-800 dark:bg-white"
        >
          <NostrQrCode
            :value="`bitcoin:${btcAddress}`"
            :size="240"
            :label="btcAddress"
            class="h-full w-full"
          />
        </div>
      </div>

      <!-- Right Column: Details & Actions -->
      <div class="flex h-full w-full flex-col justify-between text-left">
        <!-- Header -->
        <div class="flex items-center gap-3">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-amber-200 bg-amber-50 transition-transform duration-200 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20">
            <Bitcoin class="h-5 w-5 text-amber-600 dark:text-amber-400" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary">
              Donate
            </h3>
            <p class="mt-1 text-[11px] font-medium yantra-text-secondary">
              Support Yantr development
            </p>
          </div>
        </div>

        <!-- Address & Quick Copy/Wallet -->
        <div class="mt-3 flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2 rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-2 dark:border-zinc-800 dark:bg-zinc-900/50">
            <div v-if="btcLoading" class="h-3 w-32 animate-pulse rounded-full bg-zinc-200 dark:bg-zinc-700"></div>
            <div v-else-if="btcError" class="text-[11px] text-red-500">{{ btcError }}</div>
            <code v-else class="truncate font-mono text-[11px] yantra-text-primary" :title="btcAddress">{{ shortBtc }}</code>
          </div>

          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              class="flex h-10 items-center justify-center gap-2 rounded-xl border border-zinc-200 text-[10px] font-bold uppercase tracking-widest yantra-text-primary transition-colors dark:border-zinc-800 dark:bg-zinc-900/50 active:scale-[0.98]"
              :disabled="btcLoading || !!btcError"
              @click="copyAddress"
            >
              <Check class="h-3.5 w-3.5 text-emerald-500" v-if="copied" />
              <Copy class="h-3.5 w-3.5" v-else />
              <span>{{ copied ? t("sponsorCard.btcCopied") : t("sponsorCard.btcCopy") }}</span>
            </button>
            <a
              :href="btcAddress ? `bitcoin:${btcAddress}` : '#'"
              class="flex h-10 items-center justify-center gap-2 rounded-xl bg-amber-500 text-[10px] font-bold uppercase tracking-widest text-zinc-900 active:scale-[0.98]"
            >
              <Wallet class="h-3.5 w-3.5" />
              <span>{{ t("sponsorCard.btcWallet") }}</span>
            </a>
          </div>
        </div>

        <!-- or divider -->
        <div class="my-4 flex items-center gap-3">
          <div class="h-px flex-1 bg-zinc-200 dark:bg-zinc-800"></div>
          <span class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">or</span>
          <div class="h-px flex-1 bg-zinc-200 dark:bg-zinc-800"></div>
        </div>

        <!-- Sponsor on GitHub -->
        <a
          href="https://github.com/sponsors/besoeasy"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta relative flex h-11 w-full items-center justify-between gap-3 overflow-hidden rounded-xl bg-rose-500 px-5 yantra-text-primary transition-[border-color,box-shadow,transform] duration-200 hover:bg-rose-600 active:scale-[0.98]"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Heart class="h-4 w-4 shrink-0 fill-current" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('sponsorCard.cta') }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 shrink-0 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>
      </div>
    </div>
  </div>
</template>
