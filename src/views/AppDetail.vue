<script setup>
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useToast } from "vue-toastification";
import { Globe, FileCode, ArrowLeft, Package, Clock, Tag } from "lucide-vue-next";

const route = useRoute();
const router = useRouter();
const toast = useToast();

// State
const app = ref(null);
const containers = ref([]);
const loading = ref(true);
const deploying = ref(false);
const envValues = ref({});
const temporaryInstall = ref(false);
const expirationHours = ref(24);
const apiUrl = ref("");

// Computed
const isInstalled = computed(() => {
  return containers.value.some((c) => c.app.id === route.params.appname);
});

const ports = computed(() => {
  if (!app.value?.port) return [];
  return app.value.port.split(",").map((p) => p.trim());
});

const categories = computed(() => {
  if (!app.value?.category) return [];
  return app.value.category.split(",").map((c) => c.trim());
});

const chatGptUrl = computed(() => {
  if (!app.value) return '';
  
  const composeUrl = `https://github.com/besoeasy/yantra/blob/main/apps/${app.value.id}/compose.yml`;
  const query = `Understand this Yantra Docker stack:\n${composeUrl}\n\nTell me:\n1. What does this app do?\n2.(Yantra handles deployment, so skip Docker/installation commands)\n3. 5 main features of this app\n\n4. What are some alternatives?\n\nNote: Yantra App List is available at https://github.com/besoeasy/yantra - when suggesting alternatives, prefer apps from this list as they're easy to install in Yantra.\n\nMake this a well-informed list, keep it short and minimal, and ask if I want to know more.`;
  
  return `https://chatgpt.com/?q=${encodeURIComponent(query)}`;
});

