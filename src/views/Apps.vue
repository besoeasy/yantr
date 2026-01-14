<script setup>
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useToast } from "vue-toastification";
import { Tag, Grid3x3 } from "lucide-vue-next";

const router = useRouter();
const toast = useToast();

// State
const apps = ref([]);
const containers = ref([]);
const loading = ref(false);
const appSearch = ref("");
const apiUrl = ref("");
const selectedCategory = ref(null);

// Computed
const installedAppIds = computed(() => {
  const ids = new Set(containers.value.map((c) => c.app.id));
  return ids;
});

const appInstanceCounts = computed(() => {
  const counts = {};
  containers.value.forEach((c) => {
    if (c.app.id) {
      counts[c.app.id] = (counts[c.app.id] || 0) + 1;
    }
  });
  return counts;
});

const allAppsShuffled = computed(() => {
  // Show ALL apps (both installed and uninstalled), shuffled
  return shuffleWithSeed(apps.value).map((app) => ({
    ...app,
    isInstalled: installedAppIds.value.has(app.id),
  }));
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

  // Calculate count for each category and filter out categories with less than 2 apps
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
  let combined = [...allAppsShuffled.value];

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

  // Auto-refresh every 10 seconds
  setInterval(() => {
    fetchContainers();
  }, 10000);
});
</script>

