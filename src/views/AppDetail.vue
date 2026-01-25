<script setup>
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useToast } from "vue-toastification";
import { Globe, FileCode, ArrowLeft, Package, Clock, Tag } from "lucide-vue-next";
import { buildChatGptExplainUrl } from "../utils/chatgpt";

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
  return app.value.ports.filter((p) => !p.isNamed);
});

const allPorts = computed(() => {
  // For customization, show ALL ports (both named and unnamed)
  if (!app.value?.ports) return [];
  return app.value.ports;
});

// Port conflict detection
function checkPortConflict(hostPort, protocol) {
  return containers.value.find((c) => c.Ports?.some((p) => p.PublicPort === parseInt(hostPort) && p.Type === protocol));
}

function getPortStatus(port) {
  const hostPort = customPortMappings.value[port.hostPort + "/" + port.protocol] || port.hostPort;
  const conflict = checkPortConflict(hostPort, port.protocol);

  if (conflict) {
    return {
      status: "conflict",
      color: "red",
      message: `Port in use by ${conflict.Names?.[0]?.replace(/^\//, "") || "another container"}`,
    };
  }

  if (parseInt(hostPort) < 1024) {
    return {
      status: "warning",
      color: "yellow",
      message: "Privileged port (requires root)",
    };
  }

  return {
    status: "available",
    color: "green",
    message: "Available",
  };
}

const categories = computed(() => {
  if (!app.value?.category) return [];
  return app.value.category.split(",").map((c) => c.trim());
});

