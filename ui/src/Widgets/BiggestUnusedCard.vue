<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { HardDrive, Layers, Trash2 } from '@lucide/vue'
import { formatBytes } from '../utils/metrics.js'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useNotification } from '../composables/useNotification'

const { t } = useI18n()
const { apiUrl } = useApiUrl()
const toast = useNotification()
const router = useRouter()

const unusedImages = ref([])
const unusedVolumes = ref([])
const deleting = ref(false)
const flipKey = ref(0)
let refreshInterval = null

function imageName(img) {
  const tag = img?.tags?.[0]
  if (tag && tag !== '<none>:<none>') return tag
  return img?.shortId || img?.id || 'unknown'
}

function buildQueue() {
  const images = (unusedImages.value || []).map((img) => ({
    kind: 'image',
    id: img.id,
    name: imageName(img),
    sizeBytes: Number(img.sizeBytes) || 0,
  }))
  const volumes = (unusedVolumes.value || []).map((vol) => ({
    kind: 'volume',
    id: vol.name,
    name: vol.name,
    sizeBytes: Number(vol.sizeBytes) || 0,
  }))
  return [...images, ...volumes].sort((a, b) => b.sizeBytes - a.sizeBytes)
}

const queue = computed(() => buildQueue())
const current = computed(() => queue.value[0] || null)
const remaining = computed(() => queue.value.length)

async function fetchUnused() {
  try {
    const [imgRes, volRes] = await Promise.all([
      fetch(`${apiUrl.value}/api/images`),
      fetch(`${apiUrl.value}/api/volumes`),
    ])
    const [imgData, volData] = await Promise.all([
      expectApiSuccess(imgRes, 'Failed to load images'),
      expectApiSuccess(volRes, 'Failed to load volumes'),
    ])
    unusedImages.value = Array.isArray(imgData.unusedImages) ? imgData.unusedImages : []
    unusedVolumes.value = Array.isArray(volData.unusedVolumes) ? volData.unusedVolumes : []
  } catch {
    // keep last known queue
  }
}

async function freeCurrent() {
  const item = current.value
  if (!item || deleting.value) return

  const confirmKey = item.kind === 'image'
    ? 'quickMetrics.biggestUnused.confirmImage'
    : 'quickMetrics.biggestUnused.confirmVolume'
  if (!confirm(t(confirmKey, { name: item.name }))) return

  deleting.value = true
  const freedSize = formatBytes(item.sizeBytes)

  try {
    // Match Images.vue / Volumes.vue — do not encodeURIComponent.
    // Image IDs are "sha256:…"; encoding the colon breaks chi's {id} param.
    const url = item.kind === 'image'
      ? `${apiUrl.value}/api/images/${item.id}`
      : `${apiUrl.value}/api/volumes/${item.id}`
    const response = await fetch(url, { method: 'DELETE' })
    const data = await response.json()

    if (!data.success) {
      toast.error(t('quickMetrics.biggestUnused.deleteFailed', {
        message: data.message || data.error || 'unknown error',
      }))
      return
    }

    toast.success(t('quickMetrics.biggestUnused.deleted', { size: freedSize }))
    await fetchUnused()
    flipKey.value += 1
  } catch (error) {
    toast.error(t('quickMetrics.biggestUnused.deleteFailed', {
      message: error.message || 'network error',
    }))
  } finally {
    deleting.value = false
  }
}

function goToManage() {
  const item = current.value
  if (!item) return
  router.push(item.kind === 'image' ? '/images' : '/volumes')
}

onMounted(() => {
  fetchUnused()
  refreshInterval = setInterval(fetchUnused, 30000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<template>
  <div
    v-if="current"
    class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- Header -->
    <div class="relative z-10 flex items-center gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-amber-200 bg-amber-50 transition-transform duration-300 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20">
        <HardDrive class="h-5 w-5 text-amber-600 dark:text-amber-400" />
      </div>
      <div>
        <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 dark:text-white">
          {{ t('quickMetrics.biggestUnused.title') }}
        </h3>
        <div class="mt-2 flex items-center gap-1.5">
          <div class="h-1.5 w-1.5 rounded-full bg-amber-500"></div>
          <span class="text-[10px] font-semibold uppercase tracking-widest text-amber-600 dark:text-amber-500">
            {{ t('quickMetrics.biggestUnused.subtitle') }}
          </span>
        </div>
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <Transition name="flip" mode="out-in">
        <div :key="`${current.kind}-${current.id}-${flipKey}`" class="flex flex-col">
          <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">
            {{ t('quickMetrics.biggestUnused.reclaimable') }}
          </div>

          <div class="mb-2 tabular-nums text-4xl font-black tracking-tighter text-zinc-900 sm:text-5xl dark:text-white">
            {{ formatBytes(current.sizeBytes) }}
          </div>

          <button
            type="button"
            class="mb-6 truncate text-left text-xs font-semibold text-zinc-500 transition-colors hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
            :title="current.name"
            @click.stop="goToManage"
          >
            <span class="inline-flex items-center gap-1.5">
              <component :is="current.kind === 'image' ? Layers : HardDrive" class="h-3.5 w-3.5 shrink-0" />
              <span class="truncate">{{ current.name }}</span>
              <span class="text-zinc-400">·</span>
              <span>{{ current.kind === 'image' ? t('quickMetrics.biggestUnused.image') : t('quickMetrics.biggestUnused.volume') }}</span>
              <span class="text-zinc-400">·</span>
              <span>{{ t('quickMetrics.biggestUnused.remaining') }} {{ remaining }}</span>
            </span>
          </button>

          <button
            type="button"
            :disabled="deleting"
            @click.stop="freeCurrent"
            class="flex w-full items-center justify-center gap-2 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-[10px] font-bold uppercase tracking-widest text-rose-600 transition-colors hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-rose-900/40 dark:bg-rose-900/20 dark:text-rose-400 dark:hover:bg-rose-900/30"
          >
            <Trash2 class="h-3.5 w-3.5" :class="{ 'animate-pulse': deleting }" />
            {{ deleting
              ? t('quickMetrics.biggestUnused.deleting')
              : t('quickMetrics.biggestUnused.freeSize', { size: formatBytes(current.sizeBytes) })
            }}
          </button>
        </div>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.flip-enter-active,
.flip-leave-active {
  transition: opacity 0.28s ease, transform 0.28s ease;
}

.flip-leave-to {
  opacity: 0;
  transform: rotateX(12deg) translateY(-10px);
}

.flip-enter-from {
  opacity: 0;
  transform: rotateX(-12deg) translateY(10px);
}
</style>
