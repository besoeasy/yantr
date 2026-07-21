<script setup>
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  Copy, Check, LoaderCircle, LockKeyhole, Eye, EyeOff, AlertTriangle,
  ShieldCheck, Sparkles, KeyRound, Download, RefreshCw, CheckCircle2,
  ArrowRight, ArrowLeft, Shield
} from '@lucide/vue'
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
const confirmPasswordInput = ref(null)

// ─── Strength indicator ──────────────────────────────────────────────────────
const strengthCriteria = computed(() => {
  const pw = password.value
  const p = pin.value
  return [
    { label: 'Min 8 characters', met: pw.length >= 8 },
    { label: 'Uppercase & number', met: /[A-Z]/.test(pw) && /[0-9]/.test(pw) },
    { label: 'Special character', met: /[^A-Za-z0-9]/.test(pw) },
    { label: 'PIN (min 4 digits)', met: p.length >= 4 },
  ]
})

const strength = computed(() => {
  const pw = password.value
  const p = pin.value
  if (!pw || !p) return { label: 'Empty', score: 0, color: 'bg-zinc-600', textColor: 'text-zinc-400' }
  let score = 0
  if (pw.length >= 8) score += 1
  if (pw.length >= 12) score += 1
  if (/[A-Z]/.test(pw) && /[0-9]/.test(pw)) score += 1
  if (/[^A-Za-z0-9]/.test(pw)) score += 1
  if (p.length >= 4) score += 1
  
  if (score <= 2) return { label: 'Weak', score: 1, color: 'bg-rose-500', textColor: 'text-rose-400' }
  if (score <= 3) return { label: 'Fair', score: 2, color: 'bg-amber-500', textColor: 'text-amber-400' }
  if (score <= 4) return { label: 'Good', score: 3, color: 'bg-blue-500', textColor: 'text-blue-400' }
  return { label: 'Strong', score: 4, color: 'bg-emerald-500', textColor: 'text-emerald-400' }
})

// ─── Random Key Generator ───────────────────────────────────────────────────
function generateRandomSecret() {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()'
  let pw = ''
  const array = new Uint8Array(16)
  crypto.getRandomValues(array)
  for (let i = 0; i < 16; i++) {
    pw += chars[array[i] % chars.length]
  }
  let p = ''
  const pinArray = new Uint8Array(6)
  crypto.getRandomValues(pinArray)
  for (let i = 0; i < 6; i++) {
    p += (pinArray[i] % 10).toString()
  }
  password.value = pw
  pin.value = p
  passwordConfirm.value = pw
  pinConfirm.value = p
  showPassword.value = true
  showPin.value = true
  toast.success('Generated strong credentials')
}

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
  }, 250)
})

const formattedPublicKey = computed(() => {
  if (!derivedPublicKey.value) return ''
  return derivedPublicKey.value.match(/.{1,8}/g)?.join(' ') || derivedPublicKey.value
})

const keyAvatarGradient = computed(() => {
  if (!derivedPublicKey.value || derivedPublicKey.value.length < 64) {
    return 'linear-gradient(135deg, #6366f1 0%, #a855f7 100%)'
  }
  const hex = derivedPublicKey.value
  const c1 = `#${hex.slice(0, 6)}`
  const c2 = `#${hex.slice(10, 16)}`
  const c3 = `#${hex.slice(20, 26)}`
  return `linear-gradient(135deg, ${c1} 0%, ${c2} 50%, ${c3} 100%)`
})

async function copyPublicKey() {
  if (!derivedPublicKey.value) return
  await navigator.clipboard.writeText(derivedPublicKey.value)
  copied.value = true
  toast.success(t('authGate.messages.publicKeyCopied'))
  setTimeout(() => { copied.value = false }, 2000)
}

