<script setup>
import { useRouter } from 'vue-router'
import { AlertTriangle } from 'lucide-vue-next'

const router = useRouter()

defineProps({
  visible: {
    type: Boolean,
    default: true
  }
})
</script>

<template>
  <div v-if="visible"
       @click="router.push('/apps/watchtower')"
       class="relative overflow-hidden group bg-gradient-to-r from-orange-500 to-red-500 rounded-2xl shadow-sm border border-orange-100 p-1 cursor-pointer hover:shadow-md hover:-translate-y-0.5 transition-all duration-300 h-full">
    <div class="bg-white/10 h-full w-full rounded-xl p-6 relative overflow-hidden backdrop-blur-sm flex flex-col justify-center">
      <div class="absolute inset-0 bg-gradient-to-r from-orange-600/20 to-red-600/20 opacity-0 group-hover:opacity-100 transition-opacity"></div>
      
      <div class="flex items-center justify-between relative z-10">
        <div class="flex items-center gap-4">
          <div class="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center shadow-inner shrink-0">
            <AlertTriangle :size="24" class="text-white" />
          </div>
          <div>
            <h3 class="text-lg font-bold text-white mb-0.5 leading-tight">Watchtower Missing</h3>
            <p class="text-white/90 font-medium text-xs leading-snug">Enable auto-updates for security</p>
          </div>
        </div>
        
        <div class="hidden sm:flex items-center justify-center w-8 h-8 bg-white text-orange-600 rounded-full shadow-lg group-hover:scale-110 transition-transform ml-4">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
            <path d="M5 12h14" />
            <path d="m12 5 7 7-7 7" />
          </svg>
        </div>
      </div>
    </div>
  </div>
</template>
