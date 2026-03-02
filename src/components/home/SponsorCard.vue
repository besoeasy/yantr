<script setup>
import { ref, onMounted } from "vue";
import { Sparkles, ExternalLink, ArrowRight } from "lucide-vue-next";

const sponsor = ref(null);
const visible = ref(false);

onMounted(async () => {
  try {
    const res = await fetch("https://ipfs.io/ipns/sponser.yantr.org");
    if (!res.ok) return;
    const data = await res.json();
    // Validate required fields before showing
    if (data?.heading && data?.link?.url && data?.link?.cta) {
      sponsor.value = data;
      visible.value = true;
    }
  } catch {
    // Silently fail — no sponsor card shown
  }
});
</script>

<template>
  <transition
    enter-active-class="transition-all duration-500 ease-out"
    enter-from-class="opacity-0 translate-y-2"
    enter-to-class="opacity-100 translate-y-0"
  >
    <a
      v-if="visible && sponsor"
      :href="sponsor.link.url"
      target="_blank"
      rel="noopener noreferrer"
      class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600 hover:-translate-y-1 focus:outline-none"
    >
      <!-- Hover accent line (amber for sponsor) -->
      <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-amber-400 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
      <!-- Dot-grid pattern -->
      <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

      <div class="relative z-10 flex flex-col h-full p-6">
        <!-- Header -->
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-100 dark:border-amber-900/50 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
            <Sparkles class="w-5 h-5 text-amber-500" />
          </div>
          <div>
            <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-amber-500 dark:text-amber-400">
              Sponsor
            </div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-snug line-clamp-1">
              {{ sponsor.heading }}
            </h3>
          </div>
        </div>

        <!-- Subheading -->
        <p
          v-if="sponsor.subheading"
          class="text-xs font-medium text-gray-500 dark:text-zinc-400 leading-relaxed line-clamp-3 mb-5 flex-1"
        >
          {{ sponsor.subheading }}
        </p>

        <!-- CTA -->
        <div class="mt-auto pt-4 border-t border-gray-100 dark:border-zinc-800/80 flex items-center justify-between">
          <span class="text-xs font-bold text-amber-500 dark:text-amber-400 uppercase tracking-wider flex items-center gap-1.5">
            {{ sponsor.link.cta }}
            <ArrowRight
              class="w-3.5 h-3.5 transition-transform duration-300 group-hover:translate-x-1"
            />
          </span>
          <ExternalLink class="w-3.5 h-3.5 text-gray-300 dark:text-zinc-600 group-hover:text-amber-400 transition-colors duration-300" />
        </div>
      </div>
    </a>
  </transition>
</template>