function downloadKeyBackup() {
  if (!derivedPublicKey.value) return
  const backupData = `=====================================================
YANTR ED25519 ADMIN IDENTITY BACKUP
=====================================================
Generated: ${new Date().toISOString()}

PUBLIC KEY:
${derivedPublicKey.value}

DOCKER ENVIRONMENT VARIABLE:
YANTR_ADMIN_PUBLIC_KEY=${derivedPublicKey.value}

CRITICAL NOTICE:
Yantr uses stateless Ed25519 authentication derived from
your Password and PIN. Your private key is NEVER stored
on the server or database. If lost, your account cannot be recovered.
=====================================================
`
  const blob = new Blob([backupData], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `yantr-admin-key-${derivedPublicKey.value.slice(0, 8)}.txt`
  a.click()
  URL.revokeObjectURL(url)
  toast.success('Backup file downloaded')
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
const steps = computed(() => [
  { id: 0, title: 'Credentials', icon: LockKeyhole },
  { id: 1, title: 'Confirm', icon: ShieldCheck },
  { id: 2, title: 'Identity Key', icon: KeyRound }
])

function goToStep(stepIndex) {
  if (!isSetup.value) return
  if (stepIndex === 1) {
    if (!password.value || !pin.value) {
      localError.value = 'Please fill in Password and PIN'
      return
    }
    if (password.value.length < 8) {
      localError.value = 'Password must be at least 8 characters'
      return
    }
    localError.value = ''
    activeTab.value = 1
    nextTick(() => confirmPasswordInput.value?.focus())
  } else if (stepIndex === 2) {
    if (password.value !== passwordConfirm.value) {
      localError.value = 'Passwords do not match'
      return
    }
    if (pin.value !== pinConfirm.value) {
      localError.value = 'PINs do not match'
      return
    }
    localError.value = ''
    activeTab.value = 2
  } else {
    localError.value = ''
    activeTab.value = 0
  }
}

function handleFormSubmit() {
  if (!isSetup.value) {
    submit()
    return
  }
  if (activeTab.value === 0) {
    goToStep(1)
  } else if (activeTab.value === 1) {
    goToStep(2)
  } else if (activeTab.value === 2) {
    submit()
  }
}

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
  <div class="min-h-screen bg-(--bg-body) text-(--text-primary) relative overflow-hidden flex items-center justify-center p-4">
    <canvas
      ref="bgCanvas"
      class="fixed inset-0 z-0 pointer-events-none opacity-85"
    />

    <div class="relative z-10 w-full max-w-[420px]">
      <!-- Brand Header -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-(--surface-muted) border border-(--border-subtle) text-[11px] font-bold tracking-[3px] text-(--text-secondary) select-none shadow-xs">
          <Shield class="h-3.5 w-3.5 text-indigo-500" />
          <span>YANTR IDENTITY</span>
        </div>
        <h1 class="mt-3.5 text-3xl sm:text-4xl font-extrabold tracking-tight leading-tight text-(--text-primary)">
          {{ title }}
        </h1>
        <p class="mt-2 text-xs sm:text-sm text-(--text-secondary) max-w-sm mx-auto">
          {{ isSetup ? 'Derive your stateless Ed25519 cryptographic key pair.' : t('authGate.loginSubtitle') }}
        </p>
      </div>

      <!-- Main Card -->
      <div
        class="relative rounded-3xl bg-(--surface) border border-(--border-subtle) shadow-2xl px-6 py-7 transition-all duration-300 backdrop-blur-xl"
        :class="{ 'opacity-60 pointer-events-none': authState.booting }"
      >
        <!-- Booting Spinner -->
        <div v-if="authState.booting" class="flex items-center justify-center min-h-[180px]">
          <div class="flex items-center gap-3 text-sm font-medium text-(--text-secondary)">
            <LoaderCircle class="h-5 w-5 animate-spin text-indigo-500" />
            <span>{{ t('authGate.bootingState') }}</span>
          </div>
        </div>

        <form v-else @submit.prevent="handleFormSubmit" class="space-y-5">
          <!-- Step Wizard Nav (Setup Mode) -->
          <div v-if="isSetup" class="space-y-3">
            <div class="flex items-center justify-between px-1">
              <span class="text-[11px] font-bold uppercase tracking-wider text-(--text-secondary)">
                Step {{ activeTab + 1 }} of 3: {{ steps[activeTab].title }}
              </span>
              <button 
                type="button" 
                @click="generateRandomSecret" 
                class="inline-flex items-center gap-1.5 text-[11px] font-semibold text-indigo-500 hover:text-indigo-400 transition-colors"
              >
                <Sparkles class="h-3.5 w-3.5" />
                <span>Auto-Generate</span>
              </button>
            </div>

            <!-- Progress Bar -->
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="step in steps"
                :key="step.id"
                type="button"
                @click="goToStep(step.id)"
                class="h-1.5 rounded-full transition-all duration-300"
                :class="activeTab >= step.id ? 'bg-indigo-500 shadow-xs' : 'bg-(--surface-muted)'"
              />
            </div>
          </div>

          <!-- STEP 0: Credentials (Setup or Login) -->
          <template v-if="!isSetup || activeTab === 0">
            <!-- Password Input -->
            <div>
              <div class="text-[11px] font-bold tracking-wider uppercase text-(--text-secondary) mb-1.5 flex items-center justify-between">
                <span class="flex items-center gap-1.5">
                  <LockKeyhole class="h-3.5 w-3.5 text-indigo-500" />
                  <span>PASSWORD</span>
                </span>
                <span v-if="isSetup && password" class="text-[10px] font-semibold" :class="strength.textColor">
                  {{ strength.label }}
                </span>
              </div>
              <div class="relative">
                <input
                  ref="passwordInput"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                  placeholder="Enter secure master password"
                  class="w-full bg-(--surface-muted) border border-(--border-subtle) rounded-2xl px-4 py-3.5 pr-11 text-sm font-medium placeholder:text-(--text-secondary)/50 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 transition-all"
                />
                <button
                  type="button"
                  @click="showPassword = !showPassword"
                  class="absolute right-3.5 top-1/2 -translate-y-1/2 text-(--text-secondary) hover:text-(--text-primary) transition-colors p-1"
                  tabindex="-1"
                >
                  <Eye v-if="showPassword" class="h-4 w-4" />
                  <EyeOff v-else class="h-4 w-4" />
                </button>
              </div>
            </div>

            <!-- PIN Input -->
            <div>
              <div class="text-[11px] font-bold tracking-wider uppercase text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                <LockKeyhole class="h-3.5 w-3.5 text-indigo-500" />
                <span>PIN CODE</span>
              </div>
              <div class="relative">
                <input
                  v-model="pin"
                  :type="showPin ? 'text' : 'password'"
                  inputmode="numeric"
                  autocomplete="off"
                  placeholder="Enter numeric PIN (e.g. 1234)"
                  class="w-full bg-(--surface-muted) border border-(--border-subtle) rounded-2xl px-4 py-3.5 pr-11 text-sm font-medium placeholder:text-(--text-secondary)/50 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 transition-all"
                />
                <button
                  type="button"
                  @click="showPin = !showPin"
                  class="absolute right-3.5 top-1/2 -translate-y-1/2 text-(--text-secondary) hover:text-(--text-primary) transition-colors p-1"
                  tabindex="-1"
                >
                  <Eye v-if="showPin" class="h-4 w-4" />
                  <EyeOff v-else class="h-4 w-4" />
                </button>
              </div>
            </div>

            <!-- Strength Criteria checklist (Setup mode) -->
            <div v-if="isSetup && (password || pin)" class="p-3 rounded-2xl bg-(--surface-muted)/60 border border-(--border-subtle) space-y-1.5">
              <div class="flex items-center justify-between text-xs font-semibold mb-2">
                <span class="text-(--text-secondary)">Credential Strength</span>
                <div class="flex gap-1">
                  <div 
                    v-for="i in 4" 
                    :key="i"
                    class="h-1.5 w-5 rounded-full transition-colors"
                    :class="i <= strength.score ? strength.color : 'bg-zinc-700/40'"
                  />
                </div>
              </div>
              <div class="grid grid-cols-2 gap-1.5 text-[11px]">
                <div 
                  v-for="(item, idx) in strengthCriteria" 
                  :key="idx" 
                  class="flex items-center gap-1.5"
                  :class="item.met ? 'text-emerald-500 font-medium' : 'text-(--text-secondary)/60'"
                >
                  <CheckCircle2 v-if="item.met" class="h-3 w-3 shrink-0" />
                  <div v-else class="h-3 w-3 rounded-full border border-current opacity-40 shrink-0" />
                  <span>{{ item.label }}</span>
                </div>
              </div>
            </div>

            <!-- Continue Button (Setup Mode Step 0) -->
            <button
              v-if="isSetup"
              type="button"
              @click="goToStep(1)"
              class="w-full flex h-12 items-center justify-center gap-2 rounded-2xl bg-indigo-600 hover:bg-indigo-500 active:scale-[0.985] text-white text-sm font-semibold tracking-wide transition-all shadow-lg shadow-indigo-500/25"
            >
              <span>Continue to Confirmation</span>
              <ArrowRight class="h-4 w-4" />
            </button>
          </template>

          <!-- STEP 1: Confirmation (Setup Mode Only) -->
          <template v-if="isSetup && activeTab === 1">
            <div class="p-3 rounded-2xl bg-amber-500/10 border border-amber-500/20 text-amber-500 text-xs flex items-start gap-2.5">
              <AlertTriangle class="h-4 w-4 shrink-0 mt-0.5" />
              <span>Confirm both credentials carefully. Your Ed25519 identity key is derived directly from them.</span>
            </div>

            <!-- Confirm Password -->
            <div>
              <div class="text-[11px] font-bold tracking-wider uppercase text-(--text-secondary) mb-1.5 flex items-center justify-between">
                <span class="flex items-center gap-1.5">
                  <ShieldCheck class="h-3.5 w-3.5 text-indigo-500" />
                  <span>CONFIRM PASSWORD</span>
                </span>
                <span v-if="passwordConfirm && password === passwordConfirm" class="text-[10px] text-emerald-500 font-semibold flex items-center gap-1">
                  <Check class="h-3 w-3" /> Matches
                </span>
              </div>
              <input
                ref="confirmPasswordInput"
                v-model="passwordConfirm"
                type="password"
                autocomplete="off"
                placeholder="Re-enter master password"
                class="w-full bg-(--surface-muted) border border-(--border-subtle) rounded-2xl px-4 py-3.5 text-sm font-medium placeholder:text-(--text-secondary)/50 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 transition-all"
              />
            </div>

            <!-- Confirm PIN -->
            <div>
              <div class="text-[11px] font-bold tracking-wider uppercase text-(--text-secondary) mb-1.5 flex items-center justify-between">
                <span class="flex items-center gap-1.5">
                  <ShieldCheck class="h-3.5 w-3.5 text-indigo-500" />
                  <span>CONFIRM PIN</span>
                </span>
                <span v-if="pinConfirm && pin === pinConfirm" class="text-[10px] text-emerald-500 font-semibold flex items-center gap-1">
                  <Check class="h-3 w-3" /> Matches
                </span>
              </div>
              <input
                v-model="pinConfirm"
                type="password"
                inputmode="numeric"
                autocomplete="off"
                placeholder="Re-enter PIN"
                class="w-full bg-(--surface-muted) border border-(--border-subtle) rounded-2xl px-4 py-3.5 text-sm font-medium placeholder:text-(--text-secondary)/50 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 transition-all"
              />
            </div>

            <!-- Actions Step 1 -->
            <div class="flex gap-2 pt-1">
              <button
                type="button"
                @click="goToStep(0)"
                class="flex-1 h-12 flex items-center justify-center gap-1.5 rounded-2xl bg-(--surface-muted) hover:bg-(--surface-muted)/80 text-(--text-primary) text-sm font-semibold transition-all"
              >
                <ArrowLeft class="h-4 w-4" />
                <span>Back</span>
              </button>
              <button
                type="button"
                @click="goToStep(2)"
                :disabled="!passwordConfirm || !pinConfirm || password !== passwordConfirm || pin !== pinConfirm"
                class="flex-1 h-12 flex items-center justify-center gap-2 rounded-2xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 text-white text-sm font-semibold transition-all shadow-lg shadow-indigo-500/25 disabled:cursor-not-allowed"
              >
                <span>Review Key</span>
                <ArrowRight class="h-4 w-4" />
              </button>
            </div>
          </template>

          <!-- STEP 2: Identity Key Review (Setup) / Key Badge (Login) -->
          <template v-if="(!isSetup) || (isSetup && activeTab === 2)">
            <!-- Key Card Component -->
            <div class="rounded-2xl border border-indigo-500/30 bg-gradient-to-b from-indigo-500/10 to-transparent p-4 space-y-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2.5">
                  <div 
                    class="h-8 w-8 rounded-xl shadow-xs transition-all duration-500" 
                    :style="{ background: keyAvatarGradient }"
                  />
                  <div>
                    <div class="text-xs font-bold tracking-wide uppercase text-(--text-primary)">
                      {{ isSetup ? 'Generated Ed25519 Public Key' : 'Session Public Key' }}
                    </div>
                    <div class="text-[10px] text-(--text-secondary)">
                      {{ derivedPublicKey ? `Identity Fingerprint: ${derivedPublicKey.slice(0, 8)}...` : 'Enter credentials above' }}
                    </div>
                  </div>
                </div>

                <div v-if="derivedPublicKey" class="flex gap-1">
                  <button
                    type="button"
                    @click="copyPublicKey"
                    class="p-2 rounded-xl bg-(--surface-muted) hover:bg-(--surface-muted)/80 text-(--text-secondary) hover:text-(--text-primary) transition-colors"
                    title="Copy Public Key"
                  >
                    <Check v-if="copied" class="h-4 w-4 text-emerald-500" />
                    <Copy v-else class="h-4 w-4" />
                  </button>
                  <button
                    v-if="isSetup"
                    type="button"
                    @click="downloadKeyBackup"
                    class="p-2 rounded-xl bg-(--surface-muted) hover:bg-(--surface-muted)/80 text-(--text-secondary) hover:text-(--text-primary) transition-colors"
                    title="Download Key Backup"
                  >
                    <Download class="h-4 w-4" />
                  </button>
                </div>
              </div>

              <!-- Key Code Box -->
              <div v-if="derivingKey" class="flex items-center justify-center py-4 text-xs text-(--text-secondary) gap-2">
                <LoaderCircle class="h-4 w-4 animate-spin text-indigo-500" />
                <span>Computing cryptographic Ed25519 key...</span>
              </div>
              <div v-else-if="derivedPublicKey" class="bg-(--bg-body) border border-(--border-subtle) rounded-xl p-3 font-mono text-[11px] break-all leading-relaxed text-indigo-400 selection:bg-indigo-500 selection:text-white">
                {{ formattedPublicKey }}
              </div>
              <div v-else class="text-xs text-(--text-secondary) italic py-2 text-center">
                Key will be derived when Password and PIN are entered.
              </div>

              <!-- Backup Notice -->
              <p v-if="isSetup && derivedPublicKey" class="text-[11px] text-(--text-secondary) leading-snug">
                💡 <span class="font-semibold text-(--text-primary)">Save this key:</span> Pass it as <code class="bg-(--surface-muted) px-1.5 py-0.5 rounded text-indigo-400 font-mono text-[10px]">YANTR_ADMIN_PUBLIC_KEY</code> to enforce server-side identity verification.
              </p>
            </div>

            <!-- Error Banner -->
            <div
              v-if="localError || authState.error"
              class="p-3 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-500 text-xs font-medium flex items-center gap-2"
            >
              <AlertTriangle class="h-4 w-4 shrink-0" />
              <span>{{ localError || authState.error }}</span>
            </div>

            <!-- Final Submit Actions -->
            <div v-if="isSetup" class="flex gap-2 pt-1">
              <button
                type="button"
                @click="goToStep(1)"
                class="flex-1 h-12 flex items-center justify-center gap-1.5 rounded-2xl bg-(--surface-muted) hover:bg-(--surface-muted)/80 text-(--text-primary) text-sm font-semibold transition-all"
              >
                <ArrowLeft class="h-4 w-4" />
                <span>Back</span>
              </button>
              <button
                type="submit"
                :disabled="submitting || !derivedPublicKey"
                class="flex-1 h-12 flex items-center justify-center gap-2 rounded-2xl bg-indigo-600 hover:bg-indigo-500 active:scale-[0.985] disabled:opacity-60 text-white text-sm font-semibold tracking-wide transition-all shadow-lg shadow-indigo-500/25 disabled:cursor-not-allowed"
              >
                <LoaderCircle v-if="submitting" class="h-4 w-4 animate-spin" />
                <Sparkles v-else class="h-4 w-4" />
                <span>{{ submitting ? t('authGate.working') : t('authGate.setupAction') }}</span>
              </button>
            </div>

            <!-- Login Submit Action -->
            <button
              v-else
              type="submit"
              :disabled="submitting || !password || !pin"
              class="group w-full flex h-12 items-center justify-center gap-2 rounded-2xl bg-indigo-600 hover:bg-indigo-500 active:scale-[0.985] disabled:opacity-60 text-white text-sm font-semibold tracking-wide transition-all shadow-lg shadow-indigo-500/25 disabled:cursor-not-allowed"
            >
              <LoaderCircle v-if="submitting" class="h-4 w-4 animate-spin" />
              <LockKeyhole v-else class="h-4 w-4 transition-transform group-hover:-translate-y-0.5" />
              <span>{{ submitting ? t('authGate.working') : t('authGate.loginAction') }}</span>
            </button>
          </template>
        </form>
      </div>

      <!-- Footer Note -->
      <div class="mt-6 text-center text-xs tracking-wide text-(--text-secondary) flex items-center justify-center gap-1.5">
        <ShieldCheck class="h-3.5 w-3.5 text-indigo-500" />
        <span>{{ t('authGate.seedNote') }}</span>
      </div>
    </div>
  </div>
</template>