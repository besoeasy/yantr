<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Globe, FileCode, Tag, Info } from "lucide-vue-next";
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
    <div class="group relative bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6 flex flex-col sm:flex-row gap-6 transition-all hover:border-gray-300 dark:hover:border-zinc-700">
      <!-- Glow Accent -->
      <div class="absolute top-0 left-0 w-full h-0.5 bg-linear-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

      <div class="w-20 h-20 bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 rounded-xl flex items-center justify-center p-4 shrink-0 shadow-sm transition-transform group-hover:scale-105 duration-500">
        <AppLogo
          :logo="app.logo"
          :name="app.name"
          :seed="app.id || app.name"
          img-class="w-full h-full object-contain filter dark:brightness-90 group-hover:brightness-100 transition-all"
          icon-class="w-full h-full text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 transition-colors"
        />
      </div>

      <div class="flex-1 flex flex-col">
        <div class="flex flex-col mb-3">
          <div class="flex items-center gap-3 mb-2">
            <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{{ app.name }}</h1>
            <span
              v-if="app.customapp"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-purple-50 dark:bg-purple-500/10 border border-purple-200 dark:border-purple-500/20 text-[10px] font-bold uppercase tracking-widest text-purple-600 dark:text-purple-400 shrink-0"
            >
              Custom App by Yantr
            </span>
          </div>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="tag in appTags"
              :key="tag"
              class="inline-flex items-center px-2 py-0.5 border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-[10px] font-bold uppercase tracking-widest text-gray-600 dark:text-zinc-400 rounded-md"
            >
              {{ tag }}
            </span>
          </div>
        </div>

        <p class="text-gray-500 dark:text-zinc-400 text-sm leading-relaxed mb-6">
          {{ app.description || t('appDetail.noDescription') }}
        </p>

        <!-- Action Links -->
        <div class="flex flex-wrap gap-3 mt-auto">
          <a
            v-if="app.website"
            :href="app.website"
            target="_blank"
            class="inline-flex items-center gap-1.5 text-gray-500 dark:text-zinc-400 hover:text-blue-500 dark:hover:text-blue-400 transition-colors text-[11px] font-bold uppercase tracking-wider"
          >
            <Globe :size="14" /> {{ t('appDetail.website') }}
          </a>
          <a
            :href="`https://github.com/besoeasy/yantr/blob/main/apps/${app.id}/compose.yml`"
            target="_blank"
            class="inline-flex items-center gap-1.5 text-gray-500 dark:text-zinc-400 hover:text-blue-500 dark:hover:text-blue-400 transition-colors text-[11px] font-bold uppercase tracking-wider"
          >
            <FileCode :size="14" /> {{ t('appDetail.source') }}
          </a>
          <a
            :href="chatGptUrl"
            target="_blank"
            class="inline-flex items-center gap-1.5 text-gray-500 dark:text-zinc-400 hover:text-blue-500 dark:hover:text-blue-400 transition-colors text-[11px] font-bold uppercase tracking-wider"
          >
            <Info :size="14" /> {{ t('appDetail.explain') }}
          </a>
        </div>
      </div>
    </div>

    <!-- Network Requirements (from info.json ports) -->
    <div v-if="infoPorts.length > 0" class="space-y-4">
      <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('appDetail.networkRequirements') }}</h3>

      <div class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden">
          <table class="w-full text-left text-sm">
              <thead>
                  <tr class="bg-gray-50 dark:bg-zinc-900 border-b border-gray-200 dark:border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">
                      <th class="px-4 py-3">{{ t('appDetail.port') }}</th>
                      <th class="px-4 py-3">{{ t('appDetail.protocol') }}</th>
                      <th class="px-4 py-3">{{ t('appDetail.label') }}</th>
                  </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-zinc-800">
                  <tr v-for="(p, idx) in infoPorts" :key="idx" class="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                      <td class="px-4 py-3 font-mono font-bold text-gray-900 dark:text-white">{{ p.port }}</td>
                      <td class="px-4 py-3">
                          <span class="text-[10px] uppercase tracking-wider px-2 py-0.5 border border-gray-200 dark:border-zinc-700 text-gray-500 dark:text-zinc-400 rounded bg-gray-50 dark:bg-zinc-800/50">{{ p.protocol }}</span>
                      </td>
                      <td class="px-4 py-3 text-gray-500 dark:text-zinc-400 font-mono text-xs">{{ p.label }}</td>
                  </tr>
              </tbody>
          </table>
      </div>
    </div>

    <!-- Image Details -->
    <div v-if="imageDetails && imageDetails.length > 0" class="space-y-4">
       <div class="flex items-center justify-between">
          <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('appDetail.dependentImages') }}</h3>
          <span class="text-[10px] font-mono text-gray-500 bg-gray-100 dark:bg-zinc-800 px-2 py-0.5 rounded">{{ imageDetails.length }}</span>
       </div>

       <div class="grid grid-cols-1 gap-3">
          <div v-for="img in imageDetails" :key="img.id" class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 transition-all hover:border-gray-300 dark:hover:border-zinc-600">
             <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
                 <div class="flex flex-wrap gap-2">
                   <div v-for="tag in img.tags" :key="tag" class="flex items-center gap-1.5 px-2 py-1 bg-gray-50 dark:bg-zinc-900/50 text-gray-700 dark:text-zinc-300 border border-gray-200 dark:border-zinc-800 rounded-md text-[10px] font-mono">
                     <Tag :size="10" class="text-gray-400 dark:text-zinc-500" />
                     {{ tag }}
                   </div>
                 </div>
                 <div class="font-mono text-[10px] text-gray-400 dark:text-zinc-500">{{ img.shortId }}</div>
             </div>

             <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-xs">
                <div>
                   <div class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-1">{{ t('appDetail.platform') }}</div>
                   <div class="font-mono text-gray-900 dark:text-zinc-200">{{ img.architecture }} / {{ img.os }}</div>
                </div>
                <div>
                   <div class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-1">{{ t('appDetail.size') }}</div>
                   <div class="font-mono text-gray-900 dark:text-zinc-200">{{ img.size }} MB</div>
                </div>
                <div>
                   <div class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-1">{{ t('appDetail.created') }}</div>
                   <div class="font-mono text-gray-900 dark:text-zinc-200 truncate" :title="img.createdDate">{{ img.relativeTime }}</div>
                </div>
                <div>
                    <div class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-1">{{ t('appDetail.digest') }}</div>
                    <div class="font-mono text-gray-900 dark:text-zinc-200 truncate" :title="img.digest">{{ img.digest.substring(7, 19) }}...</div>
                </div>
             </div>
          </div>
       </div>
    </div>
  </div>
</template>
