<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useApiUrl } from '../composables/useApiUrl'
import { FileText, AlertCircle, Info, RefreshCw, Terminal, Pause, Play, Trash2, Search, ArrowDown } from 'lucide-vue-next'

const { t } = useI18n()
const { apiUrl } = useApiUrl()
const logsData = ref({})
const logFilter = ref('all')
const loading = ref(false)
const autoRefresh = ref(true)
const searchQuery = ref('')
let refreshInterval = null
const logContainer = ref(null)

function formatTimestamp(timestamp) {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3
  })
}

async function fetchLogs() {
  if (!logsData.value.logs) loading.value = true
  
  try {
    const level = logFilter.value === 'all' ? '' : logFilter.value
    const url = level ? `${apiUrl.value}/api/logs?level=${level}` : `${apiUrl.value}/api/logs`
    const response = await fetch(url)
    const data = await response.json()
    
    // Only update if changes detected (naive check) or initial load
    if (data.success) {
      const currentLen = logsData.value.logs?.length || 0
      if (!logsData.value.logs || data.logs.length !== currentLen || (data.logs.length > 0 && data.logs[0].timestamp !== logsData.value.logs[0].timestamp)) {
        logsData.value = data
        if (autoRefresh.value) {
            scrollToBottom()
        }
      }
    }
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    loading.value = false
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

function clearLogs() {
  logsData.value = { ...logsData.value, logs: [], count: 0 }
}

onMounted(() => {
  fetchLogs()
  refreshInterval = setInterval(() => {
    if (autoRefresh.value) {
      fetchLogs()
    }
  }, 2000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<template>
  <div class="h-[calc(100vh-4rem)] flex flex-col bg-white dark:bg-[#0A0A0A] text-gray-800 dark:text-gray-300 font-mono text-[13px] overflow-hidden border border-gray-200 dark:border-zinc-800 rounded-xl mx-4 my-4 shadow-sm">
    <!-- Toolbar -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between px-4 py-3 bg-gray-50 dark:bg-zinc-900/50 border-b border-gray-200 dark:border-zinc-800 shrink-0 gap-3">
       <div class="flex items-center gap-4">
          <div class="flex items-center gap-2 text-gray-900 dark:text-white">
             <Terminal class="w-4 h-4" />
             <span class="font-bold uppercase tracking-[0.2em] text-[10px]">{{ t('logs.systemOutput') }}</span>
          </div>
          <div class="hidden sm:block h-4 w-px bg-gray-300 dark:bg-zinc-700"></div>
          
          <div class="flex bg-white dark:bg-[#0A0A0A] rounded-md border border-gray-200 dark:border-zinc-800 overflow-hidden shadow-sm">
             <button @click="logFilter = 'all'; fetchLogs()" 
                :class="logFilter === 'all' ? 'bg-gray-100 dark:bg-zinc-800 text-gray-900 dark:text-white font-medium' : 'text-gray-500 dark:text-zinc-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="px-3 py-1.5 text-xs transition-colors">
                {{ t('logs.all') }}
             </button>
             <div class="w-px bg-gray-200 dark:bg-zinc-800"></div>
             <button @click="logFilter = 'info'; fetchLogs()" 
                :class="logFilter === 'info' ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 font-medium' : 'text-gray-500 dark:text-zinc-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="px-3 py-1.5 text-xs transition-colors">
                {{ t('logs.info') }}
             </button>
             <div class="w-px bg-gray-200 dark:bg-zinc-800"></div>
             <button @click="logFilter = 'error'; fetchLogs()" 
                :class="logFilter === 'error' ? 'bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 font-medium' : 'text-gray-500 dark:text-zinc-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="px-3 py-1.5 text-xs transition-colors">
                {{ t('logs.errors') }}
             </button>
          </div>
       </div>

       <div class="flex items-center gap-2">
          <div class="relative group">
             <input v-model="searchQuery" type="text" :placeholder="t('logs.filterLogs')" class="bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 text-gray-900 dark:text-white px-3 py-1.5 rounded-md text-xs w-36 sm:w-48 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all placeholder-gray-400 dark:placeholder-zinc-600 shadow-sm" />
          </div>
          
          <div class="hidden sm:block h-4 w-px bg-gray-300 dark:bg-zinc-700 mx-2"></div>

          <button @click="autoRefresh = !autoRefresh" 
             :class="autoRefresh ? 'text-green-600 bg-green-50 dark:text-green-400 dark:bg-green-500/10 border-green-200 dark:border-green-500/20' : 'text-gray-500 border-gray-200 hover:bg-gray-100 dark:text-zinc-400 dark:border-zinc-800 dark:hover:bg-zinc-800'"
             class="p-1.5 rounded-md border transition-colors shadow-sm" 
             :title="autoRefresh ? t('logs.pauseAutoScroll') : t('logs.resumeAutoScroll')">
             <Pause v-if="autoRefresh" class="w-4 h-4" />
             <Play v-else class="w-4 h-4" />
          </button>
          
          <button @click="clearLogs" class="p-1.5 text-gray-500 border border-gray-200 hover:bg-red-50 hover:text-red-600 hover:border-red-200 dark:text-zinc-400 dark:border-zinc-800 dark:hover:bg-red-900/20 dark:hover:text-red-400 dark:hover:border-red-900/30 rounded-md transition-colors shadow-sm" :title="t('logs.clearConsole')">
             <Trash2 class="w-4 h-4" />
          </button>
       </div>
    </div>

    <!-- Log Viewer -->
    <div ref="logContainer" class="flex-1 overflow-y-auto p-2 scrollbar-thin bg-gray-50/50 dark:bg-[#050505]">
       <div v-if="loading && !logsData.logs" class="flex flex-col items-center justify-center h-full text-gray-400 dark:text-zinc-600 gap-3">
          <RefreshCw class="w-6 h-6 animate-spin" />
          <span class="text-xs font-semibold uppercase tracking-widest">{{ t('logs.connectingStream') }}</span>
       </div>

       <div v-else-if="!logsData.logs || logsData.logs.length === 0" class="flex flex-col items-center justify-center h-full text-gray-400 dark:text-zinc-600 gap-3">
          <Terminal class="w-8 h-8 opacity-50" />
          <span class="text-xs font-semibold uppercase tracking-widest">{{ t('logs.noOutputToDisplay') }}</span>
       </div>

       <div v-else class="font-mono">
          <div v-for="(log, idx) in logsData.logs" :key="idx" 
             v-show="!searchQuery || log.message.toLowerCase().includes(searchQuery.toLowerCase())"
             class="flex gap-4 px-3 py-1 hover:bg-white dark:hover:bg-[#0A0A0A] rounded group selection:bg-blue-500/30 transition-colors">
             
             <!-- Line Num -->
             <div class="select-none text-gray-400 dark:text-zinc-600 w-8 text-right text-[11px] pt-[2px] font-medium">{{ idx + 1 }}</div>
             
             <!-- Time -->
             <div class="select-none text-blue-600 dark:text-blue-400/80 w-16 sm:w-24 shrink-0 text-[10px] sm:text-[11px] pt-[2px] font-medium tracking-tight">{{ formatTimestamp(log.timestamp) }}</div>
             
             <!-- Level -->
             <div class="select-none w-[3px] rounded-full shrink-0 my-0.5" :class="log.level === 'error' ? 'bg-red-500' : 'bg-gray-300 dark:bg-zinc-700'"></div>

             <!-- Content -->
             <div class="flex-1 break-all whitespace-pre-wrap text-gray-700 dark:text-zinc-300 leading-relaxed text-xs">
                <span :class="log.level === 'error' ? 'text-red-600 dark:text-red-400 font-medium' : ''">{{ log.message }}</span>
                <span v-if="log.args && log.args.length" class="text-gray-500 dark:text-zinc-500 pl-2 text-[11px] italic">{{ log.args.join(' ') }}</span>
             </div>
          </div>
       </div>
    </div>
    
    <!-- Status Bar Footer -->
    <div class="bg-gray-100 dark:bg-zinc-900 border-t border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-400 px-4 py-2 text-[10px] font-bold uppercase tracking-wider flex justify-between items-center shrink-0">
       <div class="flex gap-6">
          <span class="flex items-center gap-1.5"><div class="w-1.5 h-1.5 rounded-full bg-green-500" :class="{'animate-pulse': autoRefresh}"></div> {{ t('logs.ln') }} {{ logsData.count || 0 }}</span>
          <span>{{ t('logs.utf8') }}</span>
       </div>
       <div class="flex gap-2 hover:text-gray-900 dark:hover:text-white transition-colors cursor-pointer" @click="scrollToBottom" :title="t('logs.scrollToBottom')">
          <ArrowDown class="w-3.5 h-3.5" />
       </div>
    </div>
  </div>
</template>

<style scoped>
.scrollbar-thin::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
.scrollbar-thin::-webkit-scrollbar-track {
  background: transparent;
}
.scrollbar-thin::-webkit-scrollbar-thumb {
  background-color: rgba(156, 163, 175, 0.3);
  border-radius: 4px;
}
:deep(.dark) .scrollbar-thin::-webkit-scrollbar-thumb {
  background-color: rgba(63, 63, 70, 0.5);
}
.scrollbar-thin::-webkit-scrollbar-thumb:hover {
  background-color: rgba(156, 163, 175, 0.5);
}
:deep(.dark) .scrollbar-thin::-webkit-scrollbar-thumb:hover {
  background-color: rgba(82, 82, 91, 0.8);
}
</style>