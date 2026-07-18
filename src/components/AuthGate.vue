<script setup>
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Copy, Check, LoaderCircle, LockKeyhole, Eye, EyeOff, AlertTriangle } from '@lucide/vue'
import { useNotification } from '../composables/useNotification'
import { useYantrAuth } from '../composables/useYantrAuth'
import { derivePrivateKey, getPublicKey } from '../utils/crypto.js'

const { t } = useI18n()
const toast = useNotification()
const { authState, loginYantr, setupYantrAdmin } = useYantrAuth()

const password = ref('')
const pin = ref('')
const passwordConfirm = ref('')
const pinConfirm = ref('')
const showPassword = ref(false)
const showPin = ref(false)
const submitting = ref(false)
const localError = ref('')
const passwordInput = ref(null)

// ─── Strength indicator ──────────────────────────────────────────────────────
const strength = computed(() => {
  const pw = password.value
  const p = pin.value
  if (!pw || !p) return { label: '', score: 0, color: '' }
  let score = 0
  if (pw.length >= 8) score += 1
  if (pw.length >= 12) score += 1
  if (/[A-Z]/.test(pw)) score += 1
  if (/[0-9]/.test(pw)) score += 1
  if (/[^A-Za-z0-9]/.test(pw)) score += 1
  if (p.length >= 4) score += 1
  if (p.length >= 8) score += 1
  if (score <= 2) return { label: 'Weak', score: 1, color: 'bg-red-500' }
  if (score <= 4) return { label: 'Fair', score: 2, color: 'bg-orange-500' }
  if (score <= 6) return { label: 'Good', score: 3, color: 'bg-yellow-500' }
  return { label: 'Strong', score: 4, color: 'bg-green-500' }
})

const strengthHint = computed(() => {
  if (!password.value || !pin.value) return ''
  return strength.value.label
})

// ─── Live public key derivation ───────────────────────────────────────────────
const derivedPublicKey = ref('')
const derivingKey = ref(false)
const copied = ref(false)
let deriveTimer = null

watch([password, pin], ([pw, p]) => {
  derivedPublicKey.value = ''
  clearTimeout(deriveTimer)
  if (!pw || !p) return
  deriveTimer = setTimeout(async () => {
    derivingKey.value = true
    try {
      const priv = await derivePrivateKey(pw, p)
      derivedPublicKey.value = getPublicKey(priv)
    } catch {
      derivedPublicKey.value = ''
    } finally {
      derivingKey.value = false
    }
  }, 300)
})

async function copyPublicKey() {
  if (!derivedPublicKey.value) return
  await navigator.clipboard.writeText(derivedPublicKey.value)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

// Background mouse canvas
const bgCanvas = ref(null)
let canvasCtx = null
let dots = []
let mouse = { x: 0, y: 0, active: false }
let raf = null

const isSetup = computed(() => !authState.configured)
const title = computed(() => authState.booting
  ? t('authGate.bootingTitle')
  : isSetup.value ? t('authGate.setupTitle') : t('authGate.loginTitle'))

const activeTab = ref(0)
const tabs = computed(() => isSetup.value
  ? [{ id: 0, label: 'Credentials' }, { id: 1, label: 'Confirm' }, { id: 2, label: 'Key' }]
  : [{ id: 0, label: 'Sign In' }]
)

function validate() {
  if (!String(password.value).trim()) return 'Password is required'
  if (!String(pin.value).trim()) return 'PIN is required'
  if (isSetup.value) {
    if (password.value !== passwordConfirm.value) return 'Passwords do not match'
    if (pin.value !== pinConfirm.value) return 'PINs do not match'
  }
  return ''
}

async function submit() {
  const errorMessage = validate()
  if (errorMessage) {
    localError.value = errorMessage
    return
  }

  submitting.value = true
  localError.value = ''

  try {
    const payload = { 
      password: password.value, 
      pin: pin.value 
    }
    if (isSetup.value) {
      await setupYantrAdmin(payload)
      toast.success(t('authGate.messages.setupComplete'))
    } else {
      await loginYantr(payload)
      toast.success(t('authGate.messages.loginComplete'))
    }
  } catch (error) {
    localError.value = error.message || t('authGate.errors.generic')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  nextTick(() => passwordInput.value?.focus())
  requestAnimationFrame(() => { initBackground() })
})

// === Premium interactive background ===
function initBackground() {
  const canvas = bgCanvas.value
  if (!canvas) return
  canvasCtx = canvas.getContext('2d', { alpha: true })

  function resize() {
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight
  }
  resize()
  window.addEventListener('resize', resize)

  dots = []
  const count = Math.min(160, Math.floor((canvas.width * canvas.height) / 12000))
  for (let i = 0; i < count; i++) {
    dots.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: 0.8 + Math.random() * 1.1,
      baseX: 0,
      baseY: 0,
      vx: (Math.random() - 0.5) * 0.15,
      vy: (Math.random() - 0.5) * 0.15,
    })
  }
  dots.forEach(d => { d.baseX = d.x; d.baseY = d.y })

  window.addEventListener('mousemove', handleMouse)
  window.addEventListener('mouseleave', handleMouseLeave)
  animateBackground()
}

