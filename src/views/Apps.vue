<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import AppCard from "../components/AppCard.vue";
import { Tag, Search, LayoutGrid, X, Command } from "lucide-vue-next";

const { t } = useI18n();

const searchInput = ref(null);

function focusSearch(e) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault();
    searchInput.value?.focus();
  }
}

const router = useRouter();
const { apiUrl } = useApiUrl();

// State
const apps = ref([]);
const containers = ref([]);
const loading = ref(false);
const appSearch = ref("");
const selectedTag = ref(null);

const hourSeed = ref(getDateHourSeed());
let refreshInterval = null;
let seedInterval = null;

// Computed
const installedAppIds = computed(() => {
  const ids = new Set(containers.value.map((c) => c?.app?.id).filter(Boolean));
  return ids;
});

const appInstanceCounts = computed(() => {
  const projectsByApp = {};
  containers.value.forEach((c) => {
    const appId = c?.app?.id;
    const projectId = c?.app?.projectId;
    if (appId && projectId) {
      if (!projectsByApp[appId]) projectsByApp[appId] = new Set();
      projectsByApp[appId].add(projectId);
    }
  });
  const counts = {};
  for (const [appId, projects] of Object.entries(projectsByApp)) {
    counts[appId] = projects.size;
  }
  return counts;
});

const shuffledApps = computed(() => {
  // Deterministic shuffle, updated at most once per hour
  void hourSeed.value;
  return shuffleWithSeed(apps.value);
});

const allAppsCount = computed(() => apps.value.length);

const tags = computed(() => {
  const tagSet = new Set();
  apps.value.forEach((app) => {
    if (Array.isArray(app.tags)) {
      app.tags.forEach((tag) => tagSet.add(tag));
    }
  });

  const tagsArray = Array.from(tagSet).sort();

  return tagsArray
    .map((tag) => {
      const count = apps.value.filter((app) =>
        Array.isArray(app.tags) && app.tags.includes(tag)
      ).length;
      return { name: tag, count };
    })
    .filter((tag) => tag.count > 2);
});

const combinedApps = computed(() => {
  let combined = shuffledApps.value.map((app) => ({
    ...app,
    isInstalled: installedAppIds.value.has(app.id),
  }));

  if (selectedTag.value) {
    combined = combined.filter((app) =>
      Array.isArray(app.tags) && app.tags.includes(selectedTag.value)
    );
  }

  if (appSearch.value) {
    const search = appSearch.value.toLowerCase();
    combined = combined.filter((app) => {
      return (
        app.name.toLowerCase().includes(search) ||
        (Array.isArray(app.tags) && app.tags.join(' ').toLowerCase().includes(search)) ||
        (app.description && app.description.toLowerCase().includes(search))
      );
    });
  }

  return combined;
});

// Helper Functions
function getDateHourSeed() {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  const hour = String(now.getHours()).padStart(2, "0");
  return `${year}-${month}-${day}-${hour}`;
}

function seededRandom(seed) {
  let x = Math.sin(seed) * 10000;
  return x - Math.floor(x);
}

function shuffleWithSeed(array) {
  const dateHourSeed = getDateHourSeed();
  let numericSeed = 0;
  for (let i = 0; i < dateHourSeed.length; i++) {
    numericSeed += dateHourSeed.charCodeAt(i) * (i + 1);
  }

  const shuffled = [...array];

  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(seededRandom(numericSeed + i) * (i + 1));
    [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
  }

  return shuffled;
}

