<script>
// Dashboard engine: two of the desktop grid's four columns. The reclaimable
// headline owns the left half and the action keeps the right, so the button
// does not sit under a 48px number fighting it for the same line — the old
// narrow card had exactly that stack. See Home.vue for how meta.cols is
// consumed, and AGENTS.md §8 for why two is the ceiling.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, HardDrive, Layers, Trash2 } from "@lucide/vue";
import { formatBytes } from "../utils/metrics.js";
import { useApiUrl } from "../composables/useApiUrl";
import { useNotification } from "../composables/useNotification";
import { useReclaimable } from "../composables/useDashboardData";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";
import { storageTab } from "../utils/routes.js";

const { t } = useI18n();
const { apiUrl } = useApiUrl();
const toast = useNotification();
const router = useRouter();

// The dashboard's storage resource, shared with SystemToolsCard: one pair of
// requests, one ticker. After a delete we refresh it rather than refetching
// here, so both cards agree on what is left.
const { unusedImages, unusedVolumes, reclaimableLoaded, reclaimableLoadFailed, refreshReclaimable } =
  useReclaimable({ live: true });

const deleting = ref(false);

function imageName(img) {
  const tag = img?.tags?.[0];
  if (tag && tag !== "<none>:<none>") return tag;
  return img?.shortId || img?.id || "unknown";
}

// Images and volumes are one queue: the user is reclaiming space, not
// curating a resource type, and the biggest thing wins regardless of which
// kind it is. A volume's id is its name — that is what DELETE takes.
const queue = computed(() => {
  const images = (unusedImages.value ?? []).map((img) => ({
    kind: "image",
    id: img.id,
    name: imageName(img),
    sizeBytes: Number(img.sizeBytes) || 0,
  }));
  const volumes = (unusedVolumes.value ?? []).map((vol) => ({
    kind: "volume",
    id: vol.name,
    name: vol.name,
    sizeBytes: Number(vol.sizeBytes) || 0,
  }));
  return [...images, ...volumes].sort((a, b) => b.sizeBytes - a.sizeBytes);
});

const current = computed(() => queue.value[0] ?? null);
const remaining = computed(() => queue.value.length);
// Two is what fits under the header without pushing the headline off its
// baseline; the count in the header says how many are behind them.
const nextUp = computed(() => queue.value.slice(1, 3));

function goToManage(item = current.value) {
  if (!item) return;
  router.push(storageTab(item.kind === "image" ? "images-unused" : "volumes-unused"));
}

async function freeCurrent() {
  const item = current.value;
  if (!item || deleting.value) return;

  const confirmKey =
    item.kind === "image"
      ? "quickMetrics.biggestUnused.confirmImage"
      : "quickMetrics.biggestUnused.confirmVolume";
  if (!confirm(t(confirmKey, { name: item.name }))) return;

  deleting.value = true;
  const freedSize = formatBytes(item.sizeBytes);

  try {
    // Match the Storage page: image ids are "sha256:…" and encoding the colon
    // breaks chi's {id} param, so the path is built verbatim.
    const url =
      item.kind === "image"
        ? `${apiUrl.value}/api/images/${item.id}`
        : `${apiUrl.value}/api/volumes/${item.id}`;
    const response = await fetch(url, { method: "DELETE" });
    const data = await response.json().catch(() => ({}));

    if (!response.ok || !data.success) {
      toast.error(
        t("quickMetrics.biggestUnused.deleteFailed", {
          message: data.message || data.error || `HTTP ${response.status}`,
        })
      );
      return;
    }

    toast.success(t("quickMetrics.biggestUnused.deleted", { size: freedSize }));
    await refreshReclaimable();
  } catch (error) {
    toast.error(
      t("quickMetrics.biggestUnused.deleteFailed", {
        message: error.message || "network error",
      })
    );
  } finally {
    deleting.value = false;
  }
}
</script>

