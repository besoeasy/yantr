<script setup>
defineProps({
  app: {
    type: Object,
    required: true
  },
  instanceCount: {
    type: Number,
    default: 0
  }
})
</script>

<template>
  <div class="app-card group bg-white rounded-2xl p-5 sm:p-6 transition-all duration-500 ease-out hover:shadow-2xl hover:shadow-gray-900/10 hover:-translate-y-1 active:scale-[0.98] cursor-pointer flex flex-col animate-fadeIn overflow-hidden relative">
    <!-- Gradient background on hover -->
    <div class="absolute inset-0 bg-gradient-to-br from-gray-50/0 to-gray-50/0 group-hover:from-gray-50/50 group-hover:to-gray-50/30 transition-all duration-500 pointer-events-none"></div>

    <!-- Content wrapper -->
    <div class="relative z-10 flex flex-col h-full">
      <!-- Header with icon and title -->
      <div class="flex items-center gap-3 sm:gap-4 mb-4 sm:mb-5">
        <div class="relative flex-shrink-0">
          <img
            :src="app.logo"
            :alt="app.name"
            class="w-14 h-14 sm:w-16 sm:h-16 object-contain transition-all duration-300 group-hover:scale-110 group-hover:rotate-3"
          />
        </div>

        <div class="flex-1 min-w-0">
          <h3 class="font-bold text-base sm:text-lg text-gray-900 truncate group-hover:text-gray-900 transition-colors line-clamp-2">
            {{ app.name }}
          </h3>
          <p v-if="app.status" class="text-xs text-gray-500 mt-0.5">{{ app.status }}</p>
        </div>

        <!-- Instance count badge -->
        <div v-if="instanceCount > 1" class="flex-shrink-0 flex items-center justify-center w-7 h-7 rounded-full bg-gradient-to-br from-blue-100 to-blue-50">
          <span class="text-xs font-bold text-blue-700">{{ instanceCount }}</span>
        </div>
      </div>

      <!-- Description -->
      <p class="text-sm text-gray-600 leading-relaxed line-clamp-2 mb-4 flex-grow group-hover:text-gray-700 transition-colors">
        {{ app.description || "No description available" }}
      </p>

      <!-- Categories -->
      <div class="flex flex-wrap gap-1.5 mb-4">
        <span
          v-for="(cat, idx) in app.category.split(',').slice(0, 3)"
          :key="cat"
          class="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-medium bg-gray-100/60 text-gray-700 group-hover:bg-gray-100/80 transition-all duration-300"
        >
          {{ cat.trim() }}
        </span>
        <span
          v-if="app.category.split(',').length > 3"
          class="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-medium text-gray-500 bg-gray-50/80 group-hover:bg-gray-100/80 transition-all duration-300"
        >
          +{{ app.category.split(',').length - 3 }}
        </span>
      </div>

      <!-- Footer action indicator -->
      <div class="flex items-center gap-2 text-xs text-gray-400 group-hover:text-gray-700 transition-colors mt-auto pt-3">
        <span>View Details</span>
        <svg class="w-3.5 h-3.5 group-hover:translate-x-1 transition-transform duration-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-fadeIn {
  animation: fadeIn 0.5s ease-out forwards;
}
</style>