function handleMouse(e) { mouse.x = e.clientX; mouse.y = e.clientY; mouse.active = true }
function handleMouseLeave() { mouse.active = false }

function animateBackground() {
  const canvas = bgCanvas.value
  const ctx = canvasCtx
  if (!ctx || !canvas) return
  ctx.clearRect(0, 0, canvas.width, canvas.height)

  const isDark = document.documentElement.classList.contains('dark')
  const dotColor = isDark ? 'rgba(148, 163, 184, 0.16)' : 'rgba(15, 23, 42, 0.12)'
  const lineColor = isDark ? 'rgba(148, 163, 184, 0.08)' : 'rgba(15, 23, 42, 0.07)'
  const mx = mouse.x; const my = mouse.y; const influence = mouse.active ? 78 : 0

  ctx.fillStyle = dotColor
  for (let i = 0; i < dots.length; i++) {
    const d = dots[i]
    if (mouse.active) {
      const dx = mx - d.x; const dy = my - d.y
      const dist = Math.sqrt(dx * dx + dy * dy) || 1
      if (dist < influence) {
        const force = (influence - dist) / influence
        d.x = d.x + (dx / dist) * force * -0.9
        d.y = d.y + (dy / dist) * force * -0.9
      }
    }
    d.x += (d.baseX - d.x) * 0.012 + d.vx
    d.y += (d.baseY - d.y) * 0.012 + d.vy
    ctx.beginPath(); ctx.arc(d.x, d.y, d.size, 0, Math.PI * 2); ctx.fill()
    if (mouse.active) {
      const dx = mx - d.x; const dy = my - d.y
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < 92 && dist > 4) {
        ctx.strokeStyle = lineColor; ctx.lineWidth = 0.7
        ctx.beginPath(); ctx.moveTo(d.x, d.y); ctx.lineTo(mx, my); ctx.stroke()
      }
    }
  }
  raf = requestAnimationFrame(animateBackground)
}

onUnmounted(() => {
  if (raf) cancelAnimationFrame(raf)
  window.removeEventListener('mousemove', handleMouse)
  window.removeEventListener('mouseleave', handleMouseLeave)
})
</script>