// Functions
async function fetchApp() {
  try {
    const response = await fetch(`${apiUrl.value}/api/apps`);
    const data = await response.json();
    
    if (data.success && data.apps) {
      app.value = data.apps.find((a) => a.id === route.params.appname);
      
      if (!app.value) {
        toast.error("App not found");
        router.push("/apps");
      }
    } else {
      throw new Error("Failed to load apps");
    }
  } catch (error) {
    console.error("Error fetching app:", error);
    toast.error("Failed to load app details");
  }
}

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`);
    const data = await response.json();
    if (data.success) {
      containers.value = data.containers;
    }
  } catch (error) {
    console.error("Error fetching containers:", error);
  }
}

async function deployApp() {
  if (deploying.value) return;

  deploying.value = true;
  toast.info(`Deploying ${app.value.name}... This may take a few minutes.`);

  try {
    const requestBody = { 
      appId: app.value.id, 
      environment: envValues.value 
    };
    
    if (temporaryInstall.value) {
      requestBody.expiresIn = expirationHours.value;
    }

    const response = await fetch(`${apiUrl.value}/api/deploy`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(requestBody),
    });

    const result = await response.json();

    if (result.success) {
      if (result.temporary) {
        toast.success(`${app.value.name} deployed as temporary (expires in ${expirationHours.value}h)! 🎉`);
      } else {
        toast.success(`${app.value.name} installed successfully! 🎉`);
      }
      
      // Wait a moment then redirect to containers
      setTimeout(() => {
        router.push("/containers");
      }, 1500);
    } else {
      throw new Error(result.error || "Deployment failed");
    }
  } catch (error) {
    console.error("Deployment error:", error);
    if (error.message.includes("timeout")) {
      toast.error(`Deployment timeout - ${app.value.name} is taking longer than expected`);
    } else {
      toast.error(`Deployment failed: ${error.message}`);
    }
  } finally {
    deploying.value = false;
  }
}

// Lifecycle
onMounted(async () => {
  loading.value = true;
  await Promise.all([fetchApp(), fetchContainers()]);
  loading.value = false;
});
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <div class="max-w-5xl mx-auto p-4 sm:p-6 md:p-10 lg:p-12">
      <!-- Back Button -->
      <button
        @click="router.push('/apps')"
        class="inline-flex items-center gap-2 mb-6 text-gray-600 hover:text-gray-900 transition-colors group"
      >
        <ArrowLeft :size="20" class="group-hover:-translate-x-1 transition-transform" />
        <span class="font-medium">Back to Apps</span>
      </button>

      <!-- Loading State -->
      <div v-if="loading" class="text-center py-16">
        <div class="w-12 h-12 border-4 border-blue-600/30 border-t-blue-600 rounded-full animate-spin mx-auto mb-4"></div>
        <div class="text-gray-500 font-medium">Loading app details...</div>
      </div>

      <!-- App Detail -->
      <div v-else-if="app" class="space-y-6">
        <!-- Header Card -->
        <div class="bg-white rounded-2xl p-6 sm:p-8 shadow-sm border border-gray-100">
          <div class="flex items-start gap-6">
            <img
              :src="app.logo"
              :alt="app.name"
              class="w-20 h-20 sm:w-24 sm:h-24 object-contain flex-shrink-0"
            />
            
            <div class="flex-1">
              <div class="flex items-start justify-between gap-4 mb-3">
                <h1 class="text-3xl sm:text-4xl font-bold text-gray-900">{{ app.name }}</h1>
                
                <div v-if="isInstalled" class="flex items-center gap-2 px-3 py-1.5 bg-green-50 text-green-700 rounded-lg text-sm font-medium">
                  <Package :size="16" />
                  <span>Installed</span>
                </div>
              </div>

              <p class="text-gray-600 text-lg leading-relaxed mb-4">
                {{ app.description || "No description available" }}
              </p>

              <!-- Categories -->
              <div class="flex flex-wrap gap-2 mb-4">
                <span
                  v-for="cat in categories"
                  :key="cat"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-50 text-blue-700 text-sm font-medium"
                >
                  <Tag :size="14" />
                  {{ cat }}
                </span>
              </div>

              <!-- Links -->
              <div class="flex flex-wrap gap-3">
                <a
                  v-if="app.website"
                  :href="app.website"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gray-100 hover:bg-blue-100 text-gray-700 hover:text-blue-700 transition-all font-medium text-sm"
                >
                  <Globe :size="18" />
                  <span>Visit Website</span>
                </a>
                <a
                  :href="`https://github.com/besoeasy/yantra/blob/main/apps/${app.id}/compose.yml`"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gray-100 hover:bg-purple-100 text-gray-700 hover:text-purple-700 transition-all font-medium text-sm"
                >
                  <FileCode :size="18" />
                  <span>View Source</span>
                </a>
                <a
                  :href="chatGptUrl"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-green-50 to-emerald-50 hover:from-green-100 hover:to-emerald-100 text-green-700 hover:text-green-800 transition-all font-medium text-sm border border-green-200/50"
                >
                  <span class="text-base">🤖</span>
                  <span>Ask ChatGPT</span>
                </a>
              </div>
            </div>
          </div>
        </div>

        <!-- Ports Information -->
        <div v-if="ports.length > 0" class="bg-white rounded-2xl p-6 sm:p-8 shadow-sm border border-gray-100">
          <h2 class="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
            <Package :size="24" />
            <span>Ports & Access</span>
          </h2>
          <div class="space-y-2">
            <div
              v-for="(port, idx) in ports"
              :key="idx"
              class="flex items-center gap-3 p-3 rounded-lg bg-gray-50 border border-gray-100"
            >
              <div class="w-2 h-2 rounded-full bg-green-500"></div>
              <span class="font-mono text-sm text-gray-900">{{ port }}</span>
            </div>
          </div>
        </div>

        <!-- Installation Form -->
        <div v-if="!isInstalled" class="bg-white rounded-2xl p-6 sm:p-8 shadow-sm border border-gray-100">
          <h2 class="text-xl font-bold text-gray-900 mb-6">Installation</h2>

          <div class="space-y-6">
            <!-- Environment Variables -->
            <div v-if="app.environment?.length > 0" class="space-y-4">
              <h3 class="text-sm font-bold text-gray-700 uppercase tracking-wide">Environment Variables</h3>
              <div v-for="env in app.environment" :key="env.envVar" class="space-y-2">
                <label class="block text-sm font-semibold text-gray-700">
                  {{ env.name }}
                  <span v-if="env.default" class="text-gray-400 text-xs font-normal ml-1">(default: {{ env.default }})</span>
                </label>
                <input
                  v-model="envValues[env.envVar]"
                  type="text"
                  :placeholder="env.default || `Enter ${env.name.toLowerCase()}`"
                  class="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-gray-900 placeholder-gray-400 transition-all text-base focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <p v-if="env.description" class="text-xs text-gray-500">{{ env.description }}</p>
              </div>
            </div>

            <!-- Temporary Installation -->
            <div :class="app.environment?.length > 0 ? 'pt-6 border-t border-gray-200' : ''">
              <label class="flex items-start gap-3 cursor-pointer group">
                <input
                  type="checkbox"
                  v-model="temporaryInstall"
                  class="w-5 h-5 mt-0.5 rounded border-gray-300 text-blue-600 focus:ring-2 focus:ring-blue-500 cursor-pointer"
                />
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <Clock :size="18" class="text-gray-600" />
                    <span class="text-base font-semibold text-gray-700 group-hover:text-blue-600 transition-colors">
                      Temporary Installation
                    </span>
                  </div>
                  <p class="text-sm text-gray-500 mt-1">
                    Automatically remove this app after a specified time period
                  </p>
                </div>
              </label>

              <div v-if="temporaryInstall" class="mt-4 ml-8 space-y-2 animate-fadeIn">
                <label class="block text-sm font-medium text-gray-700">Expires in</label>
                <select
                  v-model.number="expirationHours"
                  class="w-full px-4 py-3 bg-gray-50 border border-gray-200 rounded-xl text-gray-900 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option :value="1">1 hour</option>
                  <option :value="6">6 hours</option>
                  <option :value="12">12 hours</option>
                  <option :value="24">24 hours (1 day)</option>
                  <option :value="72">72 hours (3 days)</option>
                  <option :value="168">168 hours (1 week)</option>
                </select>
              </div>
            </div>

            <!-- Deploy Button -->
            <div class="pt-6">
              <button
                @click="deployApp"
                :disabled="deploying"
                :class="deploying ? 'bg-gray-400 cursor-not-allowed' : 'bg-blue-600 hover:bg-blue-700 active:scale-95'"
                class="w-full px-6 py-4 text-white rounded-xl text-lg font-bold transition-all shadow-md hover:shadow-lg disabled:cursor-not-allowed touch-manipulation"
              >
                <span v-if="deploying" class="inline-flex items-center gap-3 justify-center">
                  <span class="w-5 h-5 border-3 border-white/30 border-t-white rounded-full animate-spin"></span>
                  <span>Installing {{ app.name }}...</span>
                </span>
                <span v-else>Install {{ app.name }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Already Installed Message -->
        <div v-else class="bg-green-50 border border-green-200 rounded-2xl p-6 sm:p-8">
          <div class="flex items-start gap-4">
            <div class="flex-shrink-0">
              <div class="w-12 h-12 rounded-full bg-green-100 flex items-center justify-center">
                <Package :size="24" class="text-green-600" />
              </div>
            </div>
            <div class="flex-1">
              <h3 class="text-lg font-bold text-green-900 mb-2">Already Installed</h3>
              <p class="text-green-700 mb-4">
                This app is already running on your system. You can manage it from the Containers page.
              </p>
              <button
                @click="router.push('/containers')"
                class="inline-flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-xl font-semibold transition-all"
              >
                <span>View Containers</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
  animation: fadeIn 0.3s ease-out forwards;
}
</style>
