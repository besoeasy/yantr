<script setup>
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Copy, Check, LoaderCircle, LockKeyhole, Eye, EyeOff, AlertTriangle,
  ShieldCheck, Sparkles, Download,
  ArrowRight, ArrowLeft
} from '@lucide/vue'
import { useNotification } from '../composables/useNotification'
import { useYantrAuth } from '../composables/useYantrAuth'
import { derivePrivateKey, getPublicKey } from '../utils/crypto.js'
import { copyText } from '../utils/clipboard.js'

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

const isSetup = computed(() => !authState.configured)

// ─── Strength indicator ──────────────────────────────────────────────────────
const strength = computed(() => {
  const pw = password.value
  const p = pin.value
  if (!pw || !p) {
    return { label: t('authGate.strengthEmpty'), score: 0, bar: '', textColor: 'yantra-text-secondary' }
  }
  let score = 0
  if (pw.length >= 8) score += 1
  if (pw.length >= 12) score += 1
  if (/[A-Z]/.test(pw) && /[0-9]/.test(pw)) score += 1
  if (/[^A-Za-z0-9]/.test(pw)) score += 1
  if (p.length >= 4) score += 1

  if (score <= 2) return { label: t('authGate.strengthWeak'), score: 1, bar: 'yantra-auth__bar--weak', textColor: 'yantra-text-danger' }
  if (score <= 3) return { label: t('authGate.strengthFair'), score: 2, bar: 'yantra-auth__bar--fair', textColor: 'yantra-text-warn' }
  if (score <= 4) return { label: t('authGate.strengthGood'), score: 3, bar: 'yantra-auth__bar--good', textColor: 'yantra-text-accent' }
  return { label: t('authGate.strengthStrong'), score: 4, bar: 'yantra-auth__bar--strong', textColor: 'yantra-text-ok' }
})

// ─── Random key generator ───────────────────────────────────────────────────
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
  toast.success(t('authGate.messages.generated'))
}

// ─── Live public key derivation ───────────────────────────────────────────────
const derivedPublicKey = ref('')
const derivingKey = ref(false)
const copied = ref(false)
let deriveTimer = null

