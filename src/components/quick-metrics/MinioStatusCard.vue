<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Cloud, CheckCircle2, AlertCircle, Settings } from 'lucide-vue-next'

const router = useRouter()
const apiUrl = ref('')
const configured = ref(false)
const loading = ref(true)
const config = ref(null)

const status = computed(() => {
  if (loading.value) return { type: 'loading', text: 'Checking...', color: 'slate' }
  if (configured.value) return { type: 'configured', text: 'Configured', color: 'green' }
  return { type: 'not-configured', text: 'Not Configured', color: 'yellow' }
})

const theme = computed(() => {
  if (status.value.type === 'configured') {
    return {
      text: 'text-emerald-600 dark:text-emerald-400',
      bg: 'bg-emerald-500/10 dark:bg-emerald-500/20',
      border: 'group-hover:border-emerald-500/30 dark:group-hover:border-emerald-400/30',
      pill: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
    }
  }
  if (status.value.type === 'not-configured') {
    return {
      text: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-500/10 dark:bg-amber-500/20',
      border: 'group-hover:border-amber-500/30 dark:group-hover:border-amber-400/30',
      pill: 'bg-amber-500/10 text-amber-700 dark:text-amber-300'
    }
  }
  return {
    text: 'text-slate-500 dark:text-slate-400',
    bg: 'bg-slate-500/10 dark:bg-slate-500/20',
    border: 'group-hover:border-slate-500/30 dark:group-hover:border-slate-400/30',
    pill: 'bg-slate-500/10 text-slate-600 dark:text-slate-400'
  }
})

async function checkConfig() {
  loading.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/backup/config`)
    const data = await response.json()

    if (data.success) {
      configured.value = data.configured
      if (data.configured) {
        config.value = data.config
      }
    }
  } catch (error) {
    console.error('Failed to check MinIO config:', error)
  } finally {
    loading.value = false
  }
}

function goToConfig() {
  router.push('/minioconfig')
}

onMounted(() => {
  apiUrl.value = window.VITE_API_URL || ''
  checkConfig()
})
</script>

<template>
  <div
    @click="goToConfig"
    class="relative h-full overflow-hidden group rounded-none bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-all duration-300 hover:shadow-lg dark:hover:shadow-slate-900/50 cursor-pointer"
    :class="theme.border"
  >
    <!-- Background Texture -->
    <div
      class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none"
      style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;"
    ></div>

    <div class="relative z-10 h-full p-6 flex flex-col justify-between">
      <!-- Header -->
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-3 min-w-0">
          <div class="p-2.5 rounded-xl transition-colors duration-300" :class="theme.bg">
            <Cloud class="w-5 h-5" :class="theme.text" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-bold text-slate-900 dark:text-white uppercase tracking-wider">MinIO / S3 Storage</h3>
            <div class="mt-1 inline-flex items-center gap-2 text-[11px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded-full" :class="theme.pill">
              <span>{{ status.text }}</span>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <div class="px-2.5 py-1 rounded-lg text-[10px] font-semibold uppercase tracking-wider border border-slate-200 dark:border-slate-700 text-slate-500 dark:text-slate-400 bg-slate-50 dark:bg-slate-800/70 opacity-0 group-hover:opacity-100 transition-opacity">
            Configure
          </div>
          <div class="p-2 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-400 opacity-0 group-hover:opacity-100 transition-opacity">
            <Settings class="w-4 h-4" />
          </div>
        </div>
      </div>

      <!-- Status Details -->
      <div v-if="!loading" class="mt-5 pt-4 border-t border-slate-100 dark:border-slate-800">
        <div v-if="configured && config" class="space-y-2.5">
          <div class="flex items-center gap-2 text-xs">
            <CheckCircle2 :size="14" :class="theme.text" />
            <span class="text-slate-500 dark:text-slate-400">Provider</span>
            <span class="ml-auto font-semibold text-slate-900 dark:text-white">{{ config.provider }}</span>
          </div>
          <div class="flex items-center gap-2 text-xs">
            <CheckCircle2 :size="14" :class="theme.text" />
            <span class="text-slate-500 dark:text-slate-400">Bucket</span>
            <span class="ml-auto font-mono text-slate-900 dark:text-white">{{ config.bucket }}</span>
          </div>
          <div v-if="config.endpoint" class="flex items-center gap-2 text-xs">
            <CheckCircle2 :size="14" :class="theme.text" />
            <span class="text-slate-500 dark:text-slate-400">Endpoint</span>
            <span class="ml-auto font-mono text-xs text-slate-900 dark:text-white truncate">{{ config.endpoint }}</span>
          </div>
        </div>

        <div v-else class="flex items-start gap-2 text-xs">
          <AlertCircle :size="14" class="text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
          <p class="text-amber-700 dark:text-amber-300">
            Configure S3/MinIO storage to enable volume backups
          </p>
        </div>
      </div>

      <!-- Loading Skeleton -->
      <div v-else class="mt-5 pt-4 border-t border-slate-100 dark:border-slate-800 space-y-2">
        <div class="h-3.5 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
        <div class="h-3.5 bg-slate-200 dark:bg-slate-800 rounded animate-pulse w-3/4"></div>
      </div>

      <!-- Click hint -->
      <div class="mt-4 text-[10px] text-slate-400 group-hover:text-slate-600 dark:group-hover:text-slate-300 transition-colors flex items-center gap-1">
        <span>Open configuration</span>
        <span class="group-hover:translate-x-0.5 transition-transform">→</span>
      </div>
    </div>
  </div>
</template>
