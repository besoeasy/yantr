<script setup>
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Users, ExternalLink } from "@lucide/vue";
import { h } from 'vue';

// Lucide dropped brand icons, so both marks are inlined here — the same
// approach OpenSourceCard uses for the GitHub mark.
const DISCORD_URL = "https://discord.gg/YZG5NVUcAt";

const Github = (props, context) => h('svg', {
  xmlns: "http://www.w3.org/2000/svg",
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "currentColor",
  ...props,
  ...context.attrs
}, [
  h('path', { "fill-rule": "evenodd", "clip-rule": "evenodd", d: "M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" })
]);

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

const links = [
  { title: t("home.externalLinks.reportIssue"), href: "https://github.com/besoeasy/yantr/issues", icon: ExternalLink },
];
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-indigo-200 bg-indigo-50 transition-transform duration-300 group-hover:scale-105 dark:border-indigo-900/50 dark:bg-indigo-900/20">
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

      <div class="flex h-8 w-8 shrink-0 -translate-y-1 items-center justify-center rounded-full opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100">
        <Users class="h-5 w-5 text-zinc-400 transition-colors group-hover:text-indigo-500 dark:text-zinc-600 dark:group-hover:text-indigo-400" />
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="mb-2 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
        {{ t("discordCard.title") }}
      </div>
      <div class="mb-6 text-sm leading-relaxed text-zinc-500 line-clamp-2 dark:text-zinc-400">
        {{ t("discordCard.description") }}
      </div>

      <div class="flex flex-col gap-2.5">
        <!-- Main CTA -->
        <a
          :href="DISCORD_URL"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta relative flex h-12 w-full items-center justify-between rounded-xl bg-indigo-500 px-5 text-white transition-all duration-300 hover:bg-indigo-600 active:scale-[0.98] overflow-hidden"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Discord class="h-4 w-4 fill-current" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('discordCard.cta') }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-300 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>

        <!-- Minor Links grid -->
        <div class="grid grid-cols-2 gap-2.5">
          <a
            href="https://github.com/besoeasy/yantr"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link flex h-10 items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-white text-zinc-600 transition-all duration-300 hover:bg-zinc-50 hover:text-zinc-900 dark:border-zinc-800 dark:bg-black dark:text-zinc-400 dark:hover:bg-zinc-900/50 dark:hover:text-white"
          >
            <Github class="h-3.5 w-3.5" />
            <span class="text-[9px] font-semibold uppercase tracking-widest">{{ t("discordCard.repo") }}</span>
          </a>
          <a
            v-for="link in links"
            :key="link.title"
            :href="link.href"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link flex h-10 items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-white text-zinc-600 transition-all duration-300 hover:bg-zinc-50 hover:text-zinc-900 dark:border-zinc-800 dark:bg-black dark:text-zinc-400 dark:hover:bg-zinc-900/50 dark:hover:text-white"
          >
            <component :is="link.icon" class="h-3.5 w-3.5" />
            <span class="text-[9px] font-semibold uppercase tracking-widest">{{ link.title }}</span>
          </a>
        </div>
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