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
    class="group bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 p-5 hover:border-purple-500/50 dark:hover:border-purple-500/50 transition-all cursor-pointer shadow-sm hover:shadow-md"
  >
    <div class="flex items-start justify-between mb-3">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-lg flex items-center justify-center transition-all"
          :class="{
            'bg-green-100 dark:bg-green-900/30': status.type === 'configured',
            'bg-yellow-100 dark:bg-yellow-900/30': status.type === 'not-configured',
            'bg-slate-100 dark:bg-slate-900': status.type === 'loading'
          }">
          <Cloud :size="20"
            :class="{
              'text-green-600 dark:text-green-400': status.type === 'configured',
              'text-yellow-600 dark:text-yellow-400': status.type === 'not-configured',
              'text-slate-400': status.type === 'loading'
            }" />
        </div>
        <div>
          <h3 class="text-xs font-bold uppercase tracking-wider text-slate-500">MinIO / S3 Storage</h3>
          <p class="text-lg font-semibold text-slate-900 dark:text-white mt-0.5">{{ status.text }}</p>
        </div>
      </div>

      <button
        class="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-800"
      >
        <Settings :size="16" class="text-slate-400" />
      </button>
    </div>

    <!-- Status Details -->
    <div v-if="!loading" class="mt-4 pt-4 border-t border-slate-200 dark:border-slate-800">
      <div v-if="configured && config" class="space-y-2">
        <div class="flex items-center gap-2 text-xs">
          <CheckCircle2 :size="14" class="text-green-600 dark:text-green-400" />
          <span class="text-slate-600 dark:text-slate-400">Provider:</span>
          <span class="font-medium text-slate-900 dark:text-white">{{ config.provider }}</span>
        </div>
        <div class="flex items-center gap-2 text-xs">
          <CheckCircle2 :size="14" class="text-green-600 dark:text-green-400" />
          <span class="text-slate-600 dark:text-slate-400">Bucket:</span>
          <span class="font-mono text-slate-900 dark:text-white">{{ config.bucket }}</span>
        </div>
        <div v-if="config.endpoint" class="flex items-center gap-2 text-xs">
          <CheckCircle2 :size="14" class="text-green-600 dark:text-green-400" />
          <span class="text-slate-600 dark:text-slate-400">Endpoint:</span>
          <span class="font-mono text-xs text-slate-900 dark:text-white truncate">{{ config.endpoint }}</span>
        </div>
      </div>

      <div v-else class="flex items-start gap-2 text-xs">
        <AlertCircle :size="14" class="text-yellow-600 dark:text-yellow-400 shrink-0 mt-0.5" />
        <p class="text-yellow-700 dark:text-yellow-300">
          Configure S3/MinIO storage to enable volume backups
        </p>
      </div>
    </div>

    <!-- Loading Skeleton -->
    <div v-else class="mt-4 pt-4 border-t border-slate-200 dark:border-slate-800 space-y-2">
      <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
      <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded animate-pulse w-3/4"></div>
    </div>

    <!-- Click hint -->
    <div class="mt-3 text-[10px] text-slate-400 group-hover:text-purple-500 transition-colors flex items-center gap-1">
      <span>Click to configure</span>
      <span class="group-hover:translate-x-0.5 transition-transform">→</span>
    </div>
  </div>
</template>