const chatGptUrl = computed(() => {
  if (!app.value) return "";

  return buildChatGptExplainUrl(app.value.id);
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
    allPorts.value.forEach((port) => {
      const status = getPortStatus(port);
      if (status.status === "conflict") {
        conflicts.push(`${port.hostPort}/${port.protocol}: ${status.message}`);
      }
    });

    if (conflicts.length > 0) {
      toast.error(`Port conflicts detected:\n${conflicts.join("\n")}`);
      return;
    }
  }

  deploying.value = true;
  const instanceNum = nextInstanceNumber.value;
  const instanceSuffix = instanceNum > 1 ? ` #${instanceNum}` : "";
  toast.info(`Deploying ${app.value.name}${instanceSuffix}... This may take a few minutes.`);

  try {
    const requestBody = {
      appId: app.value.id,
      environment: envValues.value,
      instanceId: instanceNum, // Pass instance number to backend
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
  <div class="min-h-screen bg-slate-50 dark:bg-[#09090b] text-slate-900 dark:text-slate-200 font-sans pb-32">
    
    <!-- Top Navigation -->
    <nav class="sticky top-0 z-40 bg-white/80 dark:bg-[#09090b]/90 backdrop-blur-md border-b border-slate-200 dark:border-slate-800">
      <div class="max-w-7xl mx-auto px-4 lg:px-8 h-16 flex items-center justify-between">
        <button
          @click="router.push('/apps')"
          class="flex items-center gap-2 text-sm font-semibold tracking-wide uppercase text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white transition-colors"
        >
          <ArrowLeft :size="18" />
          <span>Catalog</span>
        </button>
        
        <div v-if="app" class="flex items-center gap-3">
          <span class="text-xs font-mono text-slate-400 dark:text-slate-600">{{ app.id }}</span>
          <div v-if="isInstalled" class="flex items-center gap-1.5 px-2 py-0.5 bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs font-bold uppercase tracking-wider">
            <div class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></div>
            Installed
          </div>
        </div>
      </div>
    </nav>

    <!-- Loading State -->
    <div v-if="loading" class="flex flex-col items-center justify-center min-h-[60vh]">
      <div class="relative">
        <div class="w-16 h-16 border-2 border-slate-200 dark:border-slate-800 rounded-none transform rotate-45"></div>
        <div class="absolute inset-0 w-16 h-16 border-2 border-sky-500 border-t-transparent border-l-transparent rounded-none animate-spin"></div>
      </div>
      <div class="mt-8 font-mono text-sm tracking-widest text-slate-400 uppercase">Retrieving Manifest...</div>
    </div>

    <div v-else-if="app" class="max-w-7xl mx-auto px-4 lg:px-8 py-8 lg:py-12">
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-10">
        
        <!-- Left Column: Information & Specs -->
        <div class="lg:col-span-8 space-y-10">
          
          <!-- Identity Card -->
          <div class="flex flex-col sm:flex-row gap-6 sm:gap-8 border-b border-dashed border-slate-300 dark:border-slate-800 pb-10">
            <div class="w-24 h-24 sm:w-32 sm:h-32 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-4 flex items-center justify-center shrink-0">
              <img :src="app.logo" :alt="app.name" class="w-full h-full object-contain" />
            </div>
            
            <div class="flex-1 space-y-4">
              <div>
                <h1 class="text-3xl sm:text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white mb-2">{{ app.name }}</h1>
                <div class="flex flex-wrap gap-2">
                  <span
                    v-for="cat in categories"
                    :key="cat"
                    class="px-2 py-0.5 bg-slate-100 dark:bg-slate-800/50 text-slate-600 dark:text-slate-400 text-xs font-mono border border-slate-200 dark:border-slate-700 uppercase"
                  >
                    {{ cat }}
                  </span>
                </div>
              </div>
              
              <p class="text-base sm:text-lg leading-relaxed text-slate-600 dark:text-slate-300 max-w-3xl">
                {{ app.description || "No description available for this application." }}
              </p>

              <!-- Action Links -->
              <div class="flex flex-wrap gap-3 pt-2">
                <a
                  v-if="app.website"
                  :href="app.website"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-sky-500 hover:text-sky-600 dark:hover:border-sky-500 dark:hover:text-sky-400 transition-colors text-sm font-semibold"
                >
                  <Globe :size="16" />
                  Website
                </a>
                <a
                  :href="`https://github.com/besoeasy/yantra/blob/main/apps/${app.id}/compose.yml`"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-purple-500 hover:text-purple-600 dark:hover:border-purple-500 dark:hover:text-purple-400 transition-colors text-sm font-semibold"
                >
                  <FileCode :size="16" />
                  Source
                </a>
                <a
                  :href="chatGptUrl"
                  target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2 bg-emerald-50 dark:bg-emerald-950/20 border border-emerald-200 dark:border-emerald-800/50 text-emerald-700 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-900/30 transition-colors text-sm font-semibold"
                >
                  <span class="text-lg leading-none">🤖</span>
                  Explain
                </a>
              </div>
            </div>
          </div>

          <!-- Technical Specs -->
          <div class="space-y-6">
            <h3 class="text-sm font-mono uppercase tracking-widest text-slate-500 dark:text-slate-500 border-l-2 border-sky-500 pl-3">Technical Specifications</h3>
            
            <!-- Ports Grid -->
            <div v-if="ports.length > 0 || fixedPorts.length > 0" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
               <div class="border border-slate-200 dark:border-slate-800 bg-white dark:bg-[#0c0c0e] p-5">
                  <div class="flex items-center gap-2 mb-4 text-slate-900 dark:text-white font-bold">
                    <Package :size="18" class="text-sky-500" />
                    <span>Mapped Ports</span>
                  </div>
                  <ul class="space-y-2 font-mono text-sm leading-relaxed">
                     <li v-for="(port, idx) in ports" :key="'named-'+idx" class="flex items-center justify-between border-b border-dashed border-slate-100 dark:border-slate-800/50 pb-1 last:border-0">
                       <span class="text-slate-500">Service</span>
                       <span class="text-slate-900 dark:text-slate-200">{{ port }}</span>
                     </li>
                     <li v-for="(port, idx) in fixedPorts" :key="'fixed-'+idx" class="flex items-center justify-between border-b border-dashed border-slate-100 dark:border-slate-800/50 pb-1 last:border-0">
                        <span class="text-slate-500">{{ port.containerPort }}/{{ port.protocol }}</span>
                        <span class="text-slate-900 dark:text-slate-200">→ {{ port.hostPort }}</span>
                     </li>
                  </ul>
               </div>
            </div>
          </div>

          <!-- Image Details -->
          <div v-if="imageDetails && imageDetails.length > 0" class="space-y-6">
             <div class="flex items-baseline justify-between">
                <h3 class="text-sm font-mono uppercase tracking-widest text-slate-500 dark:text-slate-500 border-l-2 border-purple-500 pl-3">Image Manifest</h3>
                <span class="text-xs font-mono text-slate-400">{{ imageDetails.length }} image{{ imageDetails.length !== 1 ? "s" : "" }}</span>
             </div>

             <div class="grid gap-4">
                <div v-for="img in imageDetails" :key="img.id" class="group relative bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-5 hover:border-slate-300 dark:hover:border-slate-600 transition-colors">
                   <div class="absolute top-0 right-0 p-2 opacity-50 font-mono text-[10px] text-slate-400">{{ img.shortId }}</div>
                   
                   <div class="mb-4">
                      <div class="flex flex-wrap gap-2">
                        <span v-for="tag in img.tags" :key="tag" class="px-2 py-0.5 bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700 text-xs font-mono">
                          {{ tag }}
                        </span>
                      </div>
                   </div>

                   <div class="grid grid-cols-2 md:grid-cols-4 gap-6 text-sm">
                      <div>
                         <div class="text-[10px] uppercase tracking-wider text-slate-500 dark:text-slate-500 mb-1">Architecture</div>
                         <div class="font-mono text-slate-900 dark:text-slate-200">{{ img.architecture }} / {{ img.os }}</div>
                      </div>
                      <div>
                         <div class="text-[10px] uppercase tracking-wider text-slate-500 dark:text-slate-500 mb-1">Size</div>
                         <div class="font-mono text-slate-900 dark:text-slate-200">{{ img.size }} MB</div>
                      </div>
                      <div>
                         <div class="text-[10px] uppercase tracking-wider text-slate-500 dark:text-slate-500 mb-1">Created</div>
                         <div class="font-mono text-slate-900 dark:text-slate-200 truncate" :title="img.createdDate">{{ img.relativeTime }}</div>
                      </div>
                      <div>
                          <div class="text-[10px] uppercase tracking-wider text-slate-500 dark:text-slate-500 mb-1">Digest</div>
                          <div class="font-mono text-slate-900 dark:text-slate-200 truncate opacity-60" :title="img.digest">{{ img.digest.substring(7, 19) }}...</div>
                      </div>
                   </div>
                </div>
             </div>
          </div>
          
        </div>

        <!-- Right Column: Deployment Configuration -->
        <div class="lg:col-span-4">
          <div class="sticky top-24 bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-6 lg:p-8 shadow-2xl shadow-slate-200/50 dark:shadow-black/50">
            <h2 class="text-xl font-bold text-slate-900 dark:text-white mb-6 uppercase tracking-tight flex items-center gap-3">
              <span class="w-2 h-8 bg-sky-500 block"></span>
              Deploy Config
            </h2>
            
            <div class="space-y-6">
              <!-- Environment Vars -->
              <div v-if="app.environment?.length > 0" class="space-y-4">
                <div v-for="env in app.environment" :key="env.envVar" class="space-y-1">
                  <label class="block text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wide">
                    {{ env.name }}
                    <span v-if="env.default" class="text-slate-400 dark:text-slate-600 font-normal ml-1">({{ env.default }})</span>
                  </label>
                  <input
                    v-model="envValues[env.envVar]"
                    type="text"
                    :placeholder="env.default || 'Value'"
                    class="w-full bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 p-3 text-sm font-mono text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-sky-500 dark:focus:border-sky-500 transition-colors"
                  />
                  <p v-if="env.description" class="text-[10px] text-slate-500">{{ env.description }}</p>
                </div>
              </div>

              <!-- Options Toggles -->
              <div class="space-y-3 pt-4 border-t border-slate-100 dark:border-slate-800">
                
                <!-- Temporary Install -->
                <div class="flex items-start gap-3">
                  <input type="checkbox" id="temp-install" v-model="temporaryInstall" class="mt-1 w-4 h-4 rounded-none border-slate-300 text-sky-600 focus:ring-offset-0 focus:ring-0 cursor-pointer" />
                  <div class="flex-1">
                    <label for="temp-install" class="block text-sm font-bold text-slate-700 dark:text-slate-200 cursor-pointer uppercase">Temporary Install</label>
                    <p class="text-xs text-slate-500 mt-0.5">Auto-remove container after set duration.</p>
                  </div>
                </div>
                 
                <div v-if="temporaryInstall" class="pl-7">
                  <select v-model.number="expirationHours" class="w-full bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 p-2 text-sm font-mono focus:border-sky-500 focus:outline-none">
                    <option :value="1">1 Hour</option>
                    <option :value="6">6 Hours</option>
                    <option :value="12">12 Hours</option>
                    <option :value="24">1 Day</option>
                    <option :value="72">3 Days</option>
                  </select>
                </div>

                <!-- Custom Ports -->
                <div v-if="allPorts.length > 0" class="flex items-start gap-3 pt-2">
                   <input type="checkbox" id="custom-ports" v-model="customizePorts" class="mt-1 w-4 h-4 rounded-none border-slate-300 text-sky-600 focus:ring-offset-0 focus:ring-0 cursor-pointer" />
                   <div class="flex-1">
                     <label for="custom-ports" class="block text-sm font-bold text-slate-700 dark:text-slate-200 cursor-pointer uppercase">Advanced: Port Map</label>
                   </div>
                </div>

                <div v-if="customizePorts" class="pl-7 space-y-3">
                   <div v-for="port in allPorts" :key="port.hostPort + '/' + port.protocol" class="space-y-1">
                      <div class="flex items-center justify-between text-xs font-mono text-slate-500">
                         <span>Container</span>
                         <span>{{ port.protocol }}</span>
                      </div>
                      <div class="flex items-center gap-2">
                        <span class="text-xs font-mono text-slate-400">{{ port.containerPort }}</span>
                        <span class="text-slate-400">→</span>
                        <input
                          v-model="customPortMappings[port.hostPort + '/' + port.protocol]"
                          type="number"
                          :placeholder="port.hostPort"
                          class="flex-1 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 p-2 text-sm font-mono focus:border-sky-500 focus:outline-none"
                        />
                      </div>
                      <!-- Port Status Indicator -->
                       <div class="flex items-center gap-2 text-[10px] uppercase font-bold tracking-wider">
                          <span :class="{
                            'text-red-500': getPortStatus(port).status === 'conflict',
                            'text-yellow-500': getPortStatus(port).status === 'warning',
                            'text-emerald-500': getPortStatus(port).status === 'available'
                          }">{{ getPortStatus(port).message }}</span>
                       </div>
                   </div>
                </div>
              </div>

              <!-- Deploy Button -->
              <div class="pt-6">
                 <button
                   @click="deployApp"
                   :disabled="deploying"
                   class="w-full relative group overflow-hidden bg-slate-900 dark:bg-white text-white dark:text-slate-900 p-4 font-bold uppercase tracking-widest text-sm hover:bg-sky-600 dark:hover:bg-sky-400 dark:hover:text-white transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                 >
                    <span v-if="deploying" class="flex items-center justify-center gap-3">
                       <span class="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin"></span>
                       Initializing...
                    </span>
                    <span v-else class="relative z-10 flex items-center justify-center gap-2">
                       {{ instanceCount > 0 ? 'Deploy New Instance' : 'Initialize Deployment' }}
                    </span>
                 </button>
                 <div v-if="instanceCount > 0" class="text-center mt-3 text-xs font-mono text-slate-500 dark:text-slate-500">
                    Active Instances: {{ instanceCount }}
                 </div>
              </div>

            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<style scoped></style>