// API Functions
async function fetchApps() {
  try {
    const response = await fetch(`${apiUrl.value}/api/apps`);
    const data = await response.json();
    if (data.success) {
      apps.value = data.apps;
    }
  } catch (error) {
    console.error("Failed to fetch apps:", error);
  }
}

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`);
    const data = await response.json();
    if (data.success) {
      containers.value = data.containers.filter((c) => c.state === "running");
    }
  } catch (error) {
    console.error("Failed to fetch containers:", error);
  }
}

function viewAppDetail(appId) {
  if (appInstanceCounts.value[appId] > 0) {
    router.push(`/app/${appId}`);
  } else {
    router.push(`/apps/${appId}`);
  }
}

// Lifecycle
onMounted(async () => {
  window.addEventListener('keydown', focusSearch);
  loading.value = true;
  await Promise.all([fetchApps(), fetchContainers()]);
  loading.value = false;

  refreshInterval = setInterval(fetchContainers, 10000);

  seedInterval = setInterval(() => {
    const nextSeed = getDateHourSeed();
    if (nextSeed !== hourSeed.value) {
      hourSeed.value = nextSeed;
    }
  }, 60_000);
});

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval);
  if (seedInterval) clearInterval(seedInterval);
  window.removeEventListener('keydown', focusSearch);
});
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-zinc-100 font-sans flex flex-col lg:flex-row selection:bg-blue-500/30">
    
    <!-- Sidebar / Filters — appears ABOVE main content on mobile, right panel on desktop -->
    <aside class="w-full lg:w-64 shrink-0 bg-white dark:bg-[#0A0A0A] border-b lg:border-b-0 lg:border-l border-gray-200 dark:border-zinc-800 lg:h-screen lg:sticky lg:top-0 overflow-y-auto order-1 lg:order-2 scrollbar-none">
      <div class="p-4 lg:p-6">

        <!-- Header -->
        <div class="mb-4 lg:mb-8">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('apps.categories') }}</h2>
          <p class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mt-1.5">{{ allAppsCount }} {{ t('apps.totalApps') }}</p>
        </div>

        <!-- All button -->
        <button
          @click="selectedTag = null"
          :class="[
            'w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm transition-all duration-200 mb-2',
            selectedTag === null
              ? 'bg-black dark:bg-white text-white dark:text-black font-semibold shadow-md'
              : 'text-gray-600 dark:text-zinc-400 hover:bg-gray-50 dark:hover:bg-zinc-900 font-medium'
          ]"
        >
          <div class="flex items-center gap-3">
            <LayoutGrid :size="15" :class="selectedTag === null ? 'opacity-100' : 'opacity-70'" />
            {{ t('apps.allAppsLabel') }}
          </div>
          <span :class="[
            'text-[10px] font-bold tracking-widest tabular-nums px-2 py-0.5 rounded-full',
            selectedTag === null
              ? 'bg-white/20 text-white dark:bg-black/10 dark:text-black'
              : 'bg-gray-100 dark:bg-zinc-800/80 text-gray-500 dark:text-zinc-400'
          ]">{{ allAppsCount }}</span>
        </button>

        <!-- Divider -->
        <div class="my-3 border-t border-gray-100 dark:border-zinc-800"></div>

        <!-- Category list — horizontal scroll on mobile, vertical on desktop -->
        <div class="flex lg:flex-col gap-1 overflow-x-auto lg:overflow-x-visible pb-1 lg:pb-0 scrollbar-none lg:space-y-0">
          <button
            v-for="cat in tags"
            :key="cat.name"
            @click="selectedTag = cat.name"
            :class="[
              'flex items-center justify-between px-3 py-2 rounded-lg text-sm transition-all duration-200 group shrink-0 lg:w-full',
              selectedTag === cat.name
                ? 'bg-black dark:bg-white text-white dark:text-black font-semibold shadow-md'
                : 'text-gray-600 dark:text-zinc-400 hover:bg-gray-50 dark:hover:bg-zinc-900 font-medium'
            ]"
          >
            <div class="flex items-center gap-2 min-w-0">
              <Tag :size="14" class="shrink-0 transition-opacity" :class="selectedTag === cat.name ? 'opacity-100' : 'opacity-50 group-hover:opacity-100'" />
              <span class="whitespace-nowrap lg:truncate">{{ cat.name }}</span>
            </div>
            <span :class="[
              'text-[10px] font-bold tracking-widest tabular-nums px-2 py-0.5 rounded-full shrink-0 ml-2',
              selectedTag === cat.name
                ? 'bg-white/20 text-white dark:bg-black/10 dark:text-black'
                : 'bg-gray-100 dark:bg-zinc-800/80 text-gray-500 dark:text-zinc-400 group-hover:bg-gray-200 dark:group-hover:bg-zinc-700'
            ]">{{ cat.count }}</span>
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 min-w-0 order-2 lg:order-1">
        <!-- Top Bar -->
        <div class="sticky top-0 z-30 bg-white/80 dark:bg-[#0A0A0A]/80 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800 px-6 py-4">

          <!-- Search input -->
          <div class="relative group max-w-2xl">
            <Search :size="16" class="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none transition-colors group-focus-within:text-blue-500 dark:group-focus-within:text-blue-400" />
            <input
              ref="searchInput"
              v-model="appSearch"
              type="text"
              :placeholder="t('apps.searchPlaceholder')"
              class="w-full bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-xl pl-11 pr-24 py-2.5 text-sm text-gray-900 dark:text-white placeholder:text-gray-500 dark:placeholder:text-zinc-500 focus:outline-none focus:border-gray-300 dark:focus:border-zinc-700 focus:bg-white dark:focus:bg-[#0A0A0A] focus:shadow-sm transition-all duration-300"
            />
            <!-- Kbd hint -->
            <div class="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1 pointer-events-none" :class="appSearch ? 'hidden' : ''">
              <kbd class="inline-flex items-center gap-0.5 px-2 py-1 rounded-md text-[10px] font-bold uppercase tracking-widest bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border border-gray-200 dark:border-zinc-700">
                <Command :size="10" />
                K
              </kbd>
            </div>
            <!-- Clear button -->
            <button
              v-if="appSearch"
              @click="appSearch = ''"
              class="absolute right-3 top-1/2 -translate-y-1/2 p-1.5 rounded-lg text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors"
            >
              <X :size="14" />
            </button>
          </div>

          <!-- Active filters + result count -->
          <div class="mt-3 flex items-center gap-2 flex-wrap min-h-[24px]">
            <span class="text-[10px] font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500">
              {{ combinedApps.length }} {{ combinedApps.length === 1 ? t('apps.app') : t('apps.apps') }}
            </span>
            <span
              v-if="selectedTag"
              class="inline-flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black px-2.5 py-1 rounded-full"
            >
              {{ selectedTag }}
              <button @click="selectedTag = null" class="ml-1 hover:opacity-70 transition-opacity">
                <X :size="10" />
              </button>
            </span>
          </div>
        </div>

        <!-- content area -->
        <div class="p-4 sm:p-6">
            <div v-if="loading" class="flex flex-col items-center justify-center py-32">
               <div class="w-8 h-8 border-[3px] border-gray-200 dark:border-zinc-800 border-t-blue-500 dark:border-t-blue-500 rounded-full animate-spin mb-6"></div>
               <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('apps.loadingCatalog') }}</div>
            </div>

            <div v-else-if="combinedApps.length === 0" class="flex flex-col items-center justify-center py-32 text-center">
                <div class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-2xl p-5 mb-5">
                   <Search :size="28" class="text-gray-400 dark:text-zinc-500" />
                </div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('apps.noAppsFound') }}</h3>
                <p class="text-sm text-gray-500 dark:text-zinc-400 mt-2 max-w-sm mx-auto leading-relaxed">{{ t('apps.noAppsFoundDesc') }}</p>
                <button @click="appSearch = ''; selectedTag = null" class="mt-6 text-xs font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors">{{ t('apps.clearAllFilters') }}</button>
            </div>

            <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
                <AppCard
                  v-for="app in combinedApps"
                  :key="app.id"
                  :app="app"
                  :instance-count="appInstanceCounts[app.id] || 0"
                  @select="viewAppDetail(app.id)"
                />
            </div>
        </div>

    </main>
  </div>
</template>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
