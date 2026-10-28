<script>
// Dashboard engine: one of the desktop grid's four columns. A minimal Bitcoin
// donation tile: badge, centered QR plate, address, and a two-action row.
export const meta = { cols: 1 };
</script>
<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { Bitcoin, Copy, Wallet, AlertCircle } from "@lucide/vue";
import NostrQrCode from "../components/NostrQrCode.vue";

const { t } = useI18n();

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
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-center transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)] sm:border sm:border-zinc-200 dark:sm:border-zinc-800">
    <span class="mx-auto inline-flex items-center gap-1.5 rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary dark:border-zinc-800 dark:bg-zinc-900/50">
      <Bitcoin class="h-3 w-3" />
      {{ t("sponsorCard.btcBadge") }}
    </span>

    <!-- QR plate -->
    <div class="mt-5 flex flex-1 items-center justify-center">
      <div v-if="btcLoading" class="h-32 w-32 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800"></div>
      <div v-else-if="btcError" class="flex h-32 w-32 items-center justify-center rounded-xl border-red-200 bg-red-50 p-3 dark:border-red-900/50 dark:bg-red-900/20">
        <AlertCircle class="h-5 w-5 text-red-500" />
      </div>
      <div v-else class="rounded-xl bg-white p-2 dark:bg-white">
        <NostrQrCode :value="`bitcoin:${btcAddress}`" :size="116" :label="btcAddress" />
      </div>
    </div>

    <!-- Address -->
    <div class="mt-5">
      <div v-if="btcLoading" class="h-3 w-40 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></div>
      <div v-else-if="btcError" class="text-[11px] text-red-500">{{ btcError }}</div>
      <code v-else class="block font-mono text-[10px] yantra-text-secondary" :title="btcAddress">{{ shortBtc }}</code>
    </div>

    <!-- Actions -->
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