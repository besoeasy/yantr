<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Globe, FileCode, Tag, Info } from "@lucide/vue";
import AppLogo from "./AppLogo.vue";
import { buildChatGptExplainUrl } from "../utils/chatgpt";

const props = defineProps({
  app: {
    type: Object,
    required: true
  },
  imageDetails: {
    type: Array,
    default: () => []
  }
});

const { t } = useI18n();

const appTags = computed(() => {
  return Array.isArray(props.app?.tags) ? props.app.tags : [];
});

const infoPorts = computed(() => {
  return Array.isArray(props.app?.ports) ? props.app.ports : [];
});

const chatGptUrl = computed(() => {
  if (!props.app) return "";
  return buildChatGptExplainUrl(props.app.id);
});
</script>

<template>
  <div class="space-y-6">
    <!-- Identity Card -->
    <div class="group flex flex-col gap-6 rounded-2xl border border-zinc-200 bg-white p-6 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] sm:flex-row dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
      <div class="flex h-20 w-20 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 p-4 transition-transform duration-500 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
        <AppLogo
          :logo="app.logo"
          :name="app.name"
          :seed="app.id || app.name"
          img-class="h-full w-full object-contain filter transition-all dark:brightness-90 group-hover:brightness-100"
          icon-class="h-full w-full text-zinc-900 transition-colors group-hover:text-zinc-600 dark:text-zinc-100"
        />
      </div>

      <div class="flex flex-1 flex-col">
        <div class="mb-3 flex flex-col">
          <div class="mb-2 flex items-center gap-3">
            <h1 class="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">{{ app.name }}</h1>
            <span
              v-if="app.customapp"
              class="inline-flex shrink-0 items-center gap-1 rounded-md border border-purple-200 bg-purple-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-purple-600 dark:border-purple-500/20 dark:bg-purple-500/10 dark:text-purple-400"
            >
              Custom App by Yantr
            </span>
          </div>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="tag in appTags"
              :key="tag"
              class="inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
            >
              {{ tag }}
            </span>
          </div>
        </div>

        <p class="mb-6 text-sm leading-relaxed text-zinc-500">
          {{ app.description || t('appDetail.noDescription') }}
        </p>

        <!-- Action Links -->
        <div class="mt-auto flex flex-wrap gap-3">
          <a
            v-if="app.website"
            :href="app.website"
            target="_blank"
            class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
          >
            <Globe :size="14" /> {{ t('appDetail.website') }}
          </a>
          <a
            :href="`https://github.com/besoeasy/yantr/blob/main/apps/${app.id}/compose.yml`"
            target="_blank"
            class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
          >
            <FileCode :size="14" /> {{ t('appDetail.source') }}
          </a>
          <a
            :href="chatGptUrl"
            target="_blank"
            class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
          >
            <Info :size="14" /> {{ t('appDetail.explain') }}
          </a>
        </div>
      </div>
    </div>

    <!-- Network Requirements (from info.json ports) -->
    <div v-if="infoPorts.length > 0" class="space-y-4">
      <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('appDetail.networkRequirements') }}</h3>

      <div class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-[#0A0A0A]">
          <table class="w-full text-left text-sm">
              <thead>
                  <tr class="border-b border-zinc-200 bg-zinc-50 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                      <th class="px-4 py-3">{{ t('appDetail.port') }}</th>
                      <th class="px-4 py-3">{{ t('appDetail.protocol') }}</th>
                      <th class="px-4 py-3">{{ t('appDetail.label') }}</th>
                  </tr>
              </thead>
              <tbody class="divide-y divide-zinc-100 dark:divide-zinc-800">
                  <tr v-for="(p, idx) in infoPorts" :key="idx" class="transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                      <td class="px-4 py-3 font-mono font-bold text-zinc-900 dark:text-white">{{ p.port }}</td>
                      <td class="px-4 py-3">
                          <span class="rounded bg-zinc-50 px-2 py-0.5 text-[10px] uppercase tracking-wider text-zinc-500 dark:bg-zinc-800/50">{{ p.protocol }}</span>
                      </td>
                      <td class="px-4 py-3 font-mono text-xs text-zinc-500">{{ p.label }}</td>
                  </tr>
              </tbody>
          </table>
      </div>
    </div>

    <!-- Image Details -->
    <div v-if="imageDetails && imageDetails.length > 0" class="space-y-4">
       <div class="flex items-center justify-between">
          <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('appDetail.dependentImages') }}</h3>
          <span class="rounded border border-zinc-200 bg-zinc-50 px-2 py-0.5 font-mono text-[10px] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">{{ imageDetails.length }}</span>
       </div>

       <div class="grid grid-cols-1 gap-3">
          <div v-for="img in imageDetails" :key="img.id" class="group rounded-2xl border border-zinc-200 bg-white p-5 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
             <div class="mb-4 flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
                 <div class="flex flex-wrap gap-2">
                   <div v-for="tag in img.tags" :key="tag" class="flex items-center gap-1.5 rounded-md border border-zinc-200 bg-zinc-50 px-2 py-1 font-mono text-[10px] text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-100">
                     <Tag :size="10" class="text-zinc-500" />
                     {{ tag }}
                   </div>
                 </div>
                 <div class="font-mono text-[10px] text-zinc-500">{{ img.shortId }}</div>
             </div>

             <div class="grid grid-cols-2 gap-4 text-xs sm:grid-cols-4">
                <div>
                   <div class="mb-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('appDetail.platform') }}</div>
                   <div class="font-mono text-zinc-900 dark:text-zinc-200">{{ img.architecture }} / {{ img.os }}</div>
                </div>
                <div>
                   <div class="mb-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('appDetail.size') }}</div>
                   <div class="font-mono text-zinc-900 dark:text-zinc-200">{{ img.size }} MB</div>
                </div>
                <div>
                   <div class="mb-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('appDetail.created') }}</div>
                   <div class="truncate font-mono text-zinc-900 dark:text-zinc-200" :title="img.createdDate">{{ img.relativeTime }}</div>
                </div>
                <div>
                    <div class="mb-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('appDetail.digest') }}</div>
                    <div class="truncate font-mono text-zinc-900 dark:text-zinc-200" :title="img.digest">{{ img.digest.substring(7, 19) }}...</div>
                </div>
             </div>
          </div>
       </div>
    </div>
  </div>
</template>
