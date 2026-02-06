<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'vue-toastification'
import { ArrowLeft, Cloud, Save, RefreshCw } from 'lucide-vue-next'

const router = useRouter()
const toast = useToast()

const apiUrl = ref('')
const loading = ref(false)
const saving = ref(false)
const configured = ref(false)

// Form fields
const provider = ref('Other')
const endpoint = ref('')
const bucket = ref('')
const accessKey = ref('')
const secretKey = ref('')
const region = ref('us-east-1')

// Fetch existing configuration
async function fetchConfig() {
  loading.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/backup/config`)
    const data = await response.json()

    if (data.success && data.configured) {
      configured.value = true
      provider.value = data.config.provider || 'Other'
      endpoint.value = data.config.endpoint || ''
      bucket.value = data.config.bucket || ''
      region.value = data.config.region || 'us-east-1'
      // Don't populate access/secret keys for security
    }
  } catch (error) {
    console.error('Failed to fetch config:', error)
    toast.error('Failed to load configuration')
  } finally {
    loading.value = false
  }
}

// Save configuration
async function saveConfig() {
  // Validation
  if (!bucket.value || !accessKey.value || !secretKey.value) {
    toast.error('Bucket, Access Key, and Secret Key are required')
    return
  }

  if (provider.value === 'Other' && !endpoint.value) {
    toast.error('Endpoint is required for custom S3 providers')
    return
  }

  saving.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/backup/config`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        provider: provider.value,
        endpoint: endpoint.value,
        bucket: bucket.value,
        accessKey: accessKey.value,
        secretKey: secretKey.value,
        region: region.value
      })
    })

    const data = await response.json()

    if (data.success) {
      toast.success('S3 configuration saved successfully')
      configured.value = true
      // Clear sensitive fields after save
      accessKey.value = ''
      secretKey.value = ''
    } else {
      toast.error(data.error || 'Failed to save configuration')
    }
  } catch (error) {
    console.error('Failed to save config:', error)
    toast.error('Failed to save configuration')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  apiUrl.value = window.VITE_API_URL || ''
  fetchConfig()
})
</script>

<template>
  <div class="min-h-screen bg-slate-50 dark:bg-[#09090b] text-slate-900 dark:text-slate-200 font-sans">
    <!-- Header -->
    <header class="bg-white dark:bg-[#0c0c0e] border-b border-slate-200 dark:border-slate-800">
      <div class="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between">
        <div class="flex items-center gap-4">
          <router-link to="/" class="inline-flex items-center justify-center w-8 h-8 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors text-slate-500">
            <ArrowLeft :size="18" />
          </router-link>

          <div class="h-4 w-px bg-slate-200 dark:bg-slate-800"></div>

          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center">
              <Cloud :size="18" class="text-purple-600 dark:text-purple-400" />
            </div>
            <div>
              <h1 class="text-sm font-semibold text-slate-900 dark:text-white">MinIO / S3 Configuration</h1>
              <p class="text-xs text-slate-500">Configure object storage for volume backups</p>
            </div>
          </div>
        </div>

        <div v-if="configured" class="px-3 py-1 rounded-full bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 text-xs font-semibold text-green-700 dark:text-green-400">
          Configured
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="max-w-3xl mx-auto px-4 py-8">

      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <RefreshCw :size="32" class="animate-spin text-slate-300" />
      </div>

      <!-- Configuration Form -->
      <div v-else class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 p-6 space-y-6">

        <!-- Info Banner -->
        <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
          <div class="flex gap-3">
            <Cloud :size="20" class="text-blue-600 dark:text-blue-400 shrink-0 mt-0.5" />
            <div class="text-sm text-blue-900 dark:text-blue-200">
              <p class="font-semibold mb-1">About S3 / MinIO Storage</p>
              <p class="text-blue-700 dark:text-blue-300">
                Configure S3-compatible object storage (AWS S3, MinIO, or other providers) to enable automatic volume backups.
                Backups are stored securely and can be restored from any container detail page.
              </p>
            </div>
          </div>
        </div>

        <!-- Provider Selection -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Provider
          </label>
          <select
            v-model="provider"
            class="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-md text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          >
            <option value="AWS">AWS S3</option>
            <option value="Other">MinIO / Other S3-Compatible</option>
          </select>
        </div>

        <!-- Endpoint (for non-AWS) -->
        <div v-if="provider === 'Other'">
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Endpoint <span class="text-red-500">*</span>
          </label>
          <input
            v-model="endpoint"
            type="text"
            placeholder="https://s3.example.com or https://minio.example.com:9000"
            class="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-md text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
          <p class="mt-1 text-xs text-slate-500">Full URL including protocol (http:// or https://)</p>
        </div>

        <!-- Bucket -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Bucket Name <span class="text-red-500">*</span>
          </label>
          <input
            v-model="bucket"
            type="text"
            placeholder="yantra"
            class="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-md text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
          <p class="mt-1 text-xs text-slate-500">The bucket must exist before saving configuration</p>
        </div>

        <!-- Region -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Region
          </label>
          <input
            v-model="region"
            type="text"
            placeholder="us-east-1"
            class="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-md text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
          <p class="mt-1 text-xs text-slate-500">Default: us-east-1 (can be left as-is for MinIO)</p>
        </div>

        <!-- Access Key -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Access Key <span class="text-red-500">*</span>
          </label>
          <input
            v-model="accessKey"
            type="text"
            :placeholder="configured ? '••••••••••••' : 'Enter access key'"
            class="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-md text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono"
          />
        </div>

        <!-- Secret Key -->
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Secret Key <span class="text-red-500">*</span>
          </label>
          <input
            v-model="secretKey"
            type="password"
            :placeholder="configured ? '••••••••••••' : 'Enter secret key'"
            class="w-full px-3 py-2 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-md text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono"
          />
          <p class="mt-1 text-xs text-slate-500">Required to update configuration</p>
        </div>

        <!-- Save Button -->
        <div class="pt-4 flex gap-3">
          <button
            @click="saveConfig"
            :disabled="saving"
            class="flex-1 inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-purple-600 text-white rounded-md hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-medium text-sm"
          >
            <Save :size="16" />
            {{ saving ? 'Saving...' : 'Save Configuration' }}
          </button>

          <button
            @click="router.push('/')"
            class="px-4 py-2.5 bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 rounded-md hover:bg-slate-200 dark:hover:bg-slate-700 transition-all font-medium text-sm"
          >
            Cancel
          </button>
        </div>

        <!-- Security Note -->
        <div class="pt-4 border-t border-slate-200 dark:border-slate-800">
          <p class="text-xs text-slate-500">
            <span class="font-semibold">Security Note:</span> Credentials are stored in <span class="font-mono bg-slate-100 dark:bg-slate-900 px-1 py-0.5 rounded">backup-config.json</span> on the server.
            Ensure your server is properly secured and credentials have minimal required permissions.
          </p>
        </div>
      </div>

    </main>
  </div>
</template>
