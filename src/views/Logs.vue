<script setup>
import { ref, onMounted } from 'vue'

const logsData = ref({})
const logFilter = ref('all')
const loading = ref(false)
const apiUrl = ref('')

function formatTimestamp(timestamp) {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

async function fetchLogs() {
  loading.value = true
  try {
    const level = logFilter.value === 'all' ? '' : logFilter.value
    const url = level ? `${apiUrl.value}/api/logs?level=${level}` : `${apiUrl.value}/api/logs`
    const response = await fetch(url)
    const data = await response.json()
    if (data.success) {
      logsData.value = data
    }
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchLogs()
})
</script>

<template>
  <div class="p-6 md:p-10 lg:p-12">
    <h2 class="text-3xl font-bold mb-8 text-gray-800">Application Logs</h2>
    
    <div v-if="loading" class="text-center py-16">
      <div class="text-gray-500 font-medium">Loading logs...</div>
    </div>
    <div v-else>
      <!-- Controls -->
      <div class="flex items-center justify-between mb-6">
        <div class="flex gap-2">
          <button @click="logFilter = 'all'; fetchLogs()"
            :class="logFilter === 'all' ? 'bg-gradient-to-r from-blue-500 to-blue-600 text-white smooth-shadow' : 'glass text-gray-700'"
            class="px-4 py-2 rounded-xl text-sm font-semibold hover:scale-[1.02] transition-all">
            All ({{ logsData.count || 0 }})
          </button>
          <button @click="logFilter = 'info'; fetchLogs()"
            :class="logFilter === 'info' ? 'bg-gradient-to-r from-blue-500 to-blue-600 text-white smooth-shadow' : 'glass text-gray-700'"
            class="px-4 py-2 rounded-xl text-sm font-semibold hover:scale-[1.02] transition-all">
            Info
          </button>
          <button @click="logFilter = 'error'; fetchLogs()"
            :class="logFilter === 'error' ? 'bg-gradient-to-r from-red-500 to-red-600 text-white smooth-shadow' : 'glass text-gray-700'"
            class="px-4 py-2 rounded-xl text-sm font-semibold hover:scale-[1.02] transition-all">
            Errors
          </button>
        </div>
        <button @click="fetchLogs()"
          class="px-4 py-2 glass hover:bg-white/90 text-gray-700 rounded-xl text-sm font-medium transition-all">
          🔄 Refresh
        </button>
      </div>

      <!-- Logs List -->
      <div class="glass rounded-2xl overflow-hidden smooth-shadow">
        <div class="max-h-[600px] overflow-y-auto font-mono text-sm">
          <div v-if="!logsData.logs || logsData.logs.length === 0" class="p-12 text-center">
            <div class="text-5xl mb-4">📋</div>
            <div class="text-gray-500 font-medium">No logs available</div>
          </div>
          <div v-for="(logEntry, index) in logsData.logs" :key="index"
            :class="logEntry.level === 'error' ? 'bg-red-50/80 border-l-4 border-red-500' : 'border-b border-gray-100'"
            class="px-5 py-3 hover:bg-white/50 transition-colors">
            <div class="flex items-start gap-3">
              <span class="text-gray-400 text-xs whitespace-nowrap pt-0.5 font-semibold">
                {{ formatTimestamp(logEntry.timestamp) }}
              </span>
              <span :class="logEntry.level === 'error' ? 'text-red-600' : 'text-blue-600'"
                class="text-xs font-bold uppercase whitespace-nowrap pt-0.5">
                {{ logEntry.level }}
              </span>
              <span class="text-gray-800 flex-1 break-all">
                {{ logEntry.message }}
                <span v-if="logEntry.args" class="text-gray-600">
                  {{ logEntry.args.join(' ') }}
                </span>
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
