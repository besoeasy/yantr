<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Shield, ArrowRight, Key, CheckCircle, AlertCircle, Loader, Wifi } from 'lucide-vue-next'
import { useApiUrl } from '../composables/useApiUrl'

const router = useRouter()
const { apiUrl } = useApiUrl()

const authKey = ref('')
const deploying = ref(false)
const deployError = ref('')
const deploySuccess = ref(false)

// Tailscale auth key validation
// Valid formats:
//   tskey-auth-<nodekey>-<secret>  (newer ephemeral / reusable keys)
//   tskey-<secret>                 (legacy keys)
const isValidToken = computed(() => {
  const k = authKey.value.trim()
  if (!k.startsWith('tskey-')) return false
  if (k.length < 30) return false
  // No whitespace inside
  if (/\s/.test(k)) return false
  return true
})

const tokenState = computed(() => {
  const k = authKey.value.trim()
  if (!k) return 'empty'
  if (isValidToken.value) return 'valid'
  return 'invalid'
})

const perks = [
  'No port forwarding or firewall rules',
  'End-to-end WireGuard encryption',
  'Access all services by hostname',
  'Works behind CGNAT & double NAT',
]

async function deploy() {
  if (!isValidToken.value || deploying.value) return
  deploying.value = true
  deployError.value = ''
  try {
    const res = await fetch(`${apiUrl.value}/api/deploy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        appId: 'tailscale',
        environment: { TAILSCALE_AUTH_KEY: authKey.value.trim() },
      }),
    })
    const data = await res.json()
    if (!res.ok || !data.success) {
      deployError.value = data.error || data.message || 'Deployment failed'
    } else {
      deploySuccess.value = true
      setTimeout(() => router.push('/apps/tailscale'), 1500)
    }
  } catch (e) {
    deployError.value = e.message || 'Network error'
  } finally {
    deploying.value = false
  }
}
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">

    <!-- Hover accent line -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <!-- Dot-grid pattern -->
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Success overlay -->
    <transition
      enter-active-class="transition-all duration-500 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
    >
      <div v-if="deploySuccess" class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-white/95 dark:bg-[#0A0A0A]/95 rounded-xl">
        <div class="w-12 h-12 rounded-full bg-green-50 dark:bg-green-500/10 border border-green-200 dark:border-green-500/20 flex items-center justify-center mb-3">
          <CheckCircle class="w-6 h-6 text-green-600 dark:text-green-500" />
        </div>
        <p class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">Tailscale Deployed</p>
        <p class="text-[11px] text-gray-500 dark:text-zinc-400 mt-1 uppercase tracking-wider font-medium">Redirecting…</p>
      </div>
    </transition>

    <!-- Header -->
    <div class="relative z-10 flex items-start justify-between mb-5">
      <div class="flex items-center gap-4">
        <div class="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-500/10 border border-blue-100 dark:border-blue-500/20 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
          <Wifi class="w-5 h-5 text-blue-600 dark:text-blue-400" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
            Tailscale VPN
          </h3>
          <div class="flex items-center gap-2 mt-1">
            <div class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></div>
            <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider">Not Installed</span>
          </div>
        </div>
      </div>
      <div class="shrink-0 inline-flex items-center gap-1.5 px-2 py-1 rounded-md bg-amber-50 dark:bg-amber-500/10 border border-amber-100 dark:border-amber-500/20">
        <span class="text-[10px] font-bold uppercase tracking-wider text-amber-600 dark:text-amber-400">Recommended</span>
      </div>
    </div>

    <!-- Two-col body: description left, form right -->
    <div class="relative z-10 flex-1 grid grid-cols-1 sm:grid-cols-2 gap-6">

      <!-- Left: description + perks -->
      <div class="flex flex-col justify-between">
        <div>
          <p class="text-sm text-gray-500 dark:text-zinc-400 leading-relaxed mb-5">
            Secure remote access to your homelab from anywhere — zero port forwarding, zero firewall rules. Uses WireGuard under the hood.
          </p>
          <ul class="space-y-2.5">
            <li v-for="perk in perks" :key="perk.text" class="flex items-center gap-2.5">
              <div class="w-1.5 h-1.5 rounded-full bg-blue-400 dark:bg-blue-500 shrink-0"></div>
              <span class="text-[12px] font-medium text-gray-600 dark:text-zinc-300">{{ perk }}</span>
            </li>
          </ul>
        </div>
        <a
          href="https://login.tailscale.com/admin/settings/keys"
          target="_blank"
          rel="noopener noreferrer"
          class="mt-5 inline-flex items-center gap-1.5 text-[11px] font-semibold text-blue-600 dark:text-blue-400 hover:underline"
        >
          <Key class="w-3 h-3" />
          Generate auth key at login.tailscale.com ↗
        </a>
      </div>

      <!-- Right: form -->
      <div class="flex flex-col justify-between">
        <div class="space-y-3">
          <div>
            <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2">Auth Key</label>
            <div class="relative">
              <input
                v-model="authKey"
                type="text"
                placeholder="tskey-auth-…"
                autocomplete="off"
                spellcheck="false"
                class="w-full bg-gray-50 dark:bg-zinc-900 border rounded-lg px-3 py-2.5 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600 outline-none transition-all duration-200 pr-8"
                :class="{
                  'border-gray-200 dark:border-zinc-800 focus:border-blue-400 dark:focus:border-blue-500': tokenState === 'empty',
                  'border-green-300 dark:border-green-600 focus:border-green-400 dark:focus:border-green-500': tokenState === 'valid',
                  'border-red-300 dark:border-red-700 focus:border-red-400 dark:focus:border-red-600': tokenState === 'invalid',
                }"
              />
              <div class="absolute right-2.5 top-1/2 -translate-y-1/2">
                <CheckCircle v-if="tokenState === 'valid'" class="w-3.5 h-3.5 text-green-500" />
                <AlertCircle v-else-if="tokenState === 'invalid'" class="w-3.5 h-3.5 text-red-400" />
              </div>
            </div>
            <!-- Validation hint -->
            <div class="mt-1.5 h-4">
              <transition
                enter-active-class="transition-all duration-200 ease-out"
                enter-from-class="opacity-0 -translate-y-1"
                enter-to-class="opacity-100 translate-y-0"
                leave-active-class="transition-all duration-150 ease-in"
                leave-from-class="opacity-100 translate-y-0"
                leave-to-class="opacity-0 -translate-y-1"
              >
                <p v-if="tokenState === 'invalid'" class="text-[11px] text-red-500 dark:text-red-400 font-medium">
                  Must start with <span class="font-mono">tskey-</span> and be ≥ 30 chars
                </p>
                <p v-else-if="tokenState === 'valid'" class="text-[11px] text-green-600 dark:text-green-400 font-medium">
                  Token looks valid
                </p>
              </transition>
            </div>
          </div>

          <!-- Error message -->
          <transition
            enter-active-class="transition-all duration-300 ease-out"
            enter-from-class="opacity-0 translate-y-1"
            enter-to-class="opacity-100 translate-y-0"
          >
            <div v-if="deployError" class="flex items-center gap-2 p-2.5 rounded-lg bg-red-50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20">
              <AlertCircle class="w-3.5 h-3.5 text-red-500 shrink-0" />
              <p class="text-[11px] text-red-600 dark:text-red-400 font-medium">{{ deployError }}</p>
            </div>
          </transition>
        </div>

        <!-- Deploy button pushed to bottom -->
        <button
          @click="deploy"
          :disabled="!isValidToken || deploying"
          class="mt-5 w-full flex items-center justify-center gap-2 px-4 py-3 rounded-lg text-[11px] font-bold uppercase tracking-wider transition-all duration-200"
          :class="isValidToken && !deploying
            ? 'bg-gray-900 dark:bg-white text-white dark:text-gray-900 hover:-translate-y-0.5 hover:shadow-lg cursor-pointer'
            : 'bg-gray-100 dark:bg-zinc-800 text-gray-400 dark:text-zinc-500 cursor-not-allowed'"
        >
          <Loader v-if="deploying" class="w-3.5 h-3.5 animate-spin" />
          <Shield v-else class="w-3.5 h-3.5" />
          <span>{{ deploying ? 'Deploying…' : 'Deploy Tailscale' }}</span>
        </button>
      </div>
    </div>

    <!-- Footer -->
    <div class="relative z-10 mt-5 pt-4 border-t border-gray-100 dark:border-zinc-800 flex items-center justify-between">
      <div class="flex items-center gap-1.5 text-blue-600 dark:text-blue-500">
        <Shield class="w-3.5 h-3.5" />
        <span class="text-[11px] font-bold uppercase tracking-wider">Essential for Remote Access</span>
      </div>
      <button
        @click="router.push('/apps/tailscale')"
        class="flex items-center gap-1 text-[11px] font-semibold text-gray-400 dark:text-zinc-500 hover:text-gray-700 dark:hover:text-zinc-300 transition-colors group/link"
      >
        <span>View App</span>
        <ArrowRight class="w-3 h-3 transition-transform duration-200 group-hover/link:translate-x-0.5" />
      </button>
    </div>
  </div>
</template>