<template>
  <div class="min-h-screen bg-(--bg-body) text-(--text-primary) relative overflow-hidden">
    <canvas
      ref="bgCanvas"
      class="fixed inset-0 z-0 pointer-events-none"
      style="opacity: 0.85;"
    />

    <div class="relative z-10 min-h-screen flex items-center justify-center px-5 py-12">
      <div class="w-full max-w-[360px]">
        <div class="text-center mb-9">
          <div class="inline-flex items-baseline gap-1.5 text-[13px] font-semibold tracking-[3.5px] text-(--text-secondary) select-none">
            YANTR
          </div>
          <h1 class="mt-3 text-[42px] font-semibold tracking-[-1.8px] leading-none">
            {{ title }}
          </h1>
          <p class="mt-2.5 text-[13px] text-(--text-secondary)">
            {{ isSetup ? t('authGate.setupSubtitle') : t('authGate.loginSubtitle') }}
          </p>
        </div>

        <div
          class="relative rounded-3xl bg-(--surface) smooth-shadow-lg px-6 py-7 transition-all duration-300"
          :class="{ 'opacity-60 pointer-events-none': authState.booting }"
        >
          <div v-if="authState.booting" class="flex items-center justify-center min-h-[140px]">
            <div class="flex items-center gap-3 text-sm text-(--text-secondary)">
              <LoaderCircle class="h-4 w-4 animate-spin" />
              <span>{{ t('authGate.bootingState') }}</span>
            </div>
          </div>

          <form v-else @submit.prevent="submit" class="space-y-4">

            <!-- Step tabs (setup only) -->
            <div v-if="isSetup" class="flex gap-1 mb-2">
              <button
                v-for="tab in tabs"
                :key="tab.id"
                type="button"
                @click="activeTab = tab.id"
                class="flex-1 py-1.5 text-[10px] font-bold tracking-[1px] uppercase rounded-lg transition-all"
                :class="activeTab === tab.id
                  ? 'bg-(--text-primary) text-(--bg-body)'
                  : 'bg-(--surface-muted) text-(--text-secondary)'"
              >
                {{ tab.label }}
              </button>
            </div>

            <!-- Step 0: Password + PIN -->
            <template v-if="!isSetup || activeTab === 0">
              <!-- Password -->
              <div>
                <div class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                  <LockKeyhole class="h-3.5 w-3.5" />
                  <span>PASSWORD</span>
                </div>
                <div class="relative">
                  <input
                    ref="passwordInput"
                    v-model="password"
                    :type="showPassword ? 'text' : 'password'"
                    autocomplete="current-password"
                    placeholder="Enter your password"
                    class="w-full bg-(--surface-muted) rounded-2xl px-4 py-3.5 pr-11 text-[15px] font-medium placeholder:text-(--text-secondary)/60 focus:outline-none focus:-translate-y-px transition-all duration-200"
                    style="box-shadow: 0 1px 2px rgba(0,0,0,0.03);"
                  />
                  <button
                    type="button"
                    @click="showPassword = !showPassword"
                    class="absolute right-3.5 top-1/2 -translate-y-1/2 text-(--text-secondary) hover:text-(--text-primary) transition-colors"
                    tabindex="-1"
                  >
                    <Eye v-if="showPassword" class="h-4 w-4" />
                    <EyeOff v-else class="h-4 w-4" />
                  </button>
                </div>
                <!-- Strength -->
                <div v-if="password && pin && isSetup" class="mt-2 flex items-center gap-2">
                  <div class="flex-1 h-1 rounded-full bg-(--surface-muted) overflow-hidden">
                    <div
                      class="h-full rounded-full transition-all duration-300"
                      :class="strength.color"
                      :style="{ width: (strength.score / 4) * 100 + '%' }"
                    />
                  </div>
                  <span class="text-[10px] font-semibold text-(--text-secondary) shrink-0">{{ strengthHint }}</span>
                </div>
              </div>

              <!-- PIN -->
              <div>
                <div class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                  <LockKeyhole class="h-3.5 w-3.5" />
                  <span>PIN</span>
                </div>
                <div class="relative">
                  <input
                    v-model="pin"
                    :type="showPin ? 'text' : 'password'"
                    inputmode="numeric"
                    autocomplete="off"
                    placeholder="Enter your PIN"
                    class="w-full bg-(--surface-muted) rounded-2xl px-4 py-3.5 pr-11 text-[15px] font-medium placeholder:text-(--text-secondary)/60 focus:outline-none focus:-translate-y-px transition-all duration-200"
                    style="box-shadow: 0 1px 2px rgba(0,0,0,0.03);"
                  />
                  <button
                    type="button"
                    @click="showPin = !showPin"
                    class="absolute right-3.5 top-1/2 -translate-y-1/2 text-(--text-secondary) hover:text-(--text-primary) transition-colors"
                    tabindex="-1"
                  >
                    <Eye v-if="showPin" class="h-4 w-4" />
                    <EyeOff v-else class="h-4 w-4" />
                  </button>
                </div>
              </div>

              <!-- Next button (setup only) -->
              <button
                v-if="isSetup"
                type="button"
                @click="activeTab = 1"
                class="w-full mt-1 flex h-12 items-center justify-center gap-2 rounded-2xl bg-(--text-primary) text-(--bg-body) text-[13px] font-semibold tracking-[0.5px] active:scale-[0.985] transition-all duration-150 hover:shadow-xl"
              >
                Continue
              </button>
            </template>

            <!-- Step 1: Confirm (setup only) -->
            <template v-if="isSetup && activeTab === 1">
              <div class="flex items-center gap-2 px-1 mb-1">
                <AlertTriangle class="h-3.5 w-3.5 text-amber-500 shrink-0" />
                <span class="text-[11px] text-(--text-secondary)">Confirm both fields — this key cannot be recovered.</span>
              </div>

              <div>
                <div class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                  <LockKeyhole class="h-3.5 w-3.5" />
                  <span>CONFIRM PASSWORD</span>
                </div>
                <input
                  v-model="passwordConfirm"
                  type="password"
                  autocomplete="off"
                  placeholder="Re-enter password"
                  class="w-full bg-(--surface-muted) rounded-2xl px-4 py-3.5 text-[15px] font-medium placeholder:text-(--text-secondary)/60 focus:outline-none focus:-translate-y-px transition-all duration-200"
                  style="box-shadow: 0 1px 2px rgba(0,0,0,0.03);"
                />
                <p
                  v-if="passwordConfirm && password !== passwordConfirm"
                  class="mt-1 text-[10px] text-red-500 font-medium"
                >Passwords do not match</p>
              </div>

              <div>
                <div class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                  <LockKeyhole class="h-3.5 w-3.5" />
                  <span>CONFIRM PIN</span>
                </div>
                <input
                  v-model="pinConfirm"
                  type="password"
                  inputmode="numeric"
                  autocomplete="off"
                  placeholder="Re-enter PIN"
                  class="w-full bg-(--surface-muted) rounded-2xl px-4 py-3.5 text-[15px] font-medium placeholder:text-(--text-secondary)/60 focus:outline-none focus:-translate-y-px transition-all duration-200"
                  style="box-shadow: 0 1px 2px rgba(0,0,0,0.03);"
                />
                <p
                  v-if="pinConfirm && pin !== pinConfirm"
                  class="mt-1 text-[10px] text-red-500 font-medium"
                >PINs do not match</p>
              </div>

              <div class="flex gap-2">
                <button
                  type="button"
                  @click="activeTab = 0"
                  class="flex-1 h-12 rounded-2xl bg-(--surface-muted) text-(--text-secondary) text-[13px] font-semibold tracking-[0.5px] active:scale-[0.985] transition-all duration-150"
                >
                  Back
                </button>
                <button
                  type="button"
                  @click="activeTab = 2"
                  :disabled="password !== passwordConfirm || pin !== pinConfirm"
                  class="flex-1 h-12 rounded-2xl bg-(--text-primary) text-(--bg-body) text-[13px] font-semibold tracking-[0.5px] active:scale-[0.985] disabled:opacity-40 transition-all duration-150 hover:shadow-xl disabled:cursor-not-allowed"
                >
                  Continue
                </button>
              </div>
            </template>

            <!-- Step 2: Key review (setup only) / Live key display (login) -->
            <template v-if="(!isSetup) || (isSetup && activeTab === 2)">
              <div v-if="derivedPublicKey || derivingKey" class="rounded-2xl bg-(--surface-muted) px-4 py-3">
                <div class="flex items-center justify-between mb-1.5">
                  <span class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary)">
                    {{ isSetup ? 'YOUR PUBLIC KEY' : 'DERIVED PUBLIC KEY' }}
                  </span>
                  <button
                    v-if="derivedPublicKey"
                    type="button"
                    @click="copyPublicKey"
                    class="flex items-center gap-1 text-[10px] font-medium text-(--text-secondary) hover:text-(--text-primary) transition-colors"
                  >
                    <Check v-if="copied" class="h-3 w-3 text-green-500" />
                    <Copy v-else class="h-3 w-3" />
                    <span>{{ copied ? 'Copied' : 'Copy' }}</span>
                  </button>
                </div>
                <div v-if="derivingKey" class="flex items-center gap-2 text-(--text-secondary)">
                  <LoaderCircle class="h-3 w-3 animate-spin" />
                  <span class="text-[11px]">Deriving key…</span>
                </div>
                <p v-else class="font-mono text-[10px] break-all leading-relaxed text-(--text-primary) select-all">
                  {{ derivedPublicKey }}
                </p>
                <p v-if="isSetup" class="mt-2 text-[10px] text-(--text-secondary) leading-relaxed">
                  Save this to restrict access: <code class="font-mono">-e YANTR_ADMIN_PUBLIC_KEY=&lt;key&gt;</code>
                </p>
              </div>
              <p v-else class="text-[12px] text-(--text-secondary) leading-relaxed">
                {{ isSetup ? 'Set a password and PIN — your public key will appear here.' : 'Enter your password and PIN to derive your session key.' }}
              </p>

              <!-- Error -->
              <p
                v-if="localError || authState.error"
                class="text-xs font-medium text-red-600 dark:text-red-400 bg-red-500/5 px-3.5 py-2 rounded-xl"
              >
                {{ localError || authState.error }}
              </p>

              <!-- Submit -->
              <div v-if="isSetup" class="flex gap-2">
                <button
                  type="button"
                  @click="activeTab = 1"
                  class="flex-1 h-12 rounded-2xl bg-(--surface-muted) text-(--text-secondary) text-[13px] font-semibold tracking-[0.5px] active:scale-[0.985] transition-all duration-150"
                >
                  Back
                </button>
                <button
                  type="submit"
                  :disabled="submitting"
                  class="flex-1 h-12 flex items-center justify-center gap-2 rounded-2xl bg-(--text-primary) text-(--bg-body) text-[13px] font-semibold tracking-[0.5px] active:scale-[0.985] disabled:opacity-60 transition-all duration-150 hover:shadow-xl disabled:cursor-not-allowed"
                >
                  <LoaderCircle v-if="submitting" class="h-3.5 w-3.5 animate-spin" />
                  <span>{{ submitting ? t('authGate.working') : t('authGate.setupAction') }}</span>
                </button>
              </div>
              <button
                v-else
                type="submit"
                :disabled="submitting"
                class="group w-full mt-1 flex h-12 items-center justify-center gap-2 rounded-2xl bg-(--text-primary) text-(--bg-body) text-[13px] font-semibold tracking-[0.5px] active:scale-[0.985] disabled:opacity-60 transition-all duration-150 hover:shadow-xl disabled:cursor-not-allowed"
              >
                <LoaderCircle v-if="submitting" class="h-3.5 w-3.5 animate-spin" />
                <LockKeyhole v-else class="h-3.5 w-3.5 transition-transform group-hover:-translate-y-[0.5px]" />
                <span>{{ submitting ? t('authGate.working') : t('authGate.loginAction') }}</span>
              </button>
            </template>
          </form>
        </div>

        <div class="mt-6 text-center text-[10px] tracking-[0.5px] text-(--text-secondary)">
          {{ t('authGate.seedNote') }}
        </div>
      </div>
    </div>
  </div>
</template>