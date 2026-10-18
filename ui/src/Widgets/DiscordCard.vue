<script>
// Dashboard engine: one column — a single call to action keeps the dashboard's
// narrow-card rhythm, and wide CTAs read as banners. See Home.vue.
export const meta = { cols: 1 };
</script>
<script setup>
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Users } from "@lucide/vue";
import { h } from 'vue';

// Lucide dropped brand icons, so the Discord mark is inlined here — the same
// approach GitHubCard uses for the GitHub mark.
const DISCORD_URL = "https://discord.gg/YZG5NVUcAt";

const Discord = (props, context) => h('svg', {
  xmlns: "http://www.w3.org/2000/svg",
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "currentColor",
  ...props,
  ...context.attrs
}, [
  h('path', { d: "M20.317 4.3698a19.7913 19.7913 0 0 0-4.8851-1.5152.0741.0741 0 0 0-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 0 0-.0785-.037 19.7363 19.7363 0 0 0-4.8852 1.515.0699.0699 0 0 0-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 0 0 .0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 0 0 .0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 0 0-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 0 1-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 0 1 .0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 0 1 .0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 0 1-.0066.1276 12.2986 12.2986 0 0 1-1.873.8914.0766.0766 0 0 0-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 0 0 .0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 0 0 .0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 0 0-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z" })
]);

const { t } = useI18n();
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-indigo-200 bg-indigo-50 transition-transform duration-200 group-hover:scale-105 dark:border-indigo-900/50 dark:bg-indigo-900/20">
          <Discord class="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
            {{ t("discordCard.label") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
              {{ t("discordCard.subtitle") }}
            </span>
          </div>
        </div>
      </div>

      <div class="flex h-8 w-8 shrink-0 -translate-y-1 items-center justify-center rounded-full opacity-0 transition-[border-color,box-shadow,transform] duration-200 group-hover:translate-y-0 group-hover:opacity-100">
        <Users class="h-5 w-5 text-zinc-400 transition-colors group-hover:text-indigo-500 dark:text-zinc-600 dark:group-hover:text-indigo-400" />
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="mb-2 text-2xl font-black leading-[1.2] tracking-tight text-zinc-900 transition-colors duration-200 group-hover:-translate-y-[3px] dark:text-white">
        {{ t("discordCard.title") }}
      </div>
      <div class="mb-5 text-xs leading-relaxed text-zinc-500 line-clamp-3 dark:text-zinc-400">
        {{ t("discordCard.description") }}
      </div>

      <div class="flex flex-col gap-2.5">
        <!-- Main CTA -->
        <a
          :href="DISCORD_URL"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta relative flex h-11 w-full items-center justify-between rounded-xl bg-indigo-500 px-5 text-white transition-[border-color,box-shadow,transform] duration-200 hover:bg-indigo-600 active:scale-[0.98] overflow-hidden"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Discord class="h-4 w-4 fill-current" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('discordCard.cta') }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>
      </div>

      <!-- Footer note -->
      <div class="mt-4 flex items-center gap-1.5 border-t border-zinc-100 pt-3 text-[9px] font-semibold uppercase tracking-widest text-zinc-400 dark:border-zinc-900 dark:text-zinc-600">
        <span class="relative flex h-1.5 w-1.5">
          <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
        </span>
        {{ t("discordCard.footer") }}
      </div>
    </div>
  </div>
</template>