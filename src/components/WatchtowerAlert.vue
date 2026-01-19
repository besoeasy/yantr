<script setup>
import { useRouter } from 'vue-router'
import { ShieldAlert, ArrowRight, Zap } from 'lucide-vue-next'

const router = useRouter()
</script>

<template>
  <div
    @click="router.push('/apps/watchtower')"
    class="relative h-full overflow-hidden group rounded-2xl cursor-pointer transition-all duration-500 hover:-translate-y-1"
  >
    <!-- Animated Background Mesh -->
    <div class="absolute inset-0 bg-gray-900">
      <div class="absolute inset-0 bg-gradient-to-br from-orange-600/20 via-red-600/10 to-gray-900 z-10"></div>
      
      <!-- Animated Orbs -->
      <div class="absolute top-0 right-0 w-64 h-64 bg-orange-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-orange-500/30 transition-colors duration-700"></div>
      <div class="absolute bottom-0 left-0 w-48 h-48 bg-red-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-red-600/30 transition-colors duration-700"></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-20 h-full p-6 flex flex-col justify-between border border-white/5 rounded-2xl backdrop-blur-sm group-hover:border-orange-500/30 transition-colors duration-500">
      
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Icon Container -->
          <div class="relative">
            <div class="absolute inset-0 bg-orange-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-500"></div>
            <div class="relative w-12 h-12 bg-gradient-to-br from-orange-500 to-red-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500">
              <ShieldAlert class="w-6 h-6 text-white" />
            </div>
          </div>
          
          <div>
            <h3 class="text-lg font-bold text-white mb-1 group-hover:text-orange-200 transition-colors">
              Security Update
            </h3>
            <p class="text-sm font-medium text-gray-400 group-hover:text-gray-300 transition-colors">
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
        <p class="text-sm text-gray-400 leading-relaxed max-w-[90%] group-hover:text-gray-300 transition-colors">
          Automate your container updates to keep your system secure and up-to-date.
        </p>
      </div>

      <!-- Footer Action -->
      <div class="mt-4 pt-4 border-t border-white/5 flex items-center justify-between group-hover:border-white/10 transition-colors">
        <div class="flex items-center gap-2 text-xs font-bold text-orange-400 uppercase tracking-wider">
          <Zap class="w-3.5 h-3.5" />
          <span>Recommended</span>
        </div>
        
        <div class="flex items-center gap-2 text-sm font-bold text-white group-hover:translate-x-1 transition-transform duration-300">
          <span>Install Now</span>
          <ArrowRight class="w-4 h-4" />
        </div>
      </div>
    </div>
  </div>
</template>

