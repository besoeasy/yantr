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
const customizePorts = ref(false);
const customPortMappings = ref({});
const imageDetails = ref(null);
const loadingImages = ref(false);

// Computed
const isInstalled = computed(() => {
  return containers.value.some((c) => c.app.id === route.params.appname);
});

const instanceCount = computed(() => {
  return containers.value.filter((c) => c.app.id === route.params.appname).length;
});

const nextInstanceNumber = computed(() => {
  return instanceCount.value + 1;
});

const ports = computed(() => {
  if (!app.value?.port) return [];
  return app.value.port.split(",").map((p) => p.trim());
});

const fixedPorts = computed(() => {
  if (!app.value?.ports) return [];
  // Filter out ports that are already described in yantra.port (isNamed)
  return app.value.ports.filter(p => !p.isNamed);
});

const allPorts = computed(() => {
  // For customization, show ALL ports (both named and unnamed)
  if (!app.value?.ports) return [];
  return app.value.ports;
});

// Port conflict detection
function checkPortConflict(hostPort, protocol) {
  return containers.value.find(c => 
    c.Ports?.some(p => 
      p.PublicPort === parseInt(hostPort) && 
      p.Type === protocol
    )
  );
}

function getPortStatus(port) {
  const hostPort = customPortMappings.value[port.hostPort + '/' + port.protocol] || port.hostPort;
  const conflict = checkPortConflict(hostPort, port.protocol);
  
  if (conflict) {
    return {
      status: 'conflict',
      color: 'red',
      message: `Port in use by ${conflict.Names?.[0]?.replace(/^\//, '') || 'another container'}`
    };
  }
  
  if (parseInt(hostPort) < 1024) {
    return {
      status: 'warning',
      color: 'yellow',
      message: 'Privileged port (requires root)'
    };
  }
  
  return {
    status: 'available',
    color: 'green',
    message: 'Available'
  };
}

const categories = computed(() => {
  if (!app.value?.category) return [];
  return app.value.category.split(",").map((c) => c.trim());
});

