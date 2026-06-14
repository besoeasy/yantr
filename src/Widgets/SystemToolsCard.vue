<script setup>
import { ref, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ArrowRight, ClipboardList, HardDrive, Layers } from "lucide-vue-next";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";

const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();

const imageCount = ref(null);
const volumeCount = ref(null);
let interval = null;

async function fetchCounts() {
  try {
    const [imgRes, volRes] = await Promise.all([
      fetch(`${apiUrl.value}/api/images`),
      fetch(`${apiUrl.value}/api/volumes`),
    ]);
    const [imgData, volData] = await Promise.all([
      expectApiSuccess(imgRes, "images"),
      expectApiSuccess(volRes, "volumes"),
    ]);
    imageCount.value = Array.isArray(imgData.images) ? imgData.images.length : null;
    volumeCount.value = Array.isArray(volData.volumes) ? volData.volumes.length : null;
  } catch {
    // counts stay null — rows still navigate fine
  }
}

onMounted(() => {
  fetchCounts();
  interval = setInterval(fetchCounts, 30000);
});

onUnmounted(() => {
  if (interval) clearInterval(interval);
});
</script>

<template>
  <div class="flex h-full flex-col rounded-2xl p-5 smooth-shadow bg-white dark:bg-[#0A0A0A] text-(--text-primary)">

    <!-- Header -->
    <div class="mb-5">
      <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-(--text-secondary)">{{ t("home.toolsNavCard.systemTools") }}</p>
      <h3 class="mt-1 text-base font-semibold tracking-tight">{{ t("home.toolsNavCard.images") }}, {{ t("home.toolsNavCard.volumes") }} & {{ t("home.toolsNavCard.logs") }}</h3>
    </div>

    <!-- Tool rows -->
    <div class="flex flex-col gap-2">

      <!-- Images -->
      <button
        type="button"
        @click="router.push('/images')"
        class="group flex items-center justify-between rounded-xl px-3.5 py-3 bg-(--surface-muted) hover:bg-blue-500/8 transition-all duration-200 text-left"
      >
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-500/10">
            <Layers class="h-4 w-4 text-blue-500" />
          </div>
          <div class="min-w-0">
            <div class="text-xs font-semibold">{{ t("home.toolsNavCard.images") }}</div>
            <div class="mt-0.5 text-[11px] text-(--text-secondary) truncate">{{ t("home.toolsNavCard.imagesDesc") }}</div>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0 ml-3">
          <span v-if="imageCount !== null" class="text-sm font-bold text-blue-500 tabular-nums">{{ imageCount }}</span>
          <span v-else class="w-5 h-3 rounded bg-(--surface-muted) animate-pulse"></span>
          <ArrowRight class="h-3.5 w-3.5 text-(--text-secondary) -translate-x-1 opacity-0 group-hover:translate-x-0 group-hover:opacity-100 transition-all duration-200" />
        </div>
      </button>

      <!-- Volumes -->
      <button
        type="button"
        @click="router.push('/volumes')"
        class="group flex items-center justify-between rounded-xl px-3.5 py-3 bg-(--surface-muted) hover:bg-emerald-500/8 transition-all duration-200 text-left"
      >
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10">
            <HardDrive class="h-4 w-4 text-emerald-500" />
          </div>
          <div class="min-w-0">
            <div class="text-xs font-semibold">{{ t("home.toolsNavCard.volumes") }}</div>
            <div class="mt-0.5 text-[11px] text-(--text-secondary) truncate">{{ t("home.toolsNavCard.volumesDesc") }}</div>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0 ml-3">
          <span v-if="volumeCount !== null" class="text-sm font-bold text-emerald-500 tabular-nums">{{ volumeCount }}</span>
          <span v-else class="w-5 h-3 rounded bg-(--surface-muted) animate-pulse"></span>
          <ArrowRight class="h-3.5 w-3.5 text-(--text-secondary) -translate-x-1 opacity-0 group-hover:translate-x-0 group-hover:opacity-100 transition-all duration-200" />
        </div>
      </button>

      <!-- Logs -->
      <button
        type="button"
        @click="router.push('/logs')"
        class="group flex items-center justify-between rounded-xl px-3.5 py-3 bg-(--surface-muted) hover:bg-violet-500/8 transition-all duration-200 text-left"
      >
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-violet-500/10">
            <ClipboardList class="h-4 w-4 text-violet-500" />
          </div>
          <div class="min-w-0">
            <div class="text-xs font-semibold">{{ t("home.toolsNavCard.logs") }}</div>
            <div class="mt-0.5 text-[11px] text-(--text-secondary) truncate">{{ t("home.toolsNavCard.logsDesc") }}</div>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0 ml-3">
          <span class="inline-flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider text-violet-500">
            <span class="h-1.5 w-1.5 rounded-full bg-violet-500"></span>
            Live
          </span>
          <ArrowRight class="h-3.5 w-3.5 text-(--text-secondary) -translate-x-1 opacity-0 group-hover:translate-x-0 group-hover:opacity-100 transition-all duration-200" />
        </div>
      </button>

    </div>



  </div>
</template>
