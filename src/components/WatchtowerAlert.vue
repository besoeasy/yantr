<script setup>
import { useRouter } from 'vue-router'
import { ShieldAlert, ArrowRight, Zap } from 'lucide-vue-next'

const router = useRouter()
</script>

<template>
  <div
    @click="router.push('/apps/watchtower')"
    class="relative h-full overflow-hidden group rounded-2xl cursor-pointer transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] hover:-translate-y-1"
  >
    <!-- Animated Background Mesh -->
    <div class="absolute inset-0 bg-white dark:bg-gray-900">
      <div class="absolute inset-0 bg-gradient-to-br from-orange-200/60 via-red-200/30 to-white/80 dark:from-orange-600/20 dark:via-red-600/10 dark:to-gray-900 z-10"></div>
      
      <!-- Animated Orbs -->
      <div class="absolute top-0 right-0 w-64 h-64 bg-orange-300/35 dark:bg-orange-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-orange-400/45 dark:group-hover:bg-orange-500/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
      <div class="absolute bottom-0 left-0 w-48 h-48 bg-red-300/30 dark:bg-red-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-red-400/40 dark:group-hover:bg-red-600/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-20 h-full p-6 flex flex-col justify-between border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm group-hover:border-orange-300/70 dark:group-hover:border-orange-500/30 transition-none">
      
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Icon Container -->
          <div class="relative">
            <div class="absolute inset-0 bg-orange-400/25 dark:bg-orange-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
            <div class="relative w-12 h-12 bg-gradient-to-br from-orange-500 to-red-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]">
              <ShieldAlert class="w-6 h-6 text-white" />
            </div>
          </div>
          
          <div>
            <h3 class="text-lg font-bold text-slate-900 dark:text-white mb-1 group-hover:text-orange-700 dark:group-hover:text-orange-200 transition-colors">
              Security Update
            </h3>
            <p class="text-sm font-medium text-slate-600 dark:text-gray-400 group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">
              Watchtower is missing
            </p>
          </div>
        </div>

        <!-- Pulse Dot -->
        <span class="relative flex h-3 w-3">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-orange-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full h-3 w-3 bg-orange-500"></span>
        </span>
      </div>

      <div class="mt-4">
        <p class="text-sm text-slate-600 dark:text-gray-400 leading-relaxed max-w-[90%] group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">
          Automate your container updates to keep your system secure and up-to-date.
        </p>
      </div>

      <!-- Footer Action -->
      <div class="mt-4 pt-4 border-t border-slate-200/70 dark:border-slate-700/60 flex items-center justify-between group-hover:border-orange-200/70 dark:group-hover:border-slate-600/70 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]">
        <div class="flex items-center gap-2 text-xs font-bold text-orange-600 dark:text-orange-400 uppercase tracking-wider">
          <Zap class="w-3.5 h-3.5" />
          <span>Recommended</span>
        </div>
        
        <div class="flex items-center gap-2 text-sm font-bold text-slate-900 dark:text-white group-hover:translate-x-1 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]">
          <span>Install Now</span>
          <ArrowRight class="w-4 h-4" />
        </div>
      </div>
    </div>
  </div>
</template>