const chatGptUrl = computed(() => {
  if (!app.value) return '';
  
  const composeUrl = `https://github.com/besoeasy/yantra/blob/main/apps/${app.value.id}/compose.yml`;
  const query = `Understand this Yantra Docker stack:\n${composeUrl}\n(Yantra handles deployment, so skip Docker/installation commands)\nTell me:\n1. What does this app do?\n2. 5 main features of this app\n\n3. What are some alternatives?\n\nNote: Yantra App List is available at https://github.com/besoeasy/yantra - when suggesting alternatives, prefer apps from this list as they're easy to install in Yantra.\n\nMake this a well-informed list, keep it short and minimal, and ask if I want to know more.`;
  
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

async function fetchImageDetails() {
  if (!app.value) return;
  
  try {
    loadingImages.value = true;
    const response = await fetch(`${apiUrl.value}/api/image-details/${app.value.id}`);
    const data = await response.json();
    
    if (data.success) {
      imageDetails.value = data.images;
    }
  } catch (error) {
    console.error("Error fetching image details:", error);
  } finally {
    loadingImages.value = false;
  }
}

async function deployApp() {
  if (deploying.value) return;

  // Check for port conflicts if customizing ports
  if (customizePorts.value) {
    const conflicts = [];
    allPorts.value.forEach(port => {
      const status = getPortStatus(port);
      if (status.status === 'conflict') {
        conflicts.push(`${port.hostPort}/${port.protocol}: ${status.message}`);
      }
    });
    
    if (conflicts.length > 0) {
      toast.error(`Port conflicts detected:\n${conflicts.join('\n')}`);
      return;
    }
  }

  deploying.value = true;
  const instanceNum = nextInstanceNumber.value;
  const instanceSuffix = instanceNum > 1 ? ` #${instanceNum}` : '';
  toast.info(`Deploying ${app.value.name}${instanceSuffix}... This may take a few minutes.`);

  try {
    const requestBody = { 
      appId: app.value.id, 
      environment: envValues.value,
      instanceId: instanceNum // Pass instance number to backend
    };
    
    if (temporaryInstall.value) {
      requestBody.expiresIn = expirationHours.value;
    }
    
    if (customizePorts.value && Object.keys(customPortMappings.value).length > 0) {
      requestBody.customPortMappings = customPortMappings.value;
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
        router.push("/");
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
  await fetchImageDetails();
  loading.value = false;
});
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-slate-950">
    <div class="max-w-5xl mx-auto p-4 sm:p-6 md:p-10 lg:p-12">
      <!-- Back Button -->
      <button
        @click="router.push('/apps')"
        class="inline-flex items-center gap-2 mb-6 text-gray-600 hover:text-gray-900 dark:text-slate-400 dark:hover:text-slate-200 transition-colors group"
      >
        <ArrowLeft :size="20" class="group-hover:-translate-x-1 transition-transform" />
        <span class="font-medium">Back to Apps</span>
      </button>

      <!-- Loading State -->
      <div v-if="loading" class="text-center py-16">
        <div class="w-12 h-12 border-4 border-blue-600/30 border-t-blue-600 dark:border-blue-500/30 dark:border-t-blue-400 rounded-full animate-spin mx-auto mb-4"></div>
        <div class="text-gray-500 dark:text-slate-400 font-medium">Loading app details...</div>
      </div>

      <!-- App Detail -->
      <div v-else-if="app" class="space-y-6">
        <!-- Header Card -->
        <div class="bg-white dark:bg-slate-900/70 rounded-2xl p-6 sm:p-8 shadow-sm border border-gray-100 dark:border-slate-800">
          <div class="flex items-start gap-6">
            <img
              :src="app.logo"
              :alt="app.name"
              class="w-20 h-20 sm:w-24 sm:h-24 object-contain flex-shrink-0"
            />
            
            <div class="flex-1">
              <div class="flex items-start justify-between gap-4 mb-3">
                <h1 class="text-3xl sm:text-4xl font-bold text-gray-900 dark:text-white">{{ app.name }}</h1>
                
                <div v-if="isInstalled" class="flex items-center gap-2 px-3 py-1.5 bg-green-50 dark:bg-emerald-500/15 text-green-700 dark:text-emerald-200 rounded-lg text-sm font-medium">
                  <Package :size="16" />
                  <span>Installed</span>
                </div>
              </div>

              <p class="text-gray-600 dark:text-slate-300 text-lg leading-relaxed mb-4">
                {{ app.description || "No description available" }}
              </p>

              <!-- Categories -->
              <div class="flex flex-wrap gap-2 mb-4">
                <span
                  v-for="cat in categories"
                  :key="cat"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-50 dark:bg-blue-500/15 text-blue-700 dark:text-blue-200 text-sm font-medium"
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
                  class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gray-100 hover:bg-blue-100 text-gray-700 hover:text-blue-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 dark:hover:text-blue-300 transition-all font-medium text-sm"
                >
                  <Globe :size="18" />
                  <span>Visit Website</span>
                </a>
                <a
                  :href="`https://github.com/besoeasy/yantra/blob/main/apps/${app.id}/compose.yml`"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gray-100 hover:bg-purple-100 text-gray-700 hover:text-purple-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 dark:hover:text-purple-300 transition-all font-medium text-sm"
                >
                  <FileCode :size="18" />
                  <span>View Source</span>
                </a>
                <a
                  :href="chatGptUrl"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-green-50 to-emerald-50 hover:from-green-100 hover:to-emerald-100 dark:from-emerald-500/10 dark:to-green-500/10 dark:hover:from-emerald-500/20 dark:hover:to-green-500/20 text-green-700 dark:text-emerald-200 hover:text-green-800 transition-all font-medium text-sm border border-green-200/50 dark:border-emerald-500/30"
                >
                  <span class="text-base">🤖</span>
                  <span>Ask ChatGPT</span>
                </a>
              </div>
            </div>
          </div>
        </div>

        <!-- Ports Information -->
        <div v-if="ports.length > 0 || fixedPorts.length > 0" class="bg-white dark:bg-slate-900/70 rounded-2xl p-6 sm:p-8 shadow-sm border border-gray-100 dark:border-slate-800">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
            <Package :size="24" />
            <span>Ports & Access</span>
          </h2>
          <div class="space-y-2">
            <!-- Named ports (from yantra.port label) -->
            <div
              v-for="(port, idx) in ports"
              :key="'named-' + idx"
              class="flex items-center gap-3 p-3 rounded-lg bg-gray-50 dark:bg-slate-800 border border-gray-100 dark:border-slate-700"
            >
              <div class="w-2 h-2 rounded-full bg-green-500"></div>
              <span class="font-mono text-sm text-gray-900 dark:text-slate-100">{{ port }}</span>
            </div>
            <!-- Additional fixed ports (from compose.yml) -->
            <div
              v-for="(port, idx) in fixedPorts"
              :key="'fixed-' + idx"
              class="flex items-center gap-3 p-3 rounded-lg bg-gray-50 dark:bg-slate-800 border border-gray-100 dark:border-slate-700"
            >
              <div class="w-2 h-2 rounded-full bg-blue-500"></div>
              <span class="font-mono text-sm text-gray-900 dark:text-slate-100">{{ port.hostPort }}:{{ port.containerPort }}/{{ port.protocol }}</span>
            </div>
          </div>
        </div>

        <!-- Docker Image Details -->
        <div v-if="imageDetails && imageDetails.length > 0" class="bg-white dark:bg-slate-900/70 rounded-2xl p-6 sm:p-8 shadow-sm border border-gray-100 dark:border-slate-800">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-6 flex items-center gap-2">
            <Package :size="24" />
            <span>Docker Image Details</span>
            <span v-if="imageDetails.length > 1" class="ml-auto text-sm font-normal text-gray-500 dark:text-slate-400">({{ imageDetails.length }} image{{ imageDetails.length !== 1 ? 's' : '' }})</span>
          </h2>

          <div v-if="loadingImages" class="text-center py-8">
            <div class="w-8 h-8 border-3 border-gray-300 border-t-gray-900 dark:border-slate-600 dark:border-t-slate-200 rounded-full animate-spin mx-auto mb-2"></div>
            <div class="text-sm text-gray-500 dark:text-slate-400">Loading image details...</div>
          </div>

          <div v-else class="space-y-4">
            <div
              v-for="(img, idx) in imageDetails"
              :key="img.id"
              class="border border-gray-200 dark:border-slate-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-slate-700 hover:shadow-md transition-all"
            >
              <!-- Image Tags -->
              <div class="mb-4 pb-4 border-b border-gray-100 dark:border-slate-800">
                <div class="flex flex-wrap gap-2 items-center">
                  <span class="text-xs font-semibold text-gray-500 dark:text-slate-400 uppercase">Tags:</span>
                  <div class="flex flex-wrap gap-2">
                    <span
                      v-for="(tag, tIdx) in img.tags"
                      :key="'tag-' + tIdx"
                      class="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-mono bg-gray-100 dark:bg-slate-800 text-gray-700 dark:text-slate-200 border border-gray-200 dark:border-slate-700"
                    >
                      {{ tag }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Image Metadata Grid -->
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <!-- Created Date -->
                <div class="flex items-start gap-3">
                  <Clock :size="16" class="text-gray-400 dark:text-slate-400 flex-shrink-0 mt-1" />
                  <div class="flex-1 min-w-0">
                    <div class="text-xs text-gray-500 dark:text-slate-400 font-semibold uppercase mb-1">Published</div>
                    <div class="text-sm text-gray-900 dark:text-slate-100 font-mono">{{ img.createdDate }}</div>
                    <div class="text-xs text-gray-500 dark:text-slate-400 mt-0.5">{{ img.relativeTime }}</div>
                  </div>
                </div>

                <!-- Architecture -->
                <div class="flex items-start gap-3">
                  <Package :size="16" class="text-gray-400 dark:text-slate-400 flex-shrink-0 mt-1" />
                  <div class="flex-1 min-w-0">
                    <div class="text-xs text-gray-500 dark:text-slate-400 font-semibold uppercase mb-1">Architecture</div>
                    <div class="text-sm text-gray-900 dark:text-slate-100 font-mono">{{ img.architecture }}/{{ img.os }}</div>
                  </div>
                </div>

                <!-- Image Size -->
                <div class="flex items-start gap-3">
                  <Package :size="16" class="text-gray-400 dark:text-slate-400 flex-shrink-0 mt-1" />
                  <div class="flex-1 min-w-0">
                    <div class="text-xs text-gray-500 dark:text-slate-400 font-semibold uppercase mb-1">Size</div>
                    <div class="text-sm text-gray-900 dark:text-slate-100 font-mono">{{ img.size }} MB</div>
                  </div>
                </div>

                <!-- Image ID -->
                <div class="flex items-start gap-3">
                  <FileCode :size="16" class="text-gray-400 dark:text-slate-400 flex-shrink-0 mt-1" />
                  <div class="flex-1 min-w-0">
                    <div class="text-xs text-gray-500 dark:text-slate-400 font-semibold uppercase mb-1">Image ID</div>
                    <div class="text-sm text-gray-900 dark:text-slate-100 font-mono">{{ img.shortId }}</div>
                  </div>
                </div>
              </div>

              <!-- SHA256 Digest -->
              <div v-if="img.digest && img.digest !== 'N/A'" class="mt-4 pt-4 border-t border-gray-100 dark:border-slate-800">
                <div class="text-xs text-gray-500 dark:text-slate-400 font-semibold uppercase mb-2">SHA256 Digest</div>
                <div class="p-3 bg-gray-50 dark:bg-slate-800 rounded-lg border border-gray-100 dark:border-slate-700 overflow-x-auto">
                  <code class="text-xs text-gray-700 dark:text-slate-200 font-mono break-all">{{ img.digest }}</code>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Installation Form -->
        <div class="bg-white dark:bg-slate-900/70 rounded-2xl p-6 sm:p-8 shadow-sm border border-gray-100 dark:border-slate-800">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-6">
            {{ instanceCount > 0 ? 'Install Another Instance' : 'Installation' }}
          </h2>

          <div class="space-y-6">
            <!-- Environment Variables -->
            <div v-if="app.environment?.length > 0" class="space-y-4">
              <h3 class="text-sm font-bold text-gray-700 dark:text-slate-300 uppercase tracking-wide">Environment Variables</h3>
              <div v-for="env in app.environment" :key="env.envVar" class="space-y-2">
                <label class="block text-sm font-semibold text-gray-700 dark:text-slate-300">
                  {{ env.name }}
                  <span v-if="env.default" class="text-gray-400 dark:text-slate-500 text-xs font-normal ml-1">(default: {{ env.default }})</span>
                </label>
                <input
                  v-model="envValues[env.envVar]"
                  type="text"
                  :placeholder="env.default || `Enter ${env.name.toLowerCase()}`"
                  class="w-full px-4 py-3 bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-slate-700 rounded-xl text-gray-900 dark:text-slate-100 placeholder-gray-400 dark:placeholder-slate-500 transition-all text-base focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <p v-if="env.description" class="text-xs text-gray-500 dark:text-slate-400">{{ env.description }}</p>
              </div>
            </div>

            <!-- Temporary Installation -->
            <div :class="app.environment?.length > 0 ? 'pt-6 border-t border-gray-200 dark:border-slate-800' : ''">
              <label class="flex items-start gap-3 cursor-pointer group">
                <input
                  type="checkbox"
                  v-model="temporaryInstall"
                  class="w-5 h-5 mt-0.5 rounded border-gray-300 dark:border-slate-600 text-blue-600 focus:ring-2 focus:ring-blue-500 cursor-pointer"
                />
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <Clock :size="18" class="text-gray-600 dark:text-slate-300" />
                    <span class="text-base font-semibold text-gray-700 dark:text-slate-200 group-hover:text-blue-600 dark:group-hover:text-blue-300 transition-colors">
                      Temporary Installation
                    </span>
                  </div>
                  <p class="text-sm text-gray-500 dark:text-slate-400 mt-1">
                    Automatically remove this app after a specified time period
                  </p>
                </div>
              </label>

              <div v-if="temporaryInstall" class="mt-4 ml-8 space-y-2 animate-fadeIn">
                <label class="block text-sm font-medium text-gray-700 dark:text-slate-300">Expires in</label>
                <select
                  v-model.number="expirationHours"
                  class="w-full px-4 py-3 bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-slate-700 rounded-xl text-gray-900 dark:text-slate-100 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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

            <!-- Port Customization -->
            <div v-if="allPorts.length > 0" :class="(app.environment?.length > 0 || temporaryInstall) ? 'pt-6 border-t border-gray-200 dark:border-slate-800' : ''">
              <label class="flex items-start gap-3 cursor-pointer group">
                <input
                  type="checkbox"
                  v-model="customizePorts"
                  class="w-5 h-5 mt-0.5 rounded border-gray-300 dark:border-slate-600 text-blue-600 focus:ring-2 focus:ring-blue-500 cursor-pointer"
                />
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <Package :size="18" class="text-gray-600 dark:text-slate-300" />
                    <span class="text-base font-semibold text-gray-700 dark:text-slate-200 group-hover:text-blue-600 dark:group-hover:text-blue-300 transition-colors">
                      Advanced: Customize Port Mappings
                    </span>
                  </div>
                  <p class="text-sm text-gray-500 dark:text-slate-400 mt-1">
                    ⚠️ Only modify if you understand Docker port mappings or need to resolve conflicts
                  </p>
                </div>
              </label>

              <div v-if="customizePorts" class="mt-4 ml-8 space-y-3 animate-fadeIn">
                <div v-for="port in allPorts" :key="port.hostPort + '/' + port.protocol" class="space-y-2">
                  <label class="block text-sm font-medium text-gray-700 dark:text-slate-300">
                    Container Port: {{ port.containerPort }}/{{ port.protocol }}
                  </label>
                  <div class="flex items-center gap-3">
                    <input
                      v-model="customPortMappings[port.hostPort + '/' + port.protocol]"
                      type="number"
                      :placeholder="port.hostPort"
                      min="1"
                      max="65535"
                      class="flex-1 px-4 py-3 bg-gray-50 dark:bg-slate-800 border border-gray-200 dark:border-slate-700 rounded-xl text-gray-900 dark:text-slate-100 placeholder-gray-400 dark:placeholder-slate-500 transition-all text-base focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <div :class="`w-3 h-3 rounded-full bg-${getPortStatus(port).color}-500`"></div>
                  </div>
                  <p class="text-xs" :class="{
                    'text-red-600': getPortStatus(port).status === 'conflict',
                    'text-yellow-600': getPortStatus(port).status === 'warning',
                    'text-green-600': getPortStatus(port).status === 'available'
                  }">
                    {{ getPortStatus(port).message }}
                  </p>
                </div>
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
                  <span>Installing {{ app.name }}{{ instanceCount > 0 ? ` #${nextInstanceNumber}` : '' }}...</span>
                </span>
                <span v-else>
                  {{ instanceCount > 0 ? `Install Another Instance (#${nextInstanceNumber})` : `Install ${app.name}` }}
                </span>
              </button>
              <p v-if="instanceCount > 0" class="text-sm text-gray-500 dark:text-slate-400 mt-2 text-center">
                {{ instanceCount }} instance{{ instanceCount > 1 ? 's' : '' }} currently installed
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
