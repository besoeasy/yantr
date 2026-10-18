<script setup>
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  Copy, Check, LoaderCircle, LockKeyhole, Eye, EyeOff, AlertTriangle,
  ShieldCheck, Sparkles, KeyRound, Download, CheckCircle2,
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
  if (!pw || !p) return { label: 'Empty', score: 0, color: 'bg-(--card-inset)', textColor: 'yantra-text-secondary' }
  let score = 0
  if (pw.length >= 8) score += 1
  if (pw.length >= 12) score += 1
  if (/[A-Z]/.test(pw) && /[0-9]/.test(pw)) score += 1
  if (/[^A-Za-z0-9]/.test(pw)) score += 1
  if (p.length >= 4) score += 1
  
  if (score <= 2) return { label: 'Weak', score: 1, color: 'bg-rose-500', textColor: 'yantra-text-danger' }
  if (score <= 3) return { label: 'Fair', score: 2, color: 'bg-amber-500', textColor: 'yantra-text-warn' }
  if (score <= 4) return { label: 'Good', score: 3, color: 'bg-blue-500', textColor: 'yantra-text-accent' }
  return { label: 'Strong', score: 4, color: 'bg-emerald-500', textColor: 'yantra-text-ok' }
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
    return 'linear-gradient(135deg, #0ea5e9 0%, #7c3aed 100%)'
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

ENVIRONMENT VARIABLE:
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

// ─── Mouse Follow Canvas Animation (Pure White Theme) ─────────────────────────
const bgCanvas = ref(null)
let canvasCtx = null
let particles = []
const mouse = { x: -500, y: -500, targetX: -500, targetY: -500, active: false }
let raf = null

// The particle field is drawn in script, so it cannot use var(--accent) the way
// the rest of the app does: canvas needs concrete channels to build rgba()
// from, and --accent resolves to oklch(). The channel triples live in main.css
// next to the rest of the palette, and are read on every resize/paint so the
// field follows the theme instead of freezing in whichever mode booted first.
function themeChannels() {
  const cs = getComputedStyle(document.documentElement)
  const chan = (name) => (cs.getPropertyValue(name) || '2 132 199').trim()
  return {
    particle: chan('--canvas-particle'),
    particle2: chan('--canvas-particle-2'),
    thread: chan('--canvas-thread'),
    page: (cs.getPropertyValue('--canvas-page') || '#ffffff').trim(),
  }
}

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

  const theme = themeChannels()
  particles = []
  const count = Math.min(100, Math.floor((canvas.width * canvas.height) / 15000))
  for (let i = 0; i < count; i++) {
    particles.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: 1.5 + Math.random() * 2.5,
      baseX: Math.random() * canvas.width,
      baseY: Math.random() * canvas.height,
      vx: (Math.random() - 0.5) * 0.4,
      vy: (Math.random() - 0.5) * 0.4,
      color: Math.random() > 0.5
        ? `rgba(${theme.particle}, 0.25)`
        : `rgba(${theme.particle2}, 0.2)`
    })
  }

  window.addEventListener('mousemove', handleMouse)
  window.addEventListener('mouseleave', handleMouseLeave)
  animateBackground()
}

function handleMouse(e) {
  mouse.x = e.clientX
  mouse.y = e.clientY
  if (!mouse.active) {
    mouse.targetX = e.clientX
    mouse.targetY = e.clientY
  }
  mouse.active = true
}

function handleMouseLeave() {
  mouse.active = false
}

