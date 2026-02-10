<script setup>
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useToast } from "vue-toastification";
import { Globe, FileCode, ArrowLeft, Package, Clock, Tag, ExternalLink, Activity, Info, AlertTriangle, Check, Terminal, Play, CreditCard, RotateCcw } from "lucide-vue-next";
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
      message: `Conflict: ${conflict.Names?.[0]?.replace(/^\//, "") || "Container"}`,
    };
  }

  if (parseInt(hostPort) < 1024) {
    return {
      status: "warning",
      color: "yellow",
      message: "Privileged (Root req.)",
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

const dependencies = computed(() => {
  return Array.isArray(app.value?.dependencies) ? app.value.dependencies : [];
});

const runningAppIds = computed(() => {
  return new Set(containers.value.filter((c) => c.state === "running").map((c) => c.app?.id).filter(Boolean));
});

const missingDependencies = computed(() => {
  return dependencies.value.filter((dep) => !runningAppIds.value.has(dep));
});

const canDeploy = computed(() => {
  return !deploying.value;
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

  let allowMissingDependencies = false;
  if (missingDependencies.value.length > 0) {
    const depApps = missingDependencies.value.join(", ");
    const proceed = confirm(
      `Missing dependencies: ${depApps}.\n\nYou can deploy anyway, but the app may not work correctly until they are running.\n\nDeploy anyway?`
    );

    if (!proceed) {
      toast.info("Deployment cancelled. Start required apps first to avoid issues.", {
        timeout: 4000
      });
      return;
    }

    allowMissingDependencies = true;
    toast.warning(`Deploying without dependencies: ${depApps}`, {
      timeout: 5000
    });
  }

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

    if (allowMissingDependencies) {
      requestBody.allowMissingDependencies = true;
    }

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
      // Check if it's a dependency error
      if (result.missingDependencies && result.missingDependencies.length > 0) {
        const deps = result.missingDependencies.join(", ");
        toast.error(`Missing dependencies: ${deps}. Click dependency badges below to deploy them first.`, {
          timeout: 5000
        });
      } else {
        throw new Error(result.message || result.error || "Deployment failed");
      }
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
  <div class="min-h-screen bg-slate-50 dark:bg-[#09090b] text-slate-900 dark:text-slate-200 font-sans">
    
    <!-- Header -->
    <header class="bg-white dark:bg-[#0c0c0e] border-b border-slate-200 dark:border-slate-800">
      <div class="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between">
        <div class="flex items-center gap-4">
            <button
            @click="router.push('/apps')"
            class="inline-flex items-center justify-center w-8 h-8 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors text-slate-500"
            >
            <ArrowLeft :size="18" />
            </button>
          
          <div class="h-4 w-px bg-slate-200 dark:bg-slate-800"></div>

          <div class="flex items-center gap-2 text-sm">
            <span class="text-slate-500">Catalog</span>
            <span class="text-slate-300 dark:text-slate-700">/</span>
            <span class="font-medium text-slate-900 dark:text-white" v-if="app">{{ app.name }}</span>
            <span v-else class="w-24 h-4 bg-slate-200 dark:bg-slate-800 animate-pulse rounded"></span>
          </div>
        </div>
        
        <div v-if="isInstalled" class="flex items-center gap-2 px-3 py-1 rounded-full border border-emerald-200 dark:border-emerald-800/50 bg-emerald-50 dark:bg-emerald-950/20 text-emerald-600 dark:text-emerald-400 text-xs font-semibold uppercase tracking-wide">
            <div class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></div>
            Installed
        </div>
      </div>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="flex flex-col items-center justify-center min-h-[60vh]">
      <div class="animate-spin text-slate-300"><Activity :size="32" /></div>
      <div class="mt-4 font-mono text-xs tracking-widest text-slate-400 uppercase">Retrieving Manifest...</div>
    </div>

    <div v-else-if="app" class="max-w-7xl mx-auto px-4 py-8 grid grid-cols-1 lg:grid-cols-12 gap-10">
        
        <!-- Left Column: Information & Specs -->
        <div class="lg:col-span-8 space-y-8">
          
          <!-- Identity Card -->
          <div class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 p-6 flex flex-col sm:flex-row gap-6">
            <div class="w-24 h-24 sm:w-32 sm:h-32 bg-slate-50 dark:bg-slate-900 border border-slate-100 dark:border-slate-800 rounded-lg flex items-center justify-center p-4 shrink-0">
              <img :src="app.logo" :alt="app.name" class="w-full h-full object-contain" />
            </div>
            
            <div class="flex-1 space-y-4">
              <div>
                <h1 class="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white mb-2">{{ app.name }}</h1>
                <div class="flex flex-wrap gap-2">
                  <span
                    v-for="cat in categories"
                    :key="cat"
                    class="px-2 py-0.5 bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 text-xs font-mono font-medium rounded border border-slate-200 dark:border-slate-700"
                  >
                    {{ cat }}
                  </span>
                </div>
              </div>
              
              <p class="text-sm leading-relaxed text-slate-600 dark:text-slate-300">
                {{ app.description || "No description available for this application." }}
              </p>

              <!-- Action Links -->
              <div class="flex flex-wrap gap-2 pt-2">
                <a
                  v-if="app.website"
                  :href="app.website"
                  target="_blank"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded text-slate-600 dark:text-slate-400 hover:border-blue-500 hover:text-blue-500 transition-colors text-xs font-medium uppercase tracking-wide"
                >
                  <Globe :size="14" />
                  Website
                </a>
                <a
                  :href="`https://github.com/besoeasy/yantra/blob/main/apps/${app.id}/compose.yml`"
                  target="_blank"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded text-slate-600 dark:text-slate-400 hover:border-purple-500 hover:text-purple-500 transition-colors text-xs font-medium uppercase tracking-wide"
                >
                  <FileCode :size="14" />
                  Source
                </a>
                <a
                  :href="chatGptUrl"
                  target="_blank"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-emerald-50 dark:bg-emerald-950/20 border border-emerald-200 dark:border-emerald-800/50 rounded text-emerald-600 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-900/30 transition-colors text-xs font-medium uppercase tracking-wide"
                >
                  <Info :size="14" />
                  Explain
                </a>
              </div>
            </div>
          </div>

          <!-- Technical Specs -->
          <div v-if="ports.length > 0 || fixedPorts.length > 0" class="space-y-4">
            <h3 class="text-sm font-semibold uppercase tracking-wider text-slate-500 px-1">Network Requirements</h3>
            
            <div class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
                <table class="w-full text-left text-sm">
                    <thead>
                        <tr class="bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-800 text-xs font-semibold uppercase text-slate-500">
                            <th class="px-4 py-3 font-medium">Type</th>
                            <th class="px-4 py-3 font-medium">Definition</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-100 dark:divide-slate-800/50">
                        <tr v-for="(port, idx) in ports" :key="'named-'+idx" class="hover:bg-slate-50 dark:hover:bg-slate-900/20">
                            <td class="px-4 py-3 text-slate-500 font-mono text-xs uppercase">Service</td>
                            <td class="px-4 py-3 font-mono text-slate-900 dark:text-white">{{ port }}</td>
                        </tr>
                        <tr v-for="(port, idx) in fixedPorts" :key="'fixed-'+idx" class="hover:bg-slate-50 dark:hover:bg-slate-900/20">
                            <td class="px-4 py-3 text-slate-500 font-mono text-xs uppercase">{{ port.protocol }} Mapping</td>
                            <td class="px-4 py-3 font-mono text-slate-900 dark:text-white flex items-center gap-2">
                                <span>{{ port.containerPort }}</span>
                                <span class="text-slate-400">→</span>
                                <span>{{ port.hostPort }}</span>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
          </div>

          <!-- Image Details -->
          <div v-if="imageDetails && imageDetails.length > 0" class="space-y-4">
             <div class="flex items-center justify-between px-1">
                <h3 class="text-sm font-semibold uppercase tracking-wider text-slate-500">Dependent Images</h3>
                <span class="text-xs font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">{{ imageDetails.length }}</span>
             </div>

             <div class="space-y-3">
                <div v-for="img in imageDetails" :key="img.id" class="bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 rounded-lg p-4 hover:border-blue-500/30 transition-colors">
                   <div class="flex items-center justify-between mb-3">
                       <div class="flex flex-wrap gap-2">
                         <div v-for="tag in img.tags" :key="tag" class="flex items-center gap-1.5 px-2 py-1 bg-slate-100 dark:bg-slate-900 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-800 rounded text-xs font-mono">
                           <Tag :size="12" class="text-slate-400" />
                           {{ tag }}
                         </div>
                       </div>
                       <div class="font-mono text-[10px] text-slate-400">{{ img.shortId }}</div>
                   </div>

                   <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-xs">
                      <div>
                         <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-0.5">Platform</div>
                         <div class="font-mono text-slate-900 dark:text-slate-300">{{ img.architecture }} / {{ img.os }}</div>
                      </div>
                      <div>
                         <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-0.5">Size</div>
                         <div class="font-mono text-slate-900 dark:text-slate-300">{{ img.size }} MB</div>
                      </div>
                      <div>
                         <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-0.5">Created</div>
                         <div class="font-mono text-slate-900 dark:text-slate-300 truncate" :title="img.createdDate">{{ img.relativeTime }}</div>
                      </div>
                      <div>
                          <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-0.5">Digest</div>
                          <div class="font-mono text-slate-900 dark:text-slate-300 truncate opacity-60" :title="img.digest">{{ img.digest.substring(7, 19) }}...</div>
                      </div>
                   </div>
                </div>
             </div>
          </div>
          
        </div>

        <!-- Right Column: Deployment Configuration -->
        <div class="lg:col-span-4">
          <div class="space-y-6 sticky top-6">
            <!-- Dependencies -->
            <div v-if="dependencies.length > 0" class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 p-6 shadow-sm">
              <h2 class="text-sm font-bold text-slate-900 dark:text-white uppercase tracking-wider flex items-center gap-2 border-b border-slate-100 dark:border-slate-800 pb-4">
                <Package :size="16" class="text-amber-500" />
                Dependencies
              </h2>

              <div class="mt-4 flex flex-wrap gap-2">
                <button
                  v-for="dep in dependencies"
                  :key="dep"
                  @click="router.push(`/apps/${dep}`)"
                  class="inline-flex items-center gap-1.5 px-2 py-1 text-xs font-mono uppercase tracking-wider rounded border transition-all hover:scale-105 active:scale-95"
                  :class="missingDependencies.includes(dep)
                    ? 'border-amber-300 text-amber-700 bg-amber-50 dark:border-amber-700 dark:text-amber-300 dark:bg-amber-950/30 hover:bg-amber-100 dark:hover:bg-amber-900/40 cursor-pointer'
                    : 'border-emerald-200 text-emerald-700 bg-emerald-50 dark:border-emerald-800/50 dark:text-emerald-300 dark:bg-emerald-950/20 hover:bg-emerald-100 dark:hover:bg-emerald-900/30 cursor-pointer'"
                  :title="`Click to view ${dep}`"
                >
                  {{ dep }}
                  <ExternalLink :size="10" class="opacity-50" />
                </button>
              </div>

              <div v-if="missingDependencies.length > 0" class="mt-4 flex items-start gap-2 text-[11px] text-amber-600 dark:text-amber-400">
                <AlertTriangle :size="14" />
                <div class="flex-1">
                  <span class="block mb-1">
                    Start required apps first:
                  </span>
                  <div class="flex flex-wrap gap-1">
                    <button
                      v-for="dep in missingDependencies"
                      :key="dep"
                      @click="router.push(`/apps/${dep}`)"
                      class="inline-flex items-center gap-1 px-1.5 py-0.5 bg-amber-100 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-700 rounded text-amber-700 dark:text-amber-300 hover:bg-amber-200 dark:hover:bg-amber-800/40 transition-colors font-mono text-[10px] font-medium"
                    >
                      {{ dep }}
                      <ExternalLink :size="8" />
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 p-6 shadow-sm">
              <h2 class="text-sm font-bold text-slate-900 dark:text-white mb-6 uppercase tracking-wider flex items-center gap-2 border-b border-slate-100 dark:border-slate-800 pb-4">
                <span class="text-blue-500">⚙️</span>
                Configuration
              </h2>

              <div class="space-y-6">
              <!-- Environment Vars -->
              <div v-if="app.environment?.length > 0" class="space-y-4">
                <div v-for="env in app.environment" :key="env.envVar" class="space-y-1.5">
                  <label class="w-full text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wide flex items-center justify-between">
                    {{ env.name }}
                    <span v-if="env.default" class="text-[10px] text-slate-400 dark:text-slate-500 bg-slate-100 dark:bg-slate-900 px-1.5 rounded">{{ env.default }}</span>
                  </label>
                  <input
                    v-model="envValues[env.envVar]"
                    type="text"
                    :placeholder="env.default || 'Value'"
                    class="w-full bg-slate-50 dark:bg-[#1a1a1c] border border-slate-200 dark:border-slate-800 rounded px-3 py-2 text-sm font-mono text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all"
                  />
                  <p v-if="env.description" class="text-[10px] text-slate-500 leading-tight">{{ env.description }}</p>
                </div>
              </div>

              <!-- Options Toggles -->
              <div class="space-y-4 pt-4 border-t border-slate-100 dark:border-slate-800">
                
                <!-- Temporary Install -->
                <div :class="['rounded-lg border p-3 transition-colors', temporaryInstall ? 'bg-blue-50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-800' : 'bg-slate-50 dark:bg-slate-900/50 border-slate-200 dark:border-slate-800']">
                    <div class="flex items-start gap-3">
                        <input type="checkbox" id="temp-install" v-model="temporaryInstall" class="mt-1 w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-offset-0 focus:ring-0 cursor-pointer" />
                        <div class="flex-1">
                            <label for="temp-install" class="block text-xs font-bold text-slate-800 dark:text-slate-200 cursor-pointer uppercase">Temporary Install</label>
                            <p class="text-[10px] text-slate-500 mt-0.5">Expires & auto-deletes.</p>
                        </div>
                    </div>
                    
                    <div v-if="temporaryInstall" class="mt-3 pl-7 animate-in fade-in slide-in-from-top-1">
                        <select v-model.number="expirationHours" class="w-full bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-700 rounded p-1.5 text-xs font-mono focus:border-blue-500 focus:outline-none">
                            <option :value="1">1 Hour</option>
                            <option :value="6">6 Hours</option>
                            <option :value="12">12 Hours</option>
                            <option :value="24">1 Day</option>
                            <option :value="72">3 Days</option>
                        </select>
                    </div>
                </div>

                <!-- Custom Ports -->
                <div v-if="allPorts.length > 0" :class="['rounded-lg border p-3 transition-colors', customizePorts ? 'bg-indigo-50 dark:bg-indigo-900/10 border-indigo-200 dark:border-indigo-800' : 'bg-slate-50 dark:bg-slate-900/50 border-slate-200 dark:border-slate-800']">
                   <div class="flex items-start gap-3">
                        <input type="checkbox" id="custom-ports" v-model="customizePorts" class="mt-1 w-4 h-4 rounded border-slate-300 text-indigo-600 focus:ring-offset-0 focus:ring-0 cursor-pointer" />
                        <div class="flex-1">
                            <label for="custom-ports" class="block text-xs font-bold text-slate-800 dark:text-slate-200 cursor-pointer uppercase">Port Mapping</label>
                            <p class="text-[10px] text-slate-500 mt-0.5">Advanced Configuration</p>
                        </div>
                   </div>

                    <div v-if="customizePorts" class="mt-3 pl-1 space-y-3 animate-in fade-in slide-in-from-top-1">
                        <div v-for="port in allPorts" :key="port.hostPort + '/' + port.protocol" class="space-y-1">
                            <div class="flex items-center justify-between text-[10px] font-mono font-medium text-slate-500 uppercase tracking-wider">
                                <span>INTERNAL: {{ port.containerPort }} ({{ port.protocol }})</span>
                            </div>
                            <div class="flex items-center gap-2">
                                <span class="text-slate-400 text-sm">→</span>
                                <input
                                v-model="customPortMappings[port.hostPort + '/' + port.protocol]"
                                type="number"
                                :placeholder="port.hostPort"
                                class="flex-1 bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-700 rounded px-2 py-1.5 text-xs font-mono focus:border-indigo-500 focus:outline-none"
                                />
                            </div>
                            <!-- Port Status -->
                            <div v-if="customPortMappings[port.hostPort + '/' + port.protocol]" class="flex items-center justify-end">
                                <div class="flex items-center gap-1.5 text-[9px] uppercase font-bold tracking-wider">
                                    <span :class="{
                                        'text-red-500': getPortStatus(port).status === 'conflict',
                                        'text-yellow-500': getPortStatus(port).status === 'warning',
                                        'text-emerald-500': getPortStatus(port).status === 'available'
                                    }">
                                      <span v-if="getPortStatus(port).status === 'conflict'" class="flex items-center gap-1"><AlertTriangle :size="10" /> {{ getPortStatus(port).message }}</span>
                                      <span v-else-if="getPortStatus(port).status === 'available'" class="flex items-center gap-1"><Check :size="10" /> {{ getPortStatus(port).message }}</span>
                                      <span v-else>{{ getPortStatus(port).message }}</span>
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
              </div>

              <!-- Deploy Button -->
              <div class="pt-2">
                 <button
                   @click="deployApp"
                   :disabled="!canDeploy"
                   :title="missingDependencies.length > 0 ? `Missing dependencies: ${missingDependencies.join(', ')} (deploy anyway)` : ''"
                   class="w-full relative group bg-indigo-600 hover:bg-indigo-700 text-white rounded-md p-3.5 font-bold uppercase tracking-wider text-xs transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed shadow-sm hover:shadow-indigo-500/20"
                 >
                    <span v-if="deploying" class="flex items-center justify-center gap-3">
                       <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                       Initializing...
                    </span>
                    <span v-else class="flex items-center justify-center gap-2">
                       <Play :size="16" fill="currentColor" />
                       {{ instanceCount > 0 ? 'Deploy Another Instance' : 'Install Application' }}
                    </span>
                 </button>
                 <div v-if="instanceCount > 0" class="text-center mt-3 text-[10px] font-mono text-slate-400 dark:text-slate-500">
                    {{ instanceCount }} Active Instance{{ instanceCount !== 1 ? 's' : '' }} running
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
