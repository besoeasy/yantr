<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import AppCard from "../components/AppCard.vue";
import { Tag, Search, Grid, LayoutGrid, X } from "lucide-vue-next";

const router = useRouter();

// State
const apps = ref([]);
const containers = ref([]);
const loading = ref(false);
const appSearch = ref("");
const apiUrl = ref("");
const selectedCategory = ref(null);

const hourSeed = ref(getDateHourSeed());
let refreshInterval = null;
let seedInterval = null;

// Computed
const installedAppIds = computed(() => {
  const ids = new Set(containers.value.map((c) => c?.app?.id).filter(Boolean));
  return ids;
});

const appInstanceCounts = computed(() => {
  const counts = {};
  containers.value.forEach((c) => {
    const appId = c?.app?.id;
    if (appId) {
      counts[appId] = (counts[appId] || 0) + 1;
    }
  });
  return counts;
});

const shuffledApps = computed(() => {
  // Deterministic shuffle, updated at most once per hour
  void hourSeed.value;
  return shuffleWithSeed(apps.value);
});

const allAppsCount = computed(() => apps.value.length);

const categories = computed(() => {
  const categorySet = new Set();
  apps.value.forEach((app) => {
    if (app.category) {
      app.category.split(",").forEach((cat) => {
        categorySet.add(cat.trim());
      });
    }
  });

  const categoriesArray = Array.from(categorySet).sort();

  return categoriesArray
    .map((cat) => {
      const count = apps.value.filter((app) =>
        app.category
          .split(",")
          .map((c) => c.trim())
          .includes(cat)
      ).length;
      return { name: cat, count };
    })
    .filter((cat) => cat.count > 2);
});

const combinedApps = computed(() => {
  let combined = shuffledApps.value.map((app) => ({
    ...app,
    isInstalled: installedAppIds.value.has(app.id),
  }));

  if (selectedCategory.value) {
    combined = combined.filter((app) =>
      app.category
        .split(",")
        .map((c) => c.trim())
        .includes(selectedCategory.value)
    );
  }

  if (appSearch.value) {
    const search = appSearch.value.toLowerCase();
    combined = combined.filter((app) => {
      return (
        app.name.toLowerCase().includes(search) ||
        app.category.toLowerCase().includes(search) ||
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
  router.push(`/apps/${appId}`);
}

// Lifecycle
onMounted(async () => {
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
});
</script>

<template>
  <div class="min-h-screen bg-slate-50 dark:bg-[#09090b] text-slate-900 dark:text-slate-200 font-sans flex flex-col lg:flex-row">
    
    <!-- Main Content -->
    <main class="flex-1 min-w-0 order-1 lg:order-1">
        <!-- Top Bar -->
        <div class="sticky top-0 z-30 bg-slate-50/80 dark:bg-[#09090b]/80 backdrop-blur border-b border-slate-200 dark:border-slate-800 px-6 py-4">
            <div class="relative max-w-2xl">
               <input
                 v-model="appSearch"
                 type="text"
                 placeholder="Search applications..."
                 class="w-full bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 rounded-lg pl-10 pr-4 py-2.5 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 transition-all font-sans"
               />
               <Search :size="16" class="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
               <button 
                  v-if="appSearch"
                  @click="appSearch = ''"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                >
                  <X :size="14" />
               </button>
            </div>
            
            <div class="mt-3 text-xs text-slate-500 dark:text-slate-400 flex items-center gap-2" v-if="selectedCategory || appSearch">
               <span>Showing {{ combinedApps.length }} results</span>
               <span v-if="selectedCategory" class="inline-flex items-center gap-1 bg-indigo-50 dark:bg-indigo-900/20 text-indigo-600 dark:text-indigo-400 px-2 py-0.5 rounded">
                  Category: {{ selectedCategory }}
                  <button @click="selectedCategory = null" class="hover:text-indigo-800 dark:hover:text-indigo-200"><X :size="10" /></button>
               </span>
            </div>
        </div>

        <!-- content area -->
        <div class="p-6">
            <div v-if="loading" class="flex flex-col items-center justify-center py-20">
               <div class="w-8 h-8 border-2 border-slate-200 border-t-blue-500 rounded-full animate-spin mb-4"></div>
               <div class="text-sm text-slate-500">Loading catalog...</div>
            </div>

            <div v-else-if="combinedApps.length === 0" class="flex flex-col items-center justify-center py-20 text-center">
                <div class="bg-slate-100 dark:bg-slate-800 rounded-full p-4 mb-4">
                   <Search :size="32" class="text-slate-400" />
                </div>
                <h3 class="text-lg font-bold text-slate-900 dark:text-white">No apps found</h3>
                <p class="text-slate-500 mt-1 max-w-xs mx-auto">We couldn't find any apps matching your search filters.</p>
                <button @click="appSearch = ''; selectedCategory = null" class="mt-4 text-sm font-medium text-blue-600 hover:underline">Clear all filters</button>
            </div>

            <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
                <AppCard
                  v-for="app in combinedApps"
                  :key="app.id"
                  :app="app"
                  :instance-count="appInstanceCounts[app.id] || 0"
                  @click="viewAppDetail(app.id)"
                />
            </div>
        </div>

    </main>

    <!-- Sidebar / Filters -->
    <aside class="w-full lg:w-64 xl:w-72 bg-white dark:bg-[#0c0c0e] border-t lg:border-t-0 lg:border-l border-slate-200 dark:border-slate-800 lg:h-screen lg:sticky lg:top-0 overflow-y-auto order-2 lg:order-2">
       <div class="p-6">
          <div class="flex items-center gap-2 mb-8">
             <LayoutGrid class="text-indigo-500" :size="24" />
             <h1 class="text-xl font-bold text-slate-900 dark:text-white">App Catalog</h1>
          </div>

          <div class="space-y-1">
             <button
                @click="selectedCategory = null"
                :class="[
                  'w-full flex items-center justify-between px-3 py-2 text-sm font-medium rounded-md transition-colors',
                  selectedCategory === null 
                    ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-white' 
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-[#1a1a1c] hover:text-slate-900 dark:hover:text-slate-200'
                ]"
             >
                <div class="flex items-center gap-2.5">
                   <Grid :size="16" />
                   All Applications
                </div>
                <span class="text-xs font-mono font-bold">{{ allAppsCount }}</span>
             </button>

             <div class="pt-4 pb-2">
                <div class="px-3 text-xs font-bold text-slate-400 uppercase tracking-wider">Categories</div>
             </div>

             <button
                v-for="cat in categories"
                :key="cat.name"
                @click="selectedCategory = cat.name"
                :class="[
                  'w-full flex items-center justify-between px-3 py-2 text-sm font-medium rounded-md transition-colors',
                  selectedCategory === cat.name
                    ? 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-white' 
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-[#1a1a1c] hover:text-slate-900 dark:hover:text-slate-200'
                ]"
             >
                <div class="flex items-center gap-2.5">
                   <Tag :size="16" />
                   {{ cat.name }}
                </div>
                <span class="text-xs font-mono text-slate-400">{{ cat.count }}</span>
             </button>
          </div>
       </div>
    </aside>
  </div>
</template>
