<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldCheck, RefreshCw, Globe, User, ArrowLeft } from '@lucide/vue'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useRouter } from 'vue-router'

const { apiUrl } = useApiUrl()
const { t } = useI18n()
const router = useRouter()

const proxies = ref([])
const caddyRunning = ref(false)
const loading = ref(false)
let refreshInterval = null

async function fetchProxies() {
  if (!proxies.value.length) loading.value = true
  try {
    const res = await fetch(`${apiUrl.value}/api/proxy`)
    const data = await expectApiSuccess(res, 'Failed to load proxies')
    proxies.value = Array.isArray(data.proxies) ? data.proxies : []
    caddyRunning.value = !!data.caddyRunning
  } catch {}
  finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchProxies()
  refreshInterval = setInterval(fetchProxies, 10000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

async function reload() {
  loading.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/proxy/reload`, { method: 'POST' })
    await expectApiSuccess(response, 'Failed to reload proxy config')
    await fetchProxies()
  } catch {}
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="h-full max-w-5xl mx-auto p-4 sm:p-6 lg:p-8 space-y-6 animate-in fade-in zoom-in-95 duration-300">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <button 
          @click="router.back()" 
          class="p-2.5 rounded-xl border border-gray-200 dark:border-zinc-800 bg-white dark:bg-[#0A0A0A] hover:bg-gray-50 dark:hover:bg-zinc-900 transition-colors shadow-sm"
        >
          <ArrowLeft class="w-5 h-5 text-gray-600 dark:text-zinc-400" />
        </button>
        <div>
          <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{{ t('proxiesView.title') }}</h1>
          <p class="text-sm text-gray-500 dark:text-zinc-400 mt-1">{{ t('proxiesView.subtitle') }}</p>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <div v-if="caddyRunning" class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-100 dark:border-emerald-500/20">
          <div class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></div>
          <span class="text-xs font-bold uppercase tracking-widest text-emerald-600 dark:text-emerald-400">{{ t('proxiesView.caddyLive') }}</span>
        </div>
        <button
          @click="reload"
          :disabled="loading"
          class="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-gray-200 dark:border-zinc-800 bg-white dark:bg-[#0A0A0A] hover:bg-gray-50 dark:hover:bg-zinc-900 text-sm font-semibold transition-all shadow-sm disabled:opacity-50 text-gray-700 dark:text-zinc-300 active:scale-95"
        >
          <RefreshCw class="w-4 h-4 text-gray-500" :class="{ 'animate-spin': loading }" />
          <span class="hidden sm:inline">{{ t('proxiesView.reloadConfig') }}</span>
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!loading && proxies.length === 0" class="flex flex-col items-center justify-center p-12 text-center bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-3xl border-dashed shadow-sm">
      <div class="w-16 h-16 rounded-2xl bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center mb-4 shadow-sm">
        <ShieldCheck class="w-8 h-8 text-gray-400 dark:text-zinc-500" />
      </div>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('proxiesView.noProxies') }}</h3>
      <p class="text-sm text-gray-500 dark:text-zinc-400 mt-2 max-w-sm mx-auto">{{ t('proxiesView.noProxiesDesc') }}</p>
    </div>

    <!-- Loading State -->
    <div v-else-if="loading && proxies.length === 0" class="flex flex-col items-center justify-center p-12 text-center bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-3xl shadow-sm">
      <RefreshCw class="w-8 h-8 text-gray-400 dark:text-zinc-500 animate-spin mb-4" />
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('proxiesView.loading') }}</h3>
    </div>

    <!-- Proxy List -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
      <div
        v-for="p in proxies"
        :key="p.projectId"
        class="group flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-2xl p-5 hover:border-emerald-200 dark:hover:border-emerald-900/50 hover:shadow-lg hover:-translate-y-1 transition-all duration-300 shadow-sm"
      >
        <div class="flex items-center gap-4 mb-5">
          <div class="w-12 h-12 rounded-xl bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 transition-transform duration-300 group-hover:scale-110">
            <Globe class="w-6 h-6 text-gray-600 dark:text-zinc-400 group-hover:text-emerald-500 transition-colors duration-300" />
          </div>
          <div class="min-w-0">
            <h4 class="font-bold text-gray-900 dark:text-white truncate text-base tracking-tight">{{ p.containerName }}</h4>
            <div class="flex items-center gap-1.5 mt-1 text-sm text-gray-500 dark:text-zinc-500 font-mono">
              <span class="font-bold text-emerald-600 dark:text-emerald-400">:{{ p.servePort }}</span>
              <span class="text-gray-300 dark:text-zinc-700">→</span>
              <span>:{{ p.targetPort }}</span>
            </div>
          </div>
        </div>

        <div v-if="p.authUser" class="mt-auto pt-4 border-t border-gray-100 dark:border-zinc-800/80">
          <div class="flex items-center gap-2 text-xs text-gray-600 dark:text-zinc-400 bg-gray-50 dark:bg-zinc-900/50 w-fit px-3 py-2 rounded-lg border border-gray-200/60 dark:border-zinc-800">
            <User class="w-4 h-4 text-gray-400 dark:text-zinc-500" />
            <span class="font-medium tracking-wide">Auth: {{ p.authUser }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
