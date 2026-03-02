<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Cloud, CheckCircle2, AlertCircle, Settings, ArrowRight, ShieldCheck, Zap } from 'lucide-vue-next'

const router = useRouter()
const { t } = useI18n()
const apiUrl = ref('')
const configured = ref(false)
const loading = ref(true)
const config = ref(null)

const status = computed(() => {
  if (loading.value) return { type: 'loading', text: t('quickMetrics.backupStatus.checking'), color: 'slate' }
  if (configured.value) return { type: 'configured', text: t('quickMetrics.backupStatus.active'), color: 'green' }
  return { type: 'not-configured', text: t('quickMetrics.backupStatus.setupRequired'), color: 'yellow' }
})

const isConfigured = computed(() => status.value.type === 'configured')

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
    console.error('Failed to check backup config:', error)
  } finally {
    loading.value = false
  }
}

function goToConfig() {
  router.push('/backup-config')
}

onMounted(() => {
  apiUrl.value = window.VITE_API_URL || ''
  checkConfig()
})
</script>

<template>
  <div
    @click="goToConfig"
    class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 cursor-pointer overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-1 hover:border-gray-300 dark:hover:border-zinc-600"
  >
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-500"
         :class="isConfigured ? 'bg-gradient-to-r from-transparent via-blue-500 to-transparent' : 'bg-gradient-to-r from-transparent via-amber-500 to-transparent'">
    </div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 flex items-start justify-between mb-6">
      <div class="flex items-center gap-4">
        <!-- Icon Container -->
        <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500"
             :class="isConfigured ? 'bg-blue-50/50 dark:bg-blue-500/10 border border-blue-100 dark:border-blue-500/20' : 'bg-amber-50/50 dark:bg-amber-500/10 border border-amber-100 dark:border-amber-500/20'">
          <Cloud class="w-5 h-5" :class="isConfigured ? 'text-blue-600 dark:text-blue-500' : 'text-amber-600 dark:text-amber-500'" />
        </div>
        
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
            {{ t('quickMetrics.backupStatus.title') }}
          </h3>
          <div class="flex items-center gap-2 mt-1">
             <div class="w-1.5 h-1.5 rounded-full"
                  :class="loading ? 'bg-gray-400 animate-pulse' : isConfigured ? 'bg-green-500' : 'bg-amber-500 animate-pulse'">
             </div>
             <span class="text-[11px] font-medium uppercase tracking-wider"
                   :class="isConfigured ? 'text-gray-500 dark:text-zinc-400' : 'text-amber-600 dark:text-amber-500'">
               {{ status.text }}
             </span>
          </div>
        </div>
      </div>
      
      <div class="w-8 h-8 flex items-center justify-center rounded-md bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 text-gray-400 group-hover:text-blue-500 group-hover:border-gray-300 dark:group-hover:border-zinc-700 transition-colors">
         <Settings class="w-4 h-4" />
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 flex-1 flex flex-col justify-end">
      <!-- Loading Skeleton -->
      <div v-if="loading" class="space-y-3 pt-4 border-t border-gray-100 dark:border-zinc-800/80">
        <div class="flex items-center justify-between">
          <div class="h-3 w-16 bg-gray-100 dark:bg-zinc-800 rounded animate-pulse"></div>
          <div class="h-3 w-24 bg-gray-100 dark:bg-zinc-800 rounded animate-pulse"></div>
        </div>
        <div class="flex items-center justify-between">
          <div class="h-3 w-16 bg-gray-100 dark:bg-zinc-800 rounded animate-pulse"></div>
          <div class="h-3 w-32 bg-gray-100 dark:bg-zinc-800 rounded animate-pulse"></div>
        </div>
      </div>

      <!-- Configured State -->
      <div v-else-if="configured && config" class="space-y-3 pt-4 border-t border-gray-100 dark:border-zinc-800/80">
        <div class="flex items-center justify-between">
          <span class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">Provider</span>
          <span class="text-xs font-semibold text-gray-900 dark:text-white">{{ config.provider }}</span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">Bucket</span>
          <span class="text-xs font-mono text-gray-700 dark:text-gray-300 truncate max-w-[150px]">{{ config.bucket }}</span>
        </div>
      </div>

      <!-- Not Configured State -->
      <div v-else class="pt-4 border-t border-gray-100 dark:border-zinc-800/80">
        <p class="text-sm font-medium text-gray-500 dark:text-zinc-400 leading-relaxed mb-4">
          {{ t('quickMetrics.backupStatus.description') }}
        </p>
        <div class="flex items-center gap-1.5 text-amber-600 dark:text-amber-500 transition-colors duration-300">
          <Zap class="w-3.5 h-3.5" />
          <span class="text-[11px] font-bold uppercase tracking-wider">
            {{ t('quickMetrics.backupStatus.recommended') }}
          </span>
        </div>
      </div>

      <div class="mt-4 pt-4 border-t border-gray-100 dark:border-zinc-800/80 flex items-center justify-between overflow-hidden">
        <div class="flex items-center gap-1.5 text-gray-400 dark:text-zinc-500 group-hover:text-gray-600 dark:group-hover:text-zinc-300 transition-colors duration-300">
          <span class="text-[10px] font-semibold uppercase tracking-[0.15em]">{{ t('quickMetrics.backupStatus.configuration') }}</span>
        </div>
        
        <div class="flex items-center gap-1 font-semibold text-xs transform translate-y-8 opacity-0 group-hover:translate-y-0 group-hover:opacity-100 transition-all duration-300 cubic-bezier(0.4, 0, 0.2, 1)"
             :class="isConfigured ? 'text-blue-600 dark:text-blue-400' : 'text-amber-600 dark:text-amber-500'">
          <span>{{ t('quickMetrics.backupStatus.manage') }}</span>
          <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform duration-300" />
        </div>
      </div>
    </div>
  </div>
</template>