<template>
  <div class="relative">
    <!-- Main Content Area -->
    <div class="lg:mr-64 xl:mr-72 p-4 sm:p-6 md:p-10 lg:p-12">
      <!-- Search Bar -->
      <div class="mb-6 md:mb-8">
        <div class="relative">
          <input
            v-model="appSearch"
            type="text"
            placeholder="Search apps..."
            class="w-full px-4 sm:px-5 py-3 sm:py-4 pl-11 sm:pl-12 bg-white border border-gray-200 rounded-xl text-gray-900 placeholder-gray-400 transition-all text-base"
          />
          <span class="absolute left-3 sm:left-4 top-1/2 -translate-y-1/2 text-lg sm:text-xl">🔍</span>
          <button
            v-if="appSearch"
            @click="appSearch = ''"
            class="absolute right-3 sm:right-4 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors p-1 touch-manipulation"
            title="Clear search"
          >
            ✕
          </button>
        </div>
        <div class="mt-2 text-sm text-gray-500 flex items-center justify-between">
          <span v-if="combinedApps.length < allAppsCount || selectedCategory">
            Showing {{ combinedApps.length }} of {{ allAppsCount }} apps
            <span v-if="selectedCategory" class="inline-flex items-center gap-1">
              in <span class="font-semibold text-blue-600">{{ selectedCategory }}</span>
              <button @click="selectedCategory = null" class="ml-1 text-gray-400 hover:text-gray-600 transition-colors" title="Clear filter">✕</button>
            </span>
          </span>
        </div>
      </div>

      <div v-if="loading" class="text-center py-16">
        <div class="text-gray-500 font-medium">Loading apps...</div>
      </div>
      <div v-else-if="combinedApps.length === 0" class="text-center py-16">
        <div class="text-5xl mb-4">🔍</div>
        <div class="text-gray-500 font-medium">No apps found</div>
        <div class="text-sm text-gray-400 mt-2">Try a different search term</div>
      </div>
      <div v-else class="grid grid-cols-1 gap-1 sm:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        <div
          v-for="(app, index) in combinedApps"
          :key="app.id"
          :style="{ animationDelay: `${index * 30}ms` }"
          @click="viewAppDetail(app.id)"
          class="group bg-white/80 backdrop-blur-sm p-4 sm:p-6 transition-all duration-500 ease-out hover:bg-white hover:shadow-xl hover:scale-[1.02] active:scale-[0.98] cursor-pointer flex flex-col animate-fadeIn"
        >
          <div class="flex items-center gap-3 sm:gap-4 mb-3 sm:mb-4">
            <div class="relative flex-shrink-0">
              <!-- Gradient effect on hover -->
              <div class="absolute inset-0 bg-gradient-to-br from-blue-400/0 to-purple-500/0 group-hover:from-blue-400/20 group-hover:to-purple-500/20 rounded-2xl blur-xl transition-all duration-300"></div>
              <img
                :src="app.logo"
                :alt="app.name"
                class="relative w-12 h-12 sm:w-16 sm:h-16 object-contain transition-all duration-300 group-hover:scale-110 group-hover:rotate-3"
              />
            </div>

            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <h3 class="font-bold text-base sm:text-lg text-gray-900 truncate group-hover:text-blue-600 transition-colors">
                  {{ app.name }}
                </h3>
                <span v-if="appInstanceCounts[app.id] > 1" class="flex-shrink-0 text-xs px-2 py-0.5 rounded-full bg-green-100 text-green-700 font-bold">
                  {{ appInstanceCounts[app.id] }}
                </span>
              </div>
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="(cat, idx) in app.category.split(',').slice(0, 3)"
                  :key="cat"
                  class="text-xs px-2 py-0.5 rounded-md bg-blue-50/30 text-blue-700 font-medium"
                >
                  {{ cat.trim() }}
                </span>
              </div>
            </div>
          </div>

          <!-- Description -->
          <p class="text-sm text-gray-600 leading-relaxed line-clamp-2">
            {{ app.description || "No description available" }}
          </p>
        </div>
      </div>
    </div>

    <!-- Categories Sidebar - Fixed to right side on larger screens -->
    <aside class="hidden lg:block fixed top-0 right-0 w-64 xl:w-72 h-screen bg-white border-l border-gray-200 z-40 overflow-y-auto custom-scrollbar">
      <div class="p-6">
        <div class="flex items-center gap-2 mb-6 pb-4 border-b border-gray-100">
          <Tag :size="20" class="text-gray-900" />
          <h3 class="text-lg font-bold text-gray-900">Categories</h3>
        </div>

        <div class="space-y-1.5">
          <!-- All Apps Option -->
          <button
            @click="selectedCategory = null"
            :class="selectedCategory === null ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'"
            class="w-full flex items-center gap-3 px-3 py-3 rounded-xl transition-all duration-200 group"
          >
            <component
              :is="Grid3x3"
              :size="20"
              :class="selectedCategory === null ? 'text-white' : 'text-gray-400 group-hover:text-gray-600'"
              class="flex-shrink-0 transition-colors"
            />
            <span class="text-sm font-medium flex-1 text-left">All apps</span>
            <span
              class="text-xs font-bold px-2 py-1 rounded-full min-w-[28px] text-center"
              :class="selectedCategory === null ? 'bg-gray-800 text-white' : 'bg-gray-100 text-gray-600 group-hover:bg-gray-200'"
            >
              {{ allAppsCount }}
            </span>
          </button>

          <!-- Category Buttons -->
          <button
            v-for="category in categories"
            :key="category.name"
            @click="selectedCategory = category.name"
            :class="selectedCategory === category.name ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'"
            class="w-full flex items-center gap-3 px-3 py-3 rounded-xl transition-all duration-200 group"
          >
            <component
              :is="Tag"
              :size="18"
              :class="selectedCategory === category.name ? 'text-white' : 'text-gray-400 group-hover:text-gray-600'"
              class="flex-shrink-0 transition-colors"
            />
            <span class="text-sm font-medium capitalize flex-1 text-left">{{ category.name }}</span>
            <span
              class="text-xs font-bold px-2 py-1 rounded-full min-w-[28px] text-center"
              :class="selectedCategory === category.name ? 'bg-gray-800 text-white' : 'bg-gray-100 text-gray-600 group-hover:bg-gray-200'"
            >
              {{ category.count }}
            </span>
          </button>
        </div>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #e5e7eb;
  border-radius: 3px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #d1d5db;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-fadeIn {
  animation: fadeIn 0.4s ease-out forwards;
}
</style>
