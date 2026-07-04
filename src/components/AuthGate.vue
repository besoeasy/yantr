<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { LoaderCircle, LockKeyhole, UserRound } from 'lucide-vue-next'
import { useNotification } from '../composables/useNotification'
import { useYantrAuth } from '../composables/useYantrAuth'

const { t } = useI18n()
const toast = useNotification()
const { authState, loginYantr, setupYantrAdmin } = useYantrAuth()

const username = ref(localStorage.getItem('yantr-username') || '')
const password = ref('')
const pin = ref('')
const submitting = ref(false)
const localError = ref('')

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

function validate() {
  if (!String(username.value).trim()) return t('authGate.errors.usernameRequired')
  if (!String(password.value).trim()) return 'Password is required'
  if (!String(pin.value).trim()) return 'PIN is required'
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
      username: username.value, 
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

onMounted(() => { requestAnimationFrame(() => { initBackground() }) })
onUnmounted(() => {
  if (raf) cancelAnimationFrame(raf)
  window.removeEventListener('mousemove', handleMouse)
  window.removeEventListener('mouseleave', handleMouseLeave)
})
</script>

<template>
  <div class="min-h-screen bg-(--bg-body) text-(--text-primary) relative overflow-hidden">
    <!-- Premium interactive mouse-reactive background -->
    <canvas
      ref="bgCanvas"
      class="fixed inset-0 z-0 pointer-events-none"
      style="opacity: 0.85;"
    />

    <!-- Centered content -->
    <div class="relative z-10 min-h-screen flex items-center justify-center px-5 py-12">
      <div class="w-full max-w-[360px]">
        <!-- Minimal header -->
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

        <!-- Form card -->
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

          <form v-else @submit.prevent="submit" class="space-y-5">
            <!-- Username -->
            <div>
              <div class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                <UserRound class="h-3.5 w-3.5" />
                <span>{{ t('authGate.username') }}</span>
              </div>
              <input
                v-model="username"
                type="text"
                autocomplete="username"
                :placeholder="t('authGate.usernamePlaceholder')"
                class="w-full bg-(--surface-muted) rounded-2xl px-4 py-3.5 text-[15px] font-medium placeholder:text-(--text-secondary)/60 focus:outline-none focus:-translate-y-px transition-all duration-200"
                style="box-shadow: 0 1px 2px rgba(0,0,0,0.03);"
              />
            </div>

            <!-- Password -->
            <div>
              <div class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                <LockKeyhole class="h-3.5 w-3.5" />
                <span>PASSWORD</span>
              </div>
              <input
                v-model="password"
                type="password"
                autocomplete="current-password"
                placeholder="Enter your password"
                class="w-full bg-(--surface-muted) rounded-2xl px-4 py-3.5 text-[15px] font-medium placeholder:text-(--text-secondary)/60 focus:outline-none focus:-translate-y-px transition-all duration-200"
                style="box-shadow: 0 1px 2px rgba(0,0,0,0.03);"
              />
            </div>

            <!-- PIN -->
            <div>
              <div class="text-[10px] font-semibold tracking-[1.5px] text-(--text-secondary) mb-1.5 flex items-center gap-1.5">
                <LockKeyhole class="h-3.5 w-3.5" />
                <span>PIN</span>
              </div>
              <input
                v-model="pin"
                type="password"
                inputmode="numeric"
                autocomplete="off"
                placeholder="Enter your PIN"
                class="w-full bg-(--surface-muted) rounded-2xl px-4 py-3.5 text-[15px] font-medium placeholder:text-(--text-secondary)/60 focus:outline-none focus:-translate-y-px transition-all duration-200"
                style="box-shadow: 0 1px 2px rgba(0,0,0,0.03);"
              />
            </div>

            <!-- Setup note -->
            <p v-if="isSetup" class="text-[12px] text-(--text-secondary) leading-relaxed">
              Set a strong password and a PIN to generate your secure session key.
            </p>

            <!-- Login note: device-bound key -->
            <p v-else class="text-[12px] text-(--text-secondary) leading-relaxed">
              Enter your password and PIN to derive your session key and unlock Yantr.
            </p>

            <!-- Error -->
            <p
              v-if="localError || authState.error"
              class="text-xs font-medium text-red-600 dark:text-red-400 bg-red-500/5 px-3.5 py-2 rounded-xl"
            >
              {{ localError || authState.error }}
            </p>

            <!-- Submit -->
            <button
              type="submit"
              :disabled="submitting"
              class="group w-full mt-1 flex h-12 items-center justify-center gap-2 rounded-2xl bg-(--text-primary) text-(--bg-body) text-[13px] font-semibold tracking-[0.5px] active:scale-[0.985] disabled:opacity-60 transition-all duration-150 hover:shadow-xl disabled:cursor-not-allowed"
            >
              <LoaderCircle v-if="submitting" class="h-3.5 w-3.5 animate-spin" />
              <LockKeyhole v-else class="h-3.5 w-3.5 transition-transform group-hover:-translate-y-[0.5px]" />
              <span>{{ submitting ? t('authGate.working') : isSetup ? t('authGate.setupAction') : t('authGate.loginAction') }}</span>
            </button>
          </form>
        </div>

        <div class="mt-6 text-center text-[10px] tracking-[0.5px] text-(--text-secondary)">
          {{ t('authGate.seedNote') }}
        </div>
      </div>
    </div>
  </div>
</template>