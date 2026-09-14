<script setup>
import { useI18n } from 'vue-i18n'
import { CheckCircle2, AlertCircle, AlertTriangle, Info, X } from '@lucide/vue'
import { notificationState, useNotification, AUTO_DISMISS_MS } from '../composables/useNotification'

const { t } = useI18n()
const { dismiss } = useNotification()

const iconMap = {
  success: CheckCircle2,
  error:   AlertCircle,
  warning: AlertTriangle,
  info:    Info,
}

const colorMap = {
  success: {
    bg: 'bg-green-50/50 dark:bg-green-500/10',
    icon: 'text-green-600 dark:text-green-500',
    border: 'border-green-200 dark:border-green-500/20',
    progressFill: 'bg-green-500',
  },
  error: {
    bg: 'bg-red-50/50 dark:bg-red-500/10',
    icon: 'text-red-600 dark:text-red-500',
    border: 'border-red-200 dark:border-red-500/20',
    progressFill: 'bg-red-500',
  },
  warning: {
    bg: 'bg-amber-50/50 dark:bg-amber-500/10',
    icon: 'text-amber-600 dark:text-amber-500',
    border: 'border-amber-200 dark:border-amber-500/20',
    progressFill: 'bg-amber-500',
  },
  info: {
    bg: 'bg-blue-50/50 dark:bg-blue-500/10',
    icon: 'text-blue-600 dark:text-blue-500',
    border: 'border-blue-200 dark:border-blue-500/20',
    progressFill: 'bg-blue-500',
  },
}
</script>

<template>
  <Transition name="banner">
    <div
      v-if="notificationState"
      class="fixed top-0 left-0 right-0 z-60"
    >
      <div
        :class="[
          'relative flex items-center gap-3 border-b px-4 py-3 shadow-lg shadow-black/5 dark:shadow-black/20 backdrop-blur-md sm:px-6',
          'bg-white/95 dark:bg-[#0A0A0A]/95',
          colorMap[notificationState.type].border,
        ]"
      >
        <div class="mx-auto flex w-full max-w-5xl items-center gap-3">
          <!-- Icon with glow -->
          <div class="relative flex items-center justify-center shrink-0 w-6 h-6 rounded-full" :class="colorMap[notificationState.type].bg">
            <div class="absolute inset-0 rounded-full blur-md opacity-20 pointer-events-none" :class="colorMap[notificationState.type].bg"></div>
            <component
              :is="iconMap[notificationState.type]"
              :size="14"
              :class="colorMap[notificationState.type].icon"
            />
          </div>

          <!-- Message -->
          <span class="flex-1 text-sm font-medium text-gray-900 dark:text-gray-100 tracking-tight leading-snug">
            {{ notificationState.message }}
          </span>

          <!-- Dismiss -->
          <button
            @click="dismiss"
            class="shrink-0 flex items-center justify-center w-10 h-10 rounded-md text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors -mr-1"
            :aria-label="t('notificationBanner.dismiss')"
          >
            <X :size="14" />
          </button>
        </div>

        <!-- Auto-dismiss countdown -->
        <div class="absolute bottom-0 left-0 h-0.5 w-full bg-black/5 dark:bg-white/10">
          <div
            :key="notificationState.id"
            class="h-full w-full origin-left"
            :class="colorMap[notificationState.type].progressFill"
            :style="{ animation: `banner-progress ${AUTO_DISMISS_MS}ms linear forwards` }"
          ></div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.banner-enter-active,
.banner-leave-active {
  transition: opacity 0.3s ease, transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
.banner-enter-from,
.banner-leave-to {
  opacity: 0;
  transform: translateY(-100%);
}
.banner-enter-to,
.banner-leave-from {
  opacity: 1;
  transform: translateY(0);
}

@keyframes banner-progress {
  from { transform: scaleX(1); }
  to { transform: scaleX(0); }
}
</style>