function animateBackground() {
  const canvas = bgCanvas.value
  const ctx = canvasCtx
  if (!ctx || !canvas) return

  // Smooth lerp mouse coordinates for organic follow feel
  if (mouse.active) {
    mouse.targetX += (mouse.x - mouse.targetX) * 0.1
    mouse.targetY += (mouse.y - mouse.targetY) * 0.1
  }

  // Clear to the page colour: a white clear is what made the field flash white
  // behind the card on a dark install.
  ctx.fillStyle = themeChannels().page
  ctx.fillRect(0, 0, canvas.width, canvas.height)

  // 1. Mouse Spotlight Gradient Glow (Interactive white theme aura)
  if (mouse.active) {
    const radius = 450
    const grad = ctx.createRadialGradient(
      mouse.targetX, mouse.targetY, 0,
      mouse.targetX, mouse.targetY, radius
    )
    const t = themeChannels()
    grad.addColorStop(0, `rgba(${t.thread}, 0.09)`)
    grad.addColorStop(0.4, `rgba(${t.particle2}, 0.04)`)
    grad.addColorStop(1, `rgba(${t.page}, 0)`)
    ctx.fillStyle = grad
    ctx.beginPath()
    ctx.arc(mouse.targetX, mouse.targetY, radius, 0, Math.PI * 2)
    ctx.fill()
  }

  // 2. Interactive Particles & Follow filaments
  const mx = mouse.targetX
  const my = mouse.targetY
  const influenceRadius = 140

  for (let i = 0; i < particles.length; i++) {
    const p = particles[i]

    // Particle motion
    p.x += p.vx
    p.y += p.vy

    if (p.x < 0 || p.x > canvas.width) p.vx *= -1
    if (p.y < 0 || p.y > canvas.height) p.vy *= -1

    // Magnetic interaction with cursor follow
    if (mouse.active) {
      const dx = mx - p.x
      const dy = my - p.y
      const dist = Math.sqrt(dx * dx + dy * dy) || 1
      if (dist < influenceRadius) {
        const force = (influenceRadius - dist) / influenceRadius
        p.x -= (dx / dist) * force * 1.5
        p.y -= (dy / dist) * force * 1.5

        // Draw connecting light line to mouse
        ctx.strokeStyle = `rgba(${themeChannels().thread}, ${0.25 * force})`
        ctx.lineWidth = 0.8
        ctx.beginPath()
        ctx.moveTo(p.x, p.y)
        ctx.lineTo(mx, my)
        ctx.stroke()
      }
    }

    // Draw particle node
    ctx.fillStyle = p.color
    ctx.beginPath()
    ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2)
    ctx.fill()
  }

  // 3. Smooth Glowing Cursor Ring Follower
  if (mouse.active) {
    const t = themeChannels()
    ctx.lineWidth = 1.5
    ctx.strokeStyle = `rgba(${t.thread}, 0.35)`
    ctx.beginPath()
    ctx.arc(mouse.targetX, mouse.targetY, 18, 0, Math.PI * 2)
    ctx.stroke()

    ctx.fillStyle = `rgba(${t.thread}, 0.6)`
    ctx.beginPath()
    ctx.arc(mouse.targetX, mouse.targetY, 3, 0, Math.PI * 2)
    ctx.fill()
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
  <div class="min-h-screen relative overflow-hidden flex items-center justify-center p-4">
    <!-- Interactive Mouse Follow Canvas -->
    <canvas
      ref="bgCanvas"
      class="fixed inset-0 z-0 pointer-events-none"
    />

    <div class="anim-enter relative z-10 w-full max-w-[440px]">
      <!-- Brand Header -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full yantra-chip text-xs font-bold tracking-[2.5px] uppercase select-none shadow-xs">
          <Shield class="h-3.5 w-3.5 yantra-text-accent" />
          <span>YANTR IDENTITY</span>
        </div>
        <h1 class="mt-4 text-3xl sm:text-4xl font-black tracking-tight yantra-text-primary">
          {{ title }}
        </h1>
        <p class="mt-2 text-xs sm:text-sm yantra-text-secondary max-w-sm mx-auto font-medium">
          {{ isSetup ? 'Derive your stateless Ed25519 cryptographic key pair.' : t('authGate.loginSubtitle') }}
        </p>
      </div>

      <!-- Main Card: Pure White with Soft Elevation & No Harsh Borders -->
      <div
        class="yantra-surface authgate-card relative rounded-3xl p-7 sm:p-8 transition-[border-color,box-shadow,transform,opacity] duration-200"
        :class="{ 'opacity-60 pointer-events-none': authState.booting }"
      >
        <!-- Booting Spinner -->
        <div v-if="authState.booting" class="flex items-center justify-center min-h-[180px]">
          <div class="flex items-center gap-3 text-sm font-semibold yantra-text-secondary">
            <LoaderCircle class="h-5 w-5 animate-spin yantra-text-accent" />
            <span>{{ t('authGate.bootingState') }}</span>
          </div>
        </div>

        <form v-else @submit.prevent="handleFormSubmit" class="space-y-6">
          <!-- Step Wizard Nav (Setup Mode) -->
          <div v-if="isSetup" class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold uppercase tracking-wider yantra-text-secondary">
                Step {{ activeTab + 1 }} of 3: {{ steps[activeTab].title }}
              </span>
              <button 
                type="button" 
                @click="generateRandomSecret" 
                class="inline-flex items-center gap-1.5 text-xs font-bold yantra-text-accent hover:yantra-text-accent transition-colors"
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
                class="h-2 rounded-full transition-[border-color,box-shadow,transform,opacity] duration-200"
                :class="activeTab >= step.id ? 'yantra-cta' : 'yantra-chip'"
              />
            </div>
          </div>

          <!-- STEP 0: Credentials (Setup or Login) -->
          <template v-if="!isSetup || activeTab === 0">
            <!-- Password Input -->
            <div>
              <div class="text-xs font-bold tracking-wider uppercase yantra-text-secondary mb-2 flex items-center justify-between">
                <span class="flex items-center gap-1.5">
                  <LockKeyhole class="h-3.5 w-3.5 yantra-text-accent" />
                  <span>PASSWORD</span>
                </span>
                <span v-if="isSetup && password" class="text-xs font-bold" :class="strength.textColor">
                  {{ strength.label }}
                </span>
              </div>
              <div class="relative">
                <input
                  ref="passwordInput"
                  v-model="password"
                  :aria-label="t('authGate.password')"
                  :type="showPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                  placeholder="Enter secure master password"
                  class="w-full yantra-input yantra-text-primary rounded-2xl px-4 py-3.5 pr-11 text-sm font-semibold placeholder:yantra-text-secondary focus:outline-none focus:ring-2  transition-all"
                />
                <button
                  type="button"
                  @click="showPassword = !showPassword"
                  class="absolute right-3.5 top-1/2 -translate-y-1/2 yantra-text-secondary hover:yantra-text-primary transition-colors p-1"
                  tabindex="-1"
                >
                  <Eye v-if="showPassword" class="h-4 w-4" />
                  <EyeOff v-else class="h-4 w-4" />
                </button>
              </div>
            </div>

            <!-- PIN Input -->
            <div>
              <div class="text-xs font-bold tracking-wider uppercase yantra-text-secondary mb-2 flex items-center gap-1.5">
                <LockKeyhole class="h-3.5 w-3.5 yantra-text-accent" />
                <span>PIN CODE</span>
              </div>
              <div class="relative">
                <input
                  v-model="pin"
                  :aria-label="t('authGate.pin')"
                  :type="showPin ? 'text' : 'password'"
                  inputmode="numeric"
                  autocomplete="off"
                  placeholder="Enter numeric PIN (e.g. 1234)"
                  class="w-full yantra-input yantra-text-primary rounded-2xl px-4 py-3.5 pr-11 text-sm font-semibold placeholder:yantra-text-secondary focus:outline-none focus:ring-2  transition-all"
                />
                <button
                  type="button"
                  @click="showPin = !showPin"
                  class="absolute right-3.5 top-1/2 -translate-y-1/2 yantra-text-secondary hover:yantra-text-primary transition-colors p-1"
                  tabindex="-1"
                >
                  <Eye v-if="showPin" class="h-4 w-4" />
                  <EyeOff v-else class="h-4 w-4" />
                </button>
              </div>
            </div>

            <!-- Strength Criteria checklist (Setup mode) -->
            <div v-if="isSetup && (password || pin)" class="p-4 rounded-2xl yantra-surface-muted">
              <div class="flex items-center justify-between text-xs font-bold mb-2">
                <span class="yantra-text-secondary">Credential Strength</span>
                <div class="flex gap-1.5">
                  <div 
                    v-for="i in 4" 
                    :key="i"
                    class="h-1.5 w-5 rounded-full transition-colors"
                    :class="i <= strength.score ? strength.color : 'bg-(--card-inset)'"
                  />
                </div>
              </div>
              <div class="grid grid-cols-2 gap-2 text-xs">
                <div 
                  v-for="(item, idx) in strengthCriteria" 
                  :key="idx" 
                  class="flex items-center gap-1.5"
                  :class="item.met ? 'yantra-text-ok font-semibold' : 'yantra-text-secondary'"
                >
                  <CheckCircle2 v-if="item.met" class="h-3.5 w-3.5 shrink-0" />
                  <div v-else class="h-3.5 w-3.5 rounded-full bg-(--card-inset) shrink-0" />
                  <span>{{ item.label }}</span>
                </div>
              </div>
            </div>

            <!-- Continue Button (Setup Mode Step 0) -->
            <button
              v-if="isSetup"
              type="button"
              @click="goToStep(1)"
              class="w-full flex h-12 items-center justify-center gap-2 rounded-2xl yantra-cta active:scale-[0.985] text-sm font-bold tracking-wide transition-all"
            >
              <span>Continue to Confirmation</span>
              <ArrowRight class="h-4 w-4" />
            </button>
          </template>

          <!-- STEP 1: Confirmation (Setup Mode Only) -->
          <template v-if="isSetup && activeTab === 1">
            <div class="yantra-chip--warn p-4 rounded-2xl text-xs font-medium flex items-start gap-2.5">
              <AlertTriangle class="h-4 w-4 shrink-0 mt-0.5 yantra-text-warn" />
              <span>Confirm both credentials carefully. Your Ed25519 identity key is derived directly from them.</span>
            </div>

            <!-- Confirm Password -->
            <div>
              <div class="text-xs font-bold tracking-wider uppercase yantra-text-secondary mb-2 flex items-center justify-between">
                <span class="flex items-center gap-1.5">
                  <ShieldCheck class="h-3.5 w-3.5 yantra-text-accent" />
                  <span>CONFIRM PASSWORD</span>
                </span>
                <span v-if="passwordConfirm && password === passwordConfirm" class="text-xs yantra-text-ok font-bold flex items-center gap-1">
                  <Check class="h-3.5 w-3.5" /> Matches
                </span>
              </div>
              <input
                ref="confirmPasswordInput"
                v-model="passwordConfirm"
                :aria-label="t('authGate.passwordConfirm')"
                type="password"
                autocomplete="off"
                placeholder="Re-enter master password"
                class="w-full yantra-input yantra-text-primary rounded-2xl px-4 py-3.5 text-sm font-semibold placeholder:yantra-text-secondary focus:outline-none focus:ring-2  transition-all"
              />
            </div>

            <!-- Confirm PIN -->
            <div>
              <div class="text-xs font-bold tracking-wider uppercase yantra-text-secondary mb-2 flex items-center justify-between">
                <span class="flex items-center gap-1.5">
                  <ShieldCheck class="h-3.5 w-3.5 yantra-text-accent" />
                  <span>CONFIRM PIN</span>
                </span>
                <span v-if="pinConfirm && pin === pinConfirm" class="text-xs yantra-text-ok font-bold flex items-center gap-1">
                  <Check class="h-3.5 w-3.5" /> Matches
                </span>
              </div>
              <input
                v-model="pinConfirm"
                :aria-label="t('authGate.pinConfirm')"
                type="password"
                inputmode="numeric"
                autocomplete="off"
                placeholder="Re-enter PIN"
                class="w-full yantra-input yantra-text-primary rounded-2xl px-4 py-3.5 text-sm font-semibold placeholder:yantra-text-secondary focus:outline-none focus:ring-2  transition-all"
              />
            </div>

            <!-- Actions Step 1 -->
            <div class="flex gap-2.5 pt-1">
              <button
                type="button"
                @click="goToStep(0)"
                class="flex-1 h-12 flex items-center justify-center gap-1.5 yantra-btn--lg text-sm font-bold transition-all"
              >
                <ArrowLeft class="h-4 w-4" />
                <span>Back</span>
              </button>
              <button
                type="button"
                @click="goToStep(2)"
                :disabled="!passwordConfirm || !pinConfirm || password !== passwordConfirm || pin !== pinConfirm"
                class="flex-1 h-12 flex items-center justify-center gap-2 rounded-2xl yantra-cta disabled:opacity-40 text-sm font-bold transition-all disabled:cursor-not-allowed"
              >
                <span>Review Key</span>
                <ArrowRight class="h-4 w-4" />
              </button>
            </div>
          </template>

          <!-- STEP 2: Identity Key Review (Setup) / Key Badge (Login) -->
          <template v-if="(!isSetup) || (isSetup && activeTab === 2)">
            <!-- Key Card Component -->
            <div class="rounded-2xl yantra-surface-muted space-y-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <div 
                    class="h-9 w-9 rounded-xl shadow-xs transition-[border-color,box-shadow,transform,opacity] duration-200" 
                    :style="{ background: keyAvatarGradient }"
                  />
                  <div>
                    <div class="text-xs font-extrabold tracking-wide uppercase yantra-text-primary">
                      {{ isSetup ? 'Generated Ed25519 Public Key' : 'Session Public Key' }}
                    </div>
                    <div class="text-[11px] yantra-text-secondary font-medium">
                      {{ derivedPublicKey ? `Identity Fingerprint: ${derivedPublicKey.slice(0, 8)}...` : 'Enter credentials above' }}
                    </div>
                  </div>
                </div>

                <div v-if="derivedPublicKey" class="flex gap-1.5">
                  <button
                    type="button"
                    @click="copyPublicKey"
                    class="p-2 rounded-xl yantra-btn-icon hover:yantra-text-primary transition-colors"
                    title="Copy Public Key"
                  >
                    <Check v-if="copied" class="h-4 w-4 yantra-text-ok" />
                    <Copy v-else class="h-4 w-4" />
                  </button>
                  <button
                    v-if="isSetup"
                    type="button"
                    @click="downloadKeyBackup"
                    class="p-2 rounded-xl yantra-btn-icon hover:yantra-text-primary transition-colors"
                    title="Download Key Backup"
                  >
                    <Download class="h-4 w-4" />
                  </button>
                </div>
              </div>

              <!-- Key Code Box -->
              <div v-if="derivingKey" class="flex items-center justify-center py-4 text-xs font-medium yantra-text-secondary gap-2">
                <LoaderCircle class="h-4 w-4 animate-spin yantra-text-accent" />
                <span>Computing cryptographic Ed25519 key...</span>
              </div>
              <div v-else-if="derivedPublicKey" class="yantra-keywell rounded-2xl p-3.5 font-mono text-[11px] break-all leading-relaxed">
                {{ formattedPublicKey }}
              </div>
              <div v-else class="text-xs yantra-text-secondary italic py-2 text-center">
                Key will be derived when Password and PIN are entered.
              </div>

              <!-- Backup Notice -->
              <p v-if="isSetup && derivedPublicKey" class="text-[11px] yantra-text-secondary leading-snug font-medium">
                💡 <span class="font-bold yantra-text-primary">Save this key:</span> Pass it as <code class="yantra-inline-code px-1.5 py-0.5 rounded yantra-text-accent font-mono text-[10px] font-bold">YANTR_ADMIN_PUBLIC_KEY</code> to enforce server-side identity verification.
              </p>
            </div>

            <!-- Error Banner -->
            <div
              v-if="localError || authState.error"
              class="yantra-chip--danger p-4 rounded-2xl text-xs font-semibold flex items-center gap-2.5"
            >
              <AlertTriangle class="h-4 w-4 shrink-0 yantra-text-danger" />
              <span>{{ localError || authState.error }}</span>
            </div>

            <!-- Final Submit Actions -->
            <div v-if="isSetup" class="flex gap-2.5 pt-1">
              <button
                type="button"
                @click="goToStep(1)"
                class="flex-1 h-12 flex items-center justify-center gap-1.5 yantra-btn--lg text-sm font-bold transition-all"
              >
                <ArrowLeft class="h-4 w-4" />
                <span>Back</span>
              </button>
              <button
                type="submit"
                :disabled="submitting || !derivedPublicKey"
                class="flex-1 h-12 flex items-center justify-center gap-2 rounded-2xl yantra-cta active:scale-[0.985] disabled:opacity-60 text-sm font-bold tracking-wide transition-all disabled:cursor-not-allowed"
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
              class="group w-full flex h-12 items-center justify-center gap-2 rounded-2xl yantra-cta active:scale-[0.985] disabled:opacity-60 text-sm font-bold tracking-wide transition-all disabled:cursor-not-allowed"
            >
              <LoaderCircle v-if="submitting" class="h-4 w-4 animate-spin" />
              <LockKeyhole v-else class="h-4 w-4 transition-transform group-hover:-translate-y-[3px]" />
              <span>{{ submitting ? t('authGate.working') : t('authGate.loginAction') }}</span>
            </button>
          </template>
        </form>
      </div>

      <!-- Footer Note -->
      <div class="mt-6 text-center text-xs tracking-wide font-medium yantra-text-secondary flex items-center justify-center gap-1.5">
        <ShieldCheck class="h-3.5 w-3.5 yantra-text-accent" />
        <span>{{ t('authGate.seedNote') }}</span>
      </div>
    </div>
  </div>
</template>