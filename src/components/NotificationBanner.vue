<script setup>
import { useI18n } from 'vue-i18n'
import { CheckCircle2, AlertCircle, AlertTriangle, Info, X } from 'lucide-vue-next'
import { notificationState, useNotification } from '../composables/useNotification'

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
  },
  error: {
    bg: 'bg-red-50/50 dark:bg-red-500/10',
    icon: 'text-red-600 dark:text-red-500',
    border: 'border-red-200 dark:border-red-500/20',
  },
  warning: {
    bg: 'bg-amber-50/50 dark:bg-amber-500/10',
    icon: 'text-amber-600 dark:text-amber-500',
    border: 'border-amber-200 dark:border-amber-500/20',
  },
  info: {
    bg: 'bg-blue-50/50 dark:bg-blue-500/10',
    icon: 'text-blue-600 dark:text-blue-500',
    border: 'border-blue-200 dark:border-blue-500/20',
  },
}
</script>

<template>
  <Transition name="banner">
    <div
      v-if="notificationState"
      class="fixed top-4 left-1/2 -translate-x-1/2 md:left-auto md:translate-x-0 md:right-6 md:top-6 z-[60] w-[calc(100%-2rem)] max-w-sm pointer-events-auto"
    >
      <div
        :class="[
          'relative flex items-center gap-3 rounded-lg px-4 py-3 shadow-2xl shadow-black/10 dark:shadow-black/40 border backdrop-blur-md',
          'bg-white/90 dark:bg-[#0A0A0A]/95',
          colorMap[notificationState.type].border,
        ]"
      >
        <!-- Dynamic Glow Behind Icon -->
        <div class="absolute left-4 w-6 h-6 rounded-full blur-md opacity-20 pointer-events-none" :class="colorMap[notificationState.type].bg"></div>
        
        <!-- Icon -->
        <div class="relative flex items-center justify-center shrink-0 w-6 h-6 rounded-full" :class="colorMap[notificationState.type].bg">
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
  transform: translateY(-16px) scale(0.96);
}
.banner-enter-to,
.banner-leave-from {
  opacity: 1;
  transform: translateY(0) scale(1);
}
</style>