<template>
  <WidgetSkeleton v-if="!reclaimableLoaded && !reclaimableLoadFailed" />

  <!-- Nothing to reclaim is not news: the card disappears rather than
       occupying half a row to say "all clean". A failed load is treated the
       same way, so a broken endpoint cannot plant a scare on the dashboard. -->
  <div
    v-else-if="current"
    class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- Header: identity only; the number below is the card's argument. -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-amber-200 bg-amber-50 transition-transform duration-200 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20">
          <HardDrive class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.biggestUnused.title") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <div class="h-1.5 w-1.5 rounded-full bg-amber-500"></div>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-amber-600 dark:text-amber-500">
              {{ t("quickMetrics.biggestUnused.subtitle") }}
            </span>
          </div>
        </div>
      </div>

      <span class="shrink-0 rounded-full bg-amber-50 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-amber-700 dark:bg-amber-900/20 dark:text-amber-400">
        {{ t("quickMetrics.biggestUnused.remaining") }} {{ remaining }}
      </span>
    </div>

    <!-- What is behind the headline. Without this the card is a headline at
         the bottom and 190px of nothing above it; the list says the queue is
         real, and each row is a shortcut to that kind's unused tab. -->
    <div v-if="nextUp.length" class="mt-6 min-w-0">
      <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
        {{ t("quickMetrics.biggestUnused.nextUp") }}
      </div>
      <ul class="flex flex-col gap-1">
        <li v-for="item in nextUp" :key="`${item.kind}-${item.id}`">
          <button
            type="button"
            class="flex w-full cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50"
            :title="item.name"
            @click.stop="goToManage(item)"
          >
            <component :is="item.kind === 'image' ? Layers : HardDrive" class="h-3.5 w-3.5 shrink-0 text-zinc-400" />
            <span class="min-w-0 flex-1 truncate text-xs font-medium yantra-text-secondary">{{ item.name }}</span>
            <span class="shrink-0 tabular-nums text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
              {{ formatBytes(item.sizeBytes) }}
            </span>
          </button>
        </li>
      </ul>
    </div>

    <!-- Two zones: the reclaimable headline on the left, the action on the
         right, so neither queues behind the other's height. -->
    <div class="mt-auto flex flex-col gap-6 pt-6 sm:flex-row sm:items-end sm:gap-6">
      <div class="min-w-0 flex-1">
        <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
          {{ t("quickMetrics.biggestUnused.reclaimable") }}
        </div>

        <div class="mb-2 tabular-nums text-4xl font-black tracking-tighter yantra-text-primary transition-transform duration-200 group-hover:-translate-y-[3px] sm:text-5xl yantra-text-primary">
          {{ formatBytes(current.sizeBytes) }}
        </div>

        <!-- The name is the way to the full list: the card frees one thing,
             the unused tab is where the rest live. -->
        <button
          type="button"
          class="flex min-w-0 max-w-full cursor-pointer items-center gap-1.5 text-left text-xs font-semibold yantra-text-secondary transition-colors hover:text-black dark:hover:text-white"
          :title="current.name"
          @click.stop="goToManage()"
        >
          <component :is="current.kind === 'image' ? Layers : HardDrive" class="h-3.5 w-3.5 shrink-0" />
          <span class="truncate">{{ current.name }}</span>
          <span class="shrink-0 text-zinc-400">·</span>
          <span class="shrink-0">
            {{ current.kind === "image" ? t("quickMetrics.biggestUnused.image") : t("quickMetrics.biggestUnused.volume") }}
          </span>
        </button>
      </div>

      <div class="flex shrink-0 flex-col gap-2 sm:w-48">
        <button
          type="button"
          :disabled="deleting"
          @click.stop="freeCurrent"
          class="flex h-11 w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-rose-500 px-4 text-[11px] font-bold uppercase tracking-widest text-white transition-[background-color,transform] duration-200 hover:bg-rose-600 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60"
        >
          <Trash2 class="h-4 w-4 shrink-0" :class="{ 'animate-pulse': deleting }" />
          {{
            deleting
              ? t("quickMetrics.biggestUnused.deleting")
              : t("quickMetrics.biggestUnused.freeSize", { size: formatBytes(current.sizeBytes) })
          }}
        </button>

        <button
          type="button"
          class="group/link flex h-9 w-full cursor-pointer items-center justify-between gap-2 rounded-xl px-3 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary transition-colors hover:bg-zinc-50 hover:text-black dark:hover:bg-zinc-900/50 dark:hover:text-white"
          @click.stop="goToManage()"
        >
          {{ t("quickMetrics.biggestUnused.viewAll") }}
          <ArrowUpRight class="h-3.5 w-3.5 shrink-0 transition-transform duration-200 group-hover/link:translate-x-0.5" />
        </button>
      </div>
    </div>
  </div>
</template>
