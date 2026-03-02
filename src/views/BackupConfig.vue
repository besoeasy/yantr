<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'
import { useApiUrl } from '../composables/useApiUrl'
import { ArrowLeft, Cloud, Save, RefreshCw, Clock, Database, RotateCcw } from 'lucide-vue-next'

const router = useRouter()
const { t } = useI18n()
const toast = useNotification()
const { apiUrl } = useApiUrl()

const loading = ref(false)
const saving = ref(false)
const configured = ref(false)
const restoringFromRepo = ref(false)

// Form fields
const provider = ref('Other')
const endpoint = ref('')
const bucket = ref('')
const accessKey = ref('')
const secretKey = ref('')
const region = ref('us-east-1')
const resticPassword = ref('')

// Computed restic repo preview
const resticRepoPreview = computed(() => {
  if (!bucket.value) return ''
  const ep = (endpoint.value || '').replace(/\/$/, '')
  if (ep.includes('amazonaws.com')) {
    return `s3:${bucket.value}.s3.${region.value || 'us-east-1'}.amazonaws.com/yantr-restic`
  }
  if (!ep) return ''
  return `s3:${ep}/${bucket.value}/yantr-restic`
})

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
  if (!bucket.value || !accessKey.value || !secretKey.value) {
    toast.error(t('backupConfig.error.bucketAccessSecretRequired'))
    return
  }

  if (provider.value === 'Other' && !endpoint.value) {
    toast.error(t('backupConfig.error.endpointRequired'))
    return
  }

  if (!resticPassword.value && !configured.value) {
    toast.error(t('backupConfig.error.resticPasswordRequired'))
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
        region: region.value,
        resticPassword: resticPassword.value
      })
    })

    const data = await response.json()

    if (data.success) {
      toast.success(t('backupConfig.success.s3ConfigSaved'))
      configured.value = true
      accessKey.value = ''
      secretKey.value = ''
      resticPassword.value = ''
    } else {
      toast.error(data.error || t('backupConfig.error.failedToSaveConfig'))
    }
  } catch (error) {
    console.error('Failed to save config:', error)
    toast.error(t('backupConfig.error.failedToSaveConfig'))
  } finally {
    saving.value = false
  }
}