watch([password, pin], ([pw, p]) => {
  derivedPublicKey.value = ''
  clearTimeout(deriveTimer)
  if (!isSetup.value || !pw || !p) return
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
  if (!(await copyText(derivedPublicKey.value))) return
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
  toast.success(t('authGate.messages.backupDownloaded'))
}

// ─── The graph on the brand half ─────────────────────────────────────────────
// The left half is the stack itself: nodes drift, edges fade in and out between
// neighbours, and a packet crosses an edge now and then. It is drawn in script,
// so its palette lives in main.css as channel triples and is read once — the
// panel is the same deep navy in both themes, so the palette never changes.
const brandCanvas = ref(null)
const host = ref('')

let frame = null
let ctx = null
let palette = null
let observer = null
let lastTick = 0
let spawnClock = 0
let width = 0
let height = 0

const nodes = []
let pulses = []

function channelColor(channels, alpha) {
  return `rgba(${channels.trim().split(/\s+/).join(', ')}, ${alpha})`
}

function readPalette() {
  const cs = getComputedStyle(document.documentElement)
  const chan = (name, fallback) => (cs.getPropertyValue(name) || fallback).trim()
  return {
    node: chan('--canvas-node', '125 211 252'),
    particle: chan('--canvas-particle', '148 163 184'),
    thread: chan('--canvas-thread', '56 189 248'),
    pulse: chan('--canvas-pulse', '186 230 253'),
  }
}

function seedNodes() {
  nodes.length = 0
  pulses = []
  const count = Math.round(Math.min(44, (width * height) / 9000))
  for (let i = 0; i < count; i++) {
    nodes.push({
      x: Math.random() * width,
      y: Math.random() * height,
      vx: (Math.random() - 0.5) * 0.18,
      vy: (Math.random() - 0.5) * 0.18,
      r: 1.1 + Math.random() * 1.5,
      service: Math.random() > 0.74,
    })
  }
}

function sizeGraph() {
  const canvas = brandCanvas.value
  const box = canvas?.parentElement?.getBoundingClientRect()
  if (!canvas || !box) return
  const dpr = Math.min(2, window.devicePixelRatio || 1)
  width = Math.max(1, Math.round(box.width))
  height = Math.max(1, Math.round(box.height))
  canvas.width = Math.round(width * dpr)
  canvas.height = Math.round(height * dpr)
  ctx = canvas.getContext('2d')
  if (ctx) ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  seedNodes()
  drawGraph()
}

function spawnPulse() {
  if (nodes.length < 2) return
  const a = nodes[Math.floor(Math.random() * nodes.length)]
  const near = nodes.filter((n) => n !== a && Math.hypot(n.x - a.x, n.y - a.y) < 130)
  if (!near.length) return
  pulses.push({ a, b: near[Math.floor(Math.random() * near.length)], t: 0, speed: 0.006 + Math.random() * 0.006 })
}

function drawGraph() {
  if (!ctx || !palette) return
  ctx.clearRect(0, 0, width, height)

  for (let i = 0; i < nodes.length; i++) {
    const a = nodes[i]
    for (let j = i + 1; j < nodes.length; j++) {
      const b = nodes[j]
      const d = Math.hypot(a.x - b.x, a.y - b.y)
      if (d >= 132) continue
      ctx.strokeStyle = channelColor(palette.thread, (0.16 * (1 - d / 132)).toFixed(3))
      ctx.lineWidth = 0.7
      ctx.beginPath()
      ctx.moveTo(a.x, a.y)
      ctx.lineTo(b.x, b.y)
      ctx.stroke()
    }
  }

  for (const p of pulses) {
    const x = p.a.x + (p.b.x - p.a.x) * p.t
    const y = p.a.y + (p.b.y - p.a.y) * p.t
    const glow = ctx.createRadialGradient(x, y, 0, x, y, 7)
    glow.addColorStop(0, channelColor(palette.pulse, 0.9))
    glow.addColorStop(1, channelColor(palette.pulse, 0))
    ctx.fillStyle = glow
    ctx.beginPath()
    ctx.arc(x, y, 7, 0, Math.PI * 2)
    ctx.fill()
  }

  for (const n of nodes) {
    const x = n.x
    const y = n.y
    if (n.service) {
      const s = 9
      ctx.strokeStyle = channelColor(palette.node, 0.42)
      ctx.lineWidth = 1
      ctx.beginPath()
      if (ctx.roundRect) ctx.roundRect(x - s / 2, y - s / 2, s, s, 2.5)
      else ctx.rect(x - s / 2, y - s / 2, s, s)
      ctx.stroke()
      ctx.fillStyle = channelColor(palette.node, 0.85)
      ctx.beginPath()
      ctx.arc(x, y, 1.6, 0, Math.PI * 2)
      ctx.fill()
    } else {
      ctx.fillStyle = channelColor(palette.particle, 0.42)
      ctx.beginPath()
      ctx.arc(x, y, n.r, 0, Math.PI * 2)
      ctx.fill()
    }
  }
}

function tick(now) {
  if (lastTick) {
    const step = Math.min(48, now - lastTick) / 16.67
    for (const n of nodes) {
      n.x += n.vx * step
      n.y += n.vy * step
      if (n.x < -20 || n.x > width + 20) n.vx *= -1
      if (n.y < -20 || n.y > height + 20) n.vy *= -1
    }
    for (const p of pulses) p.t += p.speed * step
    pulses = pulses.filter((p) => p.t < 1)
    spawnClock += now - lastTick
    if (spawnClock > 900) {
      spawnClock = 0
      spawnPulse()
    }
  }
  lastTick = now
  drawGraph()
  frame = requestAnimationFrame(tick)
}

// ─── Steps ───────────────────────────────────────────────────────────────────
const activeTab = ref(0)

const steps = [{ id: 0 }, { id: 1 }, { id: 2 }]

const stepHeading = computed(() => [
  t('authGate.setupHeading'),
  t('authGate.confirmHeading'),
  t('authGate.keyHeading'),
][activeTab.value])

const stepSubtitle = computed(() => [
  t('authGate.setupSubtitle'),
  t('authGate.confirmSubtitle'),
  t('authGate.keySubtitle'),
][activeTab.value])

function goToStep(stepIndex) {
  if (!isSetup.value) return
  if (stepIndex === 1) {
    if (!password.value || !pin.value) {
      localError.value = t('authGate.errors.fillBoth')
      return
    }
    if (password.value.length < 8) {
      localError.value = t('authGate.errors.passwordTooShort')
      return
    }
    localError.value = ''
    activeTab.value = 1
    nextTick(() => confirmPasswordInput.value?.focus())
  } else if (stepIndex === 2) {
    if (password.value !== passwordConfirm.value) {
      localError.value = t('authGate.errors.passwordMismatch')
      return
    }
    if (pin.value !== pinConfirm.value) {
      localError.value = t('authGate.errors.pinMismatch')
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
  if (!String(password.value).trim()) return t('authGate.errors.passwordRequired')
  if (!String(pin.value).trim()) return t('authGate.errors.pinRequired')
  if (isSetup.value) {
    if (password.value !== passwordConfirm.value) return t('authGate.errors.passwordMismatch')
    if (pin.value !== pinConfirm.value) return t('authGate.errors.pinMismatch')
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
      pin: pin.value,
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
  host.value = window.location.host || 'localhost'
  nextTick(() => passwordInput.value?.focus())
  requestAnimationFrame(() => {
    palette = readPalette()
    sizeGraph()
    const brand = brandCanvas.value?.parentElement
    if (brand && typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(() => sizeGraph())
      observer.observe(brand)
    }
    // A still graph is the reduced-motion version: the same picture, one frame
    // of it, because the field is decoration and nothing depends on its drift.
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) {
      drawGraph()
      return
    }
    frame = requestAnimationFrame(tick)
  })
})

onUnmounted(() => {
  if (frame) cancelAnimationFrame(frame)
  observer?.disconnect()
  clearTimeout(deriveTimer)
})
</script>

<template>
  <div class="yantra-auth">
    <!-- The host: a quiet network field behind a short product introduction. -->
    <section class="yantra-auth__brand">
      <canvas ref="brandCanvas" class="yantra-auth__canvas" aria-hidden="true" />

      <div class="yantra-auth__brand-inner">
        <div class="yantra-auth__wordmark">
          YANTR <span>{{ t('authGate.brandKicker') }}</span>
        </div>

        <div class="yantra-auth__copy">
          <h1 class="yantra-auth__headline">
            {{ t('authGate.brandTitle') }}<br />
            <em>{{ t('authGate.brandTitleAccent') }}</em>
          </h1>
          <p class="yantra-auth__lead">{{ t('authGate.brandLead') }}</p>
          <p class="yantra-auth__trust">
            <ShieldCheck class="yantra-auth__icon" />
            <span>{{ t('authGate.proofLocal') }}</span>
          </p>

          <div class="yantra-auth__status">
            <span class="yantra-auth__host">{{ host }}</span>
            <span class="yantra-auth__status-right">
              Ed25519 <span class="yantra-auth__sep">·</span> Podman
            </span>
          </div>
        </div>
      </div>
    </section>

    <!-- The door. -->
    <section class="yantra-auth__form">
      <div class="yantra-auth__panel">
        <div class="yantra-auth__mobilebrand">YANTR</div>

        <div v-if="authState.booting" class="anim-stagger">
          <span class="yantra-auth__eyebrow">{{ t('authGate.bootingTitle') }}</span>
          <div class="yantra-auth__booting">
            <LoaderCircle class="yantra-auth__spinner yantra-text-accent" />
            <span>{{ t('authGate.bootingState') }}</span>
          </div>
        </div>

        <form v-else @submit.prevent="handleFormSubmit">
          <!-- Sign in -->
          <div v-if="!isSetup" key="login" class="anim-stagger">
            <span class="yantra-auth__eyebrow">{{ t('authGate.loginEyebrow') }}</span>
            <h2 class="yantra-auth__title">{{ t('authGate.loginHeading') }}</h2>
            <p class="yantra-auth__sub">{{ t('authGate.loginSubtitle') }}</p>

            <div class="yantra-auth__field">
              <label class="yantra-auth__label" for="yantr-password">{{ t('authGate.password') }}</label>
              <div class="yantra-auth__control">
                <input
                  id="yantr-password"
                  ref="passwordInput"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                  :placeholder="t('authGate.passwordPlaceholder')"
                  class="yantra-input"
                />
                <button
                  type="button"
                  class="yantra-auth__eye"
                  :aria-label="showPassword ? t('authGate.hidePassword') : t('authGate.showPassword')"
                  :aria-pressed="showPassword"
                  @click="showPassword = !showPassword"
                >
                  <EyeOff v-if="showPassword" class="yantra-auth__icon" />
                  <Eye v-else class="yantra-auth__icon" />
                </button>
              </div>
            </div>

            <div class="yantra-auth__field">
              <div class="yantra-auth__label">
                <label for="yantr-pin">{{ t('authGate.pin') }}</label>
                <span class="yantra-auth__hint">{{ t('authGate.pinHint') }}</span>
              </div>
              <div class="yantra-auth__control">
                <input
                  id="yantr-pin"
                  v-model="pin"
                  :type="showPin ? 'text' : 'password'"
                  inputmode="numeric"
                  autocomplete="off"
                  :placeholder="t('authGate.pinPlaceholder')"
                  class="yantra-input"
                />
                <button
                  type="button"
                  class="yantra-auth__eye"
                  :aria-label="showPin ? t('authGate.hidePin') : t('authGate.showPin')"
                  :aria-pressed="showPin"
                  @click="showPin = !showPin"
                >
                  <EyeOff v-if="showPin" class="yantra-auth__icon" />
                  <Eye v-else class="yantra-auth__icon" />
                </button>
              </div>
            </div>

            <div
              v-if="localError || authState.error"
              class="yantra-chip--danger yantra-auth__error"
              role="alert"
            >
              <AlertTriangle class="yantra-auth__icon yantra-text-danger" />
              <span>{{ localError || authState.error }}</span>
            </div>

            <button
              type="submit"
              :disabled="submitting || !password || !pin"
              class="yantra-cta yantra-auth__submit"
            >
              <LoaderCircle v-if="submitting" class="yantra-auth__spinner" />
              <LockKeyhole v-else class="yantra-auth__icon" />
              <span>{{ submitting ? t('authGate.working') : t('authGate.loginAction') }}</span>
            </button>

          </div>

          <!-- First run: credentials → confirm → key -->
          <div v-else :key="`setup-${activeTab}`" class="anim-stagger">
            <span class="yantra-auth__eyebrow">
              {{ t('authGate.setupEyebrow') }} · {{ t('authGate.stepOf', { n: activeTab + 1 }) }}
            </span>
            <h2 class="yantra-auth__title">{{ stepHeading }}</h2>
            <p class="yantra-auth__sub">{{ stepSubtitle }}</p>

            <div class="yantra-auth__steps" role="group" :aria-label="t('authGate.setupEyebrow')">
              <button
                v-for="step in steps"
                :key="step.id"
                type="button"
                class="yantra-auth__step"
                :class="{ 'yantra-auth__step--done': activeTab > step.id, 'yantra-auth__step--now': activeTab === step.id }"
                :aria-label="t('authGate.stepOf', { n: step.id + 1 })"
                :aria-current="activeTab === step.id ? 'step' : undefined"
                @click="goToStep(step.id)"
              />
            </div>

            <!-- Step 1 · credentials -->
            <template v-if="activeTab === 0">
              <div class="yantra-auth__field">
                <div class="yantra-auth__label">
                  <label for="yantr-password">{{ t('authGate.password') }}</label>
                  <span
                    v-if="password"
                    class="yantra-auth__hint"
                    :class="strength.textColor"
                  >
                    {{ strength.label }}
                  </span>
                </div>
                <div class="yantra-auth__control">
                  <input
                    id="yantr-password"
                    ref="passwordInput"
                    v-model="password"
                    :type="showPassword ? 'text' : 'password'"
                    autocomplete="new-password"
                    :placeholder="t('authGate.passwordPlaceholder')"
                    class="yantra-input"
                  />
                  <button
                    type="button"
                    class="yantra-auth__eye"
                    :aria-label="showPassword ? t('authGate.hidePassword') : t('authGate.showPassword')"
                    :aria-pressed="showPassword"
                    @click="showPassword = !showPassword"
                  >
                    <EyeOff v-if="showPassword" class="yantra-auth__icon" />
                    <Eye v-else class="yantra-auth__icon" />
                  </button>
                </div>
              </div>

              <div class="yantra-auth__field">
                <div class="yantra-auth__label">
                  <label for="yantr-pin">{{ t('authGate.pin') }}</label>
                  <span class="yantra-auth__hint">{{ t('authGate.pinHint') }}</span>
                </div>
                <div class="yantra-auth__control">
                  <input
                    id="yantr-pin"
                    v-model="pin"
                    :type="showPin ? 'text' : 'password'"
                    inputmode="numeric"
                    autocomplete="off"
                    :placeholder="t('authGate.pinPlaceholder')"
                    class="yantra-input"
                  />
                  <button
                    type="button"
                    class="yantra-auth__eye"
                    :aria-label="showPin ? t('authGate.hidePin') : t('authGate.showPin')"
                    :aria-pressed="showPin"
                    @click="showPin = !showPin"
                  >
                    <EyeOff v-if="showPin" class="yantra-auth__icon" />
                    <Eye v-else class="yantra-auth__icon" />
                  </button>
                </div>
              </div>

              <div v-if="password || pin" class="yantra-surface-muted yantra-auth__strength">
                <div class="yantra-auth__strength-head">
                  <span>{{ t('authGate.strengthTitle') }}</span>
                  <span :class="strength.textColor">{{ strength.label }}</span>
                  <span class="yantra-auth__bars" aria-hidden="true">
                    <span
                      v-for="i in 4"
                      :key="i"
                      class="yantra-auth__bar"
                      :class="i <= strength.score ? strength.bar : ''"
                    />
                  </span>
                </div>
              </div>

              <div
                v-if="localError || authState.error"
                class="yantra-chip--danger yantra-auth__error"
                role="alert"
              >
                <AlertTriangle class="yantra-auth__icon yantra-text-danger" />
                <span>{{ localError || authState.error }}</span>
              </div>

              <button
                type="button"
                class="yantra-auth__ghost"
                @click="generateRandomSecret"
              >
                <Sparkles class="yantra-auth__icon" />
                <span>{{ t('authGate.autoGenerate') }}</span>
              </button>

              <button type="submit" class="yantra-cta yantra-auth__submit">
                <span>{{ t('authGate.continueAction') }}</span>
                <ArrowRight class="yantra-auth__icon" />
              </button>
            </template>

            <!-- Step 2 · confirm -->
            <template v-else-if="activeTab === 1">
              <div class="yantra-chip--warn yantra-auth__error">
                <AlertTriangle class="yantra-auth__icon yantra-text-warn" />
                <span>{{ t('authGate.confirmWarning') }}</span>
              </div>

              <div class="yantra-auth__field">
                <div class="yantra-auth__label">
                  <label for="yantr-password-confirm">{{ t('authGate.passwordConfirm') }}</label>
                  <span
                    v-if="passwordConfirm && password === passwordConfirm"
                    class="yantra-auth__hint yantra-auth__hint--ok"
                  >
                    {{ t('authGate.matches') }}
                  </span>
                </div>
                <input
                  id="yantr-password-confirm"
                  ref="confirmPasswordInput"
                  v-model="passwordConfirm"
                  type="password"
                  autocomplete="off"
                  :placeholder="t('authGate.passwordConfirmPlaceholder')"
                  class="yantra-input yantra-auth__bare"
                />
              </div>

              <div class="yantra-auth__field">
                <div class="yantra-auth__label">
                  <label for="yantr-pin-confirm">{{ t('authGate.pinConfirm') }}</label>
                  <span
                    v-if="pinConfirm && pin === pinConfirm"
                    class="yantra-auth__hint yantra-auth__hint--ok"
                  >
                    {{ t('authGate.matches') }}
                  </span>
                </div>
                <input
                  id="yantr-pin-confirm"
                  v-model="pinConfirm"
                  type="password"
                  inputmode="numeric"
                  autocomplete="off"
                  :placeholder="t('authGate.pinConfirmPlaceholder')"
                  class="yantra-input yantra-auth__bare"
                />
              </div>

              <div
                v-if="localError"
                class="yantra-chip--danger yantra-auth__error"
                role="alert"
              >
                <AlertTriangle class="yantra-auth__icon yantra-text-danger" />
                <span>{{ localError }}</span>
              </div>

              <div class="yantra-auth__actions">
                <button type="button" class="yantra-btn yantra-auth__back" @click="goToStep(0)">
                  <ArrowLeft class="yantra-auth__icon" />
                  <span>{{ t('authGate.backAction') }}</span>
                </button>
                <button
                  type="submit"
                  :disabled="!passwordConfirm || !pinConfirm || password !== passwordConfirm || pin !== pinConfirm"
                  class="yantra-cta yantra-auth__submit"
                >
                  <span>{{ t('authGate.reviewAction') }}</span>
                  <ArrowRight class="yantra-auth__icon" />
                </button>
              </div>
            </template>

            <!-- Step 3 · the key -->
            <template v-else>
              <div class="yantra-auth__key">
                <span
                  class="yantra-auth__fingerprint"
                  :class="{ 'yantra-auth__fingerprint--live': derivingKey }"
                  :style="{ background: keyAvatarGradient }"
                />
                <span class="yantra-auth__keytext">
                  <span class="yantra-auth__keylabel">{{ t('authGate.generatedKey') }}</span>
                  <span class="yantra-auth__keysub">
                    {{
                      derivedPublicKey
                        ? t('authGate.fingerprint', { key: derivedPublicKey.slice(0, 8) })
                        : t('authGate.awaitingCredentials')
                    }}
                  </span>
                </span>
                <span v-if="derivedPublicKey" class="yantra-auth__keyactions">
                  <button
                    type="button"
                    class="yantra-btn-icon yantra-auth__keybtn"
                    :title="t('authGate.copyKey')"
                    @click="copyPublicKey"
                  >
                    <Check v-if="copied" class="yantra-auth__icon yantra-text-ok" />
                    <Copy v-else class="yantra-auth__icon" />
                  </button>
                  <button
                    type="button"
                    class="yantra-btn-icon yantra-auth__keybtn"
                    :title="t('authGate.downloadKey')"
                    @click="downloadKeyBackup"
                  >
                    <Download class="yantra-auth__icon" />
                  </button>
                </span>
                <span v-else class="yantra-auth__keytag">Ed25519</span>
              </div>

              <div v-if="derivingKey" class="yantra-auth__booting">
                <LoaderCircle class="yantra-auth__spinner yantra-text-accent" />
                <span>{{ t('authGate.computingKey') }}</span>
              </div>
              <div v-else-if="derivedPublicKey" class="yantra-keywell yantra-auth__well">
                {{ formattedPublicKey }}
              </div>
              <p v-else class="yantra-auth__keynote">{{ t('authGate.keyPending') }}</p>

              <p v-if="derivedPublicKey" class="yantra-auth__keynote">
                <strong>{{ t('authGate.backupLead') }}</strong>
                <code class="yantra-inline-code">{{ t('authGate.backupVariable') }}</code>
                {{ t('authGate.backupTail') }}
              </p>

              <div
                v-if="localError || authState.error"
                class="yantra-chip--danger yantra-auth__error"
                role="alert"
              >
                <AlertTriangle class="yantra-auth__icon yantra-text-danger" />
                <span>{{ localError || authState.error }}</span>
              </div>

              <p class="yantra-auth__foot">
                <AlertTriangle class="yantra-auth__icon" />
                <span>{{ t('authGate.noRecovery') }}</span>
              </p>

              <div class="yantra-auth__actions">
                <button type="button" class="yantra-btn yantra-auth__back" @click="goToStep(1)">
                  <ArrowLeft class="yantra-auth__icon" />
                  <span>{{ t('authGate.backAction') }}</span>
                </button>
                <button
                  type="submit"
                  :disabled="submitting || !derivedPublicKey"
                  class="yantra-cta yantra-auth__submit"
                >
                  <LoaderCircle v-if="submitting" class="yantra-auth__spinner" />
                  <Sparkles v-else class="yantra-auth__icon" />
                  <span>{{ submitting ? t('authGate.working') : t('authGate.setupAction') }}</span>
                </button>
              </div>
            </template>
          </div>
        </form>
      </div>
    </section>
  </div>
</template>