async function restoreFromRepo() {
  restoringFromRepo.value = true
  try {
    const res = await fetch(`${apiUrl.value}/api/backup/schedules/restore-from-repo`, { method: 'POST' })
    const data = await res.json()
    if (!data.success) throw new Error(data.error || 'Failed')
    if (data.imported === 0) {
      toast.error(t('backupConfig.error.noSchedulesFound'))
    } else {
      toast.success(t('backupConfig.success.restoredSchedules', { count: data.imported, plural: data.imported !== 1 ? 's' : '' }))
    }
  } catch (err) {
    toast.error(err.message || t('backupConfig.error.failedToRestoreSchedules'))
  } finally {
    restoringFromRepo.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-white font-sans">
    <!-- Header -->
    <header class="bg-white/80 dark:bg-[#0A0A0A]/80 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800 sticky top-0 z-30">
      <div class="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
        <div class="flex items-center gap-4">
          <router-link to="/" class="inline-flex items-center justify-center w-10 h-10 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-900 transition-colors text-gray-500 dark:text-zinc-400 group">
            <ArrowLeft :size="18" class="group-hover:-translate-x-0.5 transition-transform" />
          </router-link>

          <div class="h-4 w-px bg-gray-200 dark:bg-zinc-800"></div>

          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 flex items-center justify-center">
              <Cloud :size="16" class="text-gray-700 dark:text-zinc-300" />
            </div>
            <div>
              <h1 class="text-sm font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('backupConfig.title') }}</h1>
              <p class="text-xs font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.subtitle') }}</p>
            </div>
          </div>
        </div>

        <transition name="fade">
          <div v-if="configured" class="px-3 py-1 rounded-md bg-green-50 dark:bg-green-900/10 border border-green-200 dark:border-green-900/30 text-[10px] font-bold uppercase tracking-[0.2em] text-green-700 dark:text-green-500 flex items-center gap-1.5">
            <div class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></div>
            {{ t('backupConfig.configured') }}
          </div>
        </transition>
      </div>
    </header>

    <!-- Main Content -->
    <main class="max-w-2xl mx-auto px-4 py-8 sm:py-12">

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20 gap-4">
        <RefreshCw :size="24" class="animate-spin text-blue-500" />
        <span class="text-xs font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('backupConfig.loadingConfiguration') }}</span>
      </div>

      <!-- Configuration Form -->
      <transition name="fade" mode="out-in">
        <div v-if="!loading" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-4 sm:p-8 space-y-6 sm:space-y-8 shadow-sm">

          <!-- Info Banner -->
          <div class="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-900/30 rounded-lg p-5">
            <div class="flex gap-4">
              <Cloud :size="20" class="text-blue-600 dark:text-blue-500 shrink-0 mt-0.5" />
              <div class="text-sm">
                <p class="font-semibold text-blue-900 dark:text-blue-400 tracking-tight mb-1.5">{{ t('backupConfig.aboutBackupStorage') }}</p>
                <p class="text-blue-700 dark:text-blue-300/80 leading-relaxed font-medium">
                  {{ t('backupConfig.aboutBackupStorageDesc') }}
                </p>
              </div>
            </div>
          </div>

          <div class="space-y-6">
            <!-- Provider Selection -->
            <div>
              <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-2">
                {{ t('backupConfig.provider') }}
              </label>
              <select
                v-model="provider"
                class="w-full px-4 py-2.5 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm font-medium focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900 dark:text-white"
              >
                <option value="AWS">{{ t('backupConfig.awsS3') }}</option>
                <option value="Other">{{ t('backupConfig.s3Compatible') }}</option>
              </select>
            </div>

            <!-- Endpoint (for non-AWS) -->
            <transition name="fade">
              <div v-if="provider === 'Other'">
                <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-2">
                  {{ t('backupConfig.endpoint') }} <span class="text-red-500">*</span>
                </label>
                <input
                  v-model="endpoint"
                  type="text"
                  :placeholder="t('backupConfig.endpointPlaceholder')"
                  class="w-full px-4 py-2.5 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm font-medium focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600"
                />
                <p class="mt-2 text-[11px] font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.endpointHint') }}</p>
              </div>
            </transition>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
              <!-- Bucket -->
              <div>
                <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-2">
                  {{ t('backupConfig.bucketName') }} <span class="text-red-500">*</span>
                </label>
                <input
                  v-model="bucket"
                  type="text"
                  :placeholder="t('backupConfig.bucketPlaceholder')"
                  class="w-full px-4 py-2.5 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm font-medium focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600"
                />
                <p class="mt-2 text-[11px] font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.bucketHint') }}</p>
              </div>

              <!-- Region -->
              <div>
                <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-2">
                  {{ t('backupConfig.region') }}
                </label>
                <input
                  v-model="region"
                  type="text"
                  :placeholder="t('backupConfig.regionPlaceholder')"
                  class="w-full px-4 py-2.5 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm font-medium focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600"
                />
                <p class="mt-2 text-[11px] font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.regionHint') }}</p>
              </div>
            </div>

            <!-- Access Key -->
            <div>
              <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-2">
                {{ t('backupConfig.accessKey') }} <span class="text-red-500">*</span>
              </label>
              <input
                v-model="accessKey"
                type="text"
                :placeholder="configured ? t('backupConfig.accessKeyPlaceholderConfigured') : t('backupConfig.accessKeyPlaceholder')"
                class="w-full px-4 py-2.5 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm font-mono focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600"
              />
            </div>

            <!-- Secret Key -->
            <div>
              <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-2">
                {{ t('backupConfig.secretKey') }} <span class="text-red-500">*</span>
              </label>
              <input
                v-model="secretKey"
                type="password"
                :placeholder="configured ? t('backupConfig.secretKeyPlaceholderConfigured') : t('backupConfig.secretKeyPlaceholder')"
                class="w-full px-4 py-2.5 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm font-mono focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600"
              />
              <p class="mt-2 text-[11px] font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.secretKeyHint') }}</p>
            </div>

            <!-- Restic Password -->
            <div>
              <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-2">
                {{ t('backupConfig.resticPassword') }} <span class="text-red-500">*</span>
              </label>
              <input
                v-model="resticPassword"
                type="password"
                :placeholder="configured ? t('backupConfig.resticPasswordPlaceholderConfigured') : t('backupConfig.resticPasswordPlaceholder')"
                class="w-full px-4 py-2.5 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm font-mono focus:ring-1 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600"
              />
              <p class="mt-2 text-[11px] font-medium text-amber-600 dark:text-amber-500">
                {{ t('backupConfig.resticPasswordWarning') }}
              </p>
            </div>

            <!-- Restic Repo Preview -->
            <div v-if="resticRepoPreview" class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg px-4 py-3">
              <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 mb-1">{{ t('backupConfig.resticRepository') }}</p>
              <p class="font-mono text-[11px] text-gray-700 dark:text-zinc-300 break-all">{{ resticRepoPreview }}</p>
            </div>

          </div>

          <!-- Actions -->
          <div class="pt-6 flex gap-4 border-t border-gray-200 dark:border-zinc-800">
            <button
              @click="saveConfig"
              :disabled="saving"
              class="flex-1 inline-flex items-center justify-center gap-2 px-6 py-3 bg-gray-900 text-white dark:bg-white dark:text-gray-900 rounded-lg hover:bg-gray-800 dark:hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-bold text-xs uppercase tracking-wider"
            >
              <Save :size="16" v-if="!saving" />
              <RefreshCw :size="16" class="animate-spin" v-else />
              {{ saving ? t('backupConfig.saving') : t('backupConfig.saveConfiguration') }}
            </button>

            <button
              @click="router.push('/')"
              class="px-6 py-3 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-lg border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-700 transition-all font-bold text-xs uppercase tracking-wider"
            >
              {{ t('backupConfig.cancel') }}
            </button>
          </div>

          <!-- Backup Schedules shortcut (only when configured) -->
          <transition name="fade">
            <div v-if="configured" class="pt-2 space-y-2">
              <router-link
                to="/backup-schedules"
                class="w-full flex items-center justify-between px-5 py-3.5 rounded-lg border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-600 transition-all group"
              >
                <div class="flex items-center gap-3">
                  <Clock :size="15" class="text-gray-500 dark:text-zinc-500" />
                  <div>
                    <p class="text-xs font-semibold text-gray-900 dark:text-white">{{ t('backupConfig.backupSchedules') }}</p>
                    <p class="text-[11px] font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.backupSchedulesDesc') }}</p>
                  </div>
                </div>
                <ArrowLeft :size="14" class="text-gray-400 dark:text-zinc-600 rotate-180 group-hover:translate-x-0.5 transition-transform" />
              </router-link>

              <router-link
                to="/backup-volumes"
                class="w-full flex items-center justify-between px-5 py-3.5 rounded-lg border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-600 transition-all group"
              >
                <div class="flex items-center gap-3">
                  <Database :size="15" class="text-gray-500 dark:text-zinc-500" />
                  <div>
                    <p class="text-xs font-semibold text-gray-900 dark:text-white">{{ t('backupConfig.backedUpVolumes') }}</p>
                    <p class="text-[11px] font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.backedUpVolumesDesc') }}</p>
                  </div>
                </div>
                <ArrowLeft :size="14" class="text-gray-400 dark:text-zinc-600 rotate-180 group-hover:translate-x-0.5 transition-transform" />
              </router-link>

              <button
                @click="restoreFromRepo"
                :disabled="restoringFromRepo"
                class="w-full flex items-center justify-between px-5 py-3.5 rounded-lg border border-dashed border-gray-300 dark:border-zinc-700 hover:border-gray-400 dark:hover:border-zinc-500 hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <div class="flex items-center gap-3">
                  <RotateCcw :size="15" class="text-gray-500 dark:text-zinc-500 shrink-0" :class="{ 'animate-spin': restoringFromRepo }" />
                  <div class="text-left">
                    <p class="text-xs font-semibold text-gray-900 dark:text-white">{{ t('backupConfig.restoreSchedulesFromS3') }}</p>
                    <p class="text-[11px] font-medium text-gray-500 dark:text-zinc-500">{{ t('backupConfig.restoreSchedulesDesc') }}</p>
                  </div>
                </div>
              </button>
            </div>
          </transition>

          <!-- Security Note -->
          <div class="mt-8 pt-6">
            <p class="text-[11px] font-medium leading-relaxed text-gray-500 dark:text-zinc-500">
              <span class="font-bold text-gray-700 dark:text-zinc-300">{{ t('backupConfig.securityNote') }}</span> {{ t('backupConfig.securityNoteDesc', { file: 'backup-config.json' }) }}
            </p>
          </div>
        </div>
      </transition>
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
