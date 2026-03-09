<script setup>
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useNotification } from '../composables/useNotification';
import { useApiUrl } from "../composables/useApiUrl";
import { usePortConflict } from "../composables/usePortConflict";
import { useI18n } from "vue-i18n";
import { Globe, FileCode, ArrowLeft, Package, Clock, Tag, ExternalLink, Activity, Info, AlertTriangle, Check, Terminal, Play, CreditCard, RotateCcw, Download, Plus, X } from "lucide-vue-next";
import { buildChatGptExplainUrl } from "../utils/chatgpt";

const route = useRoute();
const router = useRouter();
const toast = useNotification();
const { apiUrl } = useApiUrl();
const { t } = useI18n();

// State
const app = ref(null);
const containers = ref([]);
const loading = ref(true);
const deploying = ref(false);
const envValues = ref({});
const dependencyEnvSuggestions = ref({});
const loadingDependencyEnv = ref(false);
const temporaryInstall = ref(false);
const expirationHours = ref(24);
const customizePorts = ref(false);
const customPortMappings = ref({});
const extraEnvRows = ref([]);

function addExtraEnvRow() {
  extraEnvRows.value.push({ key: '', value: '' });
}

function removeExtraEnvRow(index) {
  extraEnvRows.value.splice(index, 1);
}
const imageDetails = ref(null);
const loadingImages = ref(false);

// Port conflict detection
const { checkPortConflict, getPortStatus: getPortStatusFn } = usePortConflict(containers);

function getPortStatus(port) {
  return getPortStatusFn(port, customPortMappings.value);
}

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

// Ports from info.json — used in the Network Requirements info table
const infoPorts = computed(() => {
  return Array.isArray(app.value?.ports) ? app.value.ports : [];
});

// Ports from compose.yml — used in the deploy form port customization
const allPorts = computed(() => {
  return Array.isArray(app.value?.composePorts) ? app.value.composePorts : [];
});

const appTags = computed(() => {
  return Array.isArray(app.value?.tags) ? app.value.tags : [];
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

// Get suggested value from dependency containers with smart matching
function getSuggestedValue(envVar) {
  // First try direct match
  for (const [depId, depEnv] of Object.entries(dependencyEnvSuggestions.value)) {
    if (depEnv[envVar]) {
      return depEnv[envVar];
    }
  }

  // Try smart matching for common patterns
  const matchPatterns = {
    'BTCEXP_BITCOIND_USER': ['BITCOIN_RPC_USER', 'RPC_USER', 'RPCUSER'],
    'BTCEXP_BITCOIND_PASS': ['BITCOIN_RPC_PASSWORD', 'RPC_PASSWORD', 'RPCPASSWORD'],
    'BTCEXP_BITCOIND_PASSWORD': ['BITCOIN_RPC_PASSWORD', 'RPC_PASSWORD', 'RPCPASSWORD'],
  };

  // Check if we have a pattern for this env var
  if (matchPatterns[envVar]) {
    for (const [depId, depEnv] of Object.entries(dependencyEnvSuggestions.value)) {
      for (const pattern of matchPatterns[envVar]) {
        if (depEnv[pattern]) {
          return depEnv[pattern];
        }
      }
    }
  }

  // Generic smart matching: try to find similar variable names
  const cleanEnvVar = envVar.toLowerCase().replace(/^[a-z]+_/, ''); // Remove prefix like BTCEXP_
  for (const [depId, depEnv] of Object.entries(dependencyEnvSuggestions.value)) {
    for (const [key, value] of Object.entries(depEnv)) {
      const cleanKey = key.toLowerCase().replace(/^[a-z]+_/, '');
      if (cleanKey === cleanEnvVar || cleanKey.includes(cleanEnvVar) || cleanEnvVar.includes(cleanKey)) {
        return value;
      }
    }
  }

  return null;
}

// Functions
async function fetchApp() {
  try {
    const response = await fetch(`${apiUrl.value}/api/apps`);
    const data = await response.json();

    if (data.success && data.apps) {
      app.value = data.apps.find((a) => a.id === route.params.appname);

      if (!app.value) {
        toast.error(t('appDetail.appNotFound'));
        router.push("/apps");
      }
    } else {
      throw new Error(t('appDetail.failedToLoadApps'));
    }
  } catch (error) {
    console.error("Error fetching app:", error);
    toast.error(t('appDetail.failedToLoadAppDetails'));
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

function parseContainerEnv(envList) {
  const envMap = {};
  if (!Array.isArray(envList)) return envMap;

  envList.forEach((entry) => {
    const idx = entry.indexOf("=");
    if (idx <= 0) return;
    const key = entry.slice(0, idx).trim();
    if (!key) return;
    envMap[key] = entry.slice(idx + 1);
  });

  return envMap;
}

function extractEnvVarTokens(value) {
  if (!value || typeof value !== "string") return [];
  const matches = [...value.matchAll(/\$\{([A-Z0-9_]+)(?::?-?[^}]*)?\}/g)];
  return matches.map((match) => match[1]);
}

function buildDependencyEnvIndex() {
  const envIndex = {};
  const sourceIndex = {};

  dependencies.value.forEach((dep) => {
    const runningContainer = containers.value.find(
      (container) => container.app?.id === dep && container.state === "running"
    );
    const fallbackContainer = containers.value.find(
      (container) => container.app?.id === dep
    );
    const container = runningContainer || fallbackContainer;
    if (!container) return;

    const envMap = parseContainerEnv(container.env);
    Object.entries(envMap).forEach(([key, value]) => {
      if (envIndex[key] === undefined) {
        envIndex[key] = value;
        sourceIndex[key] = dep;
      }
    });
  });

  return { envIndex, sourceIndex };
}

function autoFillEnvFromDependencies() {
  if (!app.value?.environment?.length || !dependencies.value.length) return;

  const { envIndex, sourceIndex } = buildDependencyEnvIndex();
  const nextSources = {};
  const nextValues = { ...envValues.value };

  app.value.environment.forEach((env) => {
    if (nextValues[env.envVar]) return;

    if (envIndex[env.envVar] !== undefined) {
      nextValues[env.envVar] = envIndex[env.envVar];
      nextSources[env.envVar] = sourceIndex[env.envVar];
      return;
    }

    const tokens = extractEnvVarTokens(env.default);
    for (const token of tokens) {
      if (envIndex[token] !== undefined) {
        nextValues[env.envVar] = envIndex[token];
        nextSources[env.envVar] = sourceIndex[token];
        break;
      }
    }
  });

  envValues.value = nextValues;
}

async function fillFromDependencies() {
  if (!app.value || !dependencies.value.length) {
    toast.info(t('appDetail.noDependenciesToFill'));
    return;
  }

  loadingDependencyEnv.value = true;

  try {
    const response = await fetch(`${apiUrl.value}/api/apps/${app.value.id}/dependency-env`);
    const data = await response.json();

    if (data.success && data.environmentVariables) {
      let filledCount = 0;

      // Fill environment variables using smart matching
      if (app.value.environment) {
        app.value.environment.forEach(env => {
          const envVar = env.envVar;

          // Skip if already filled
          if (envValues.value[envVar]) return;

          // Try direct match first
          for (const [depId, depEnv] of Object.entries(data.environmentVariables)) {
            if (depEnv[envVar]) {
              envValues.value[envVar] = depEnv[envVar];
              filledCount++;
              return;
            }
          }

          // Try smart matching patterns
          const matchPatterns = {
            'BTCEXP_BITCOIND_USER': ['BITCOIN_RPC_USER', 'RPC_USER', 'RPCUSER'],
            'BTCEXP_BITCOIND_PASS': ['BITCOIN_RPC_PASSWORD', 'RPC_PASSWORD', 'RPCPASSWORD'],
          };

          if (matchPatterns[envVar]) {
            for (const [depId, depEnv] of Object.entries(data.environmentVariables)) {
              for (const pattern of matchPatterns[envVar]) {
                if (depEnv[pattern]) {
                  envValues.value[envVar] = depEnv[pattern];
                  filledCount++;
                  return;
                }
              }
            }
          }

          // Fall back: if the var has a literal default (not a ${...} reference) and at least
          // one dependency is present in the response, the default is the correct value to use.
          const literalDefault = env.default && !env.default.includes('${') ? env.default.trim() : null;
          if (literalDefault && Object.keys(data.environmentVariables).length > 0) {
            envValues.value[envVar] = literalDefault;
            filledCount++;
          }
        });
      }

      if (filledCount > 0) {
        const plural = filledCount > 1 ? 's' : '';
        toast.success(t('appDetail.filledVariables', { count: filledCount, plural }));
      } else {
        toast.info(t('appDetail.noMatchingVariables'));
      }
    }
  } catch (error) {
    console.error("Error fetching dependency environment variables:", error);
    toast.error(t('appDetail.failedToFetchDependencyVariables'));
  } finally {
    loadingDependencyEnv.value = false;
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
      t('appDetail.missingDependenciesConfirm', { deps: depApps })
    );

    if (!proceed) {
      toast.info(t('appDetail.deploymentCancelled'), {
        timeout: 4000
      });
      return;
    }

    allowMissingDependencies = true;
    toast.warning(t('appDetail.deployingWithoutDependencies', { deps: depApps }), {
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
      toast.error(t('appDetail.portConflictsDetected', { conflicts: conflicts.join("\n") }));
      return;
    }
  }

  deploying.value = true;
  const instanceNum = nextInstanceNumber.value;
  const instanceSuffix = instanceNum > 1 ? ` #${instanceNum}` : "";
  toast.info(t('appDetail.deployingApp', { name: app.value.name, suffix: instanceSuffix }));

  try {
    // Build extra env object from user-added custom rows
    const extraEnv = {};
    for (const row of extraEnvRows.value) {
      const k = row.key.trim();
      if (k) extraEnv[k] = row.value;
    }

    const requestBody = {
      appId: app.value.id,
      environment: envValues.value,
      extraEnv,
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
        toast.success(t('appDetail.deployedAsTemporary', { name: app.value.name, hours: expirationHours.value }));
      } else {
        toast.success(t('appDetail.installedSuccessfully', { name: app.value.name }));
      }

      // Wait a moment then redirect to containers
      setTimeout(() => {
        router.push("/");
      }, 1500);
    } else {
      // Check if it's a dependency error
      if (result.missingDependencies && result.missingDependencies.length > 0) {
        const deps = result.missingDependencies.join(", ");
        toast.error(t('appDetail.missingDependenciesDeployFirst', { deps }), {
          timeout: 5000
        });
      } else {
        throw new Error(result.message || result.error || t('appDetail.deploymentFailed'));
      }
    }
  } catch (error) {
    console.error("Deployment error:", error);
    if (error.message.includes("timeout")) {
      toast.error(t('appDetail.deploymentTimeout', { name: app.value.name }));
    } else {
      toast.error(t('appDetail.deploymentFailedMessage', { message: error.message }));
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
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-zinc-100 font-sans selection:bg-blue-500/30">
    
    <!-- Header -->
    <header class="bg-white/80 dark:bg-[#0A0A0A]/80 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800 sticky top-0 z-30">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 h-14 sm:h-16 flex items-center justify-between">
        <div class="flex items-center gap-2 sm:gap-4">
          <router-link to="/apps" class="inline-flex items-center justify-center w-10 h-10 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-900 transition-all text-gray-500 dark:text-zinc-400 group">
            <ArrowLeft :size="16" class="group-hover:-translate-x-0.5 transition-transform" />
          </router-link>

          <div class="h-4 w-px bg-gray-300 dark:bg-zinc-800"></div>

          <div class="flex items-center gap-2.5 text-sm min-w-0">
            <span class="hidden sm:inline text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ t('appDetail.catalog') }}</span>
            <span class="hidden sm:inline text-gray-300 dark:text-zinc-700">/</span>
            <span class="font-semibold tracking-tight text-gray-900 dark:text-white truncate" v-if="app">{{ app.name }}</span>
            <span v-else class="w-24 h-4 bg-gray-200 dark:bg-zinc-800 animate-pulse rounded"></span>
          </div>
        </div>

        <div v-if="isInstalled" class="flex items-center gap-2 px-2.5 py-1 rounded-md border border-green-200 dark:border-green-500/20 bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 text-[10px] font-bold uppercase tracking-wider">
          <div class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></div>
          <span>{{ t('appDetail.installed') }}</span>
        </div>
      </div>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="flex flex-col items-center justify-center min-h-[60vh]">
      <div class="w-8 h-8 border-[3px] border-gray-200 dark:border-zinc-800 border-t-blue-500 dark:border-t-blue-500 rounded-full animate-spin mb-6"></div>
      <div class="font-bold text-[10px] tracking-widest text-gray-400 dark:text-zinc-500 uppercase">{{ t('appDetail.retrievingManifest') }}</div>
    </div>

    <div v-else-if="app" class="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8 grid grid-cols-1 lg:grid-cols-12 gap-6">

        <!-- Left Column: Information & Specs -->
        <div class="lg:col-span-8 space-y-6">

          <!-- Identity Card -->
          <div class="group relative bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6 flex flex-col sm:flex-row gap-6 transition-all hover:border-gray-300 dark:hover:border-zinc-700">
            <!-- Glow Accent -->
            <div class="absolute top-0 left-0 w-full h-0.5 bg-linear-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

            <div class="w-20 h-20 bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 rounded-xl flex items-center justify-center p-4 shrink-0 shadow-sm transition-transform group-hover:scale-105 duration-500">
              <img :src="app.logo" :alt="app.name" loading="lazy" class="w-full h-full object-contain filter dark:brightness-90 group-hover:brightness-100 transition-all" />
            </div>

            <div class="flex-1 flex flex-col">
              <div class="flex flex-col mb-3">
                <div class="flex items-center gap-3 mb-2">
                  <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{{ app.name }}</h1>
                  <span
                    v-if="app.customapp"
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-purple-50 dark:bg-purple-500/10 border border-purple-200 dark:border-purple-500/20 text-[10px] font-bold uppercase tracking-widest text-purple-600 dark:text-purple-400 shrink-0"
                  >
                    Built by Yantr
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

        <!-- Right Column: Deployment Configuration -->
        <div class="lg:col-span-4">
          <div class="space-y-6 sticky top-24">
            
            <!-- Dependencies -->
            <div v-if="dependencies.length > 0" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-5">
              <div class="flex items-center justify-between mb-5">
                <div class="flex items-center gap-2">
                  <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('appDetail.dependencies') }}</div>
                </div>

                <div class="text-[9px] font-bold uppercase tracking-wider">
                  <span v-if="missingDependencies.length === 0" class="text-green-600 dark:text-green-500">{{ t('appDetail.allRunning') }}</span>
                  <span v-else class="text-amber-600 dark:text-amber-500">{{ missingDependencies.length }} {{ t('appDetail.missing') }}</span>
                </div>
              </div>

              <div class="space-y-2">
                <button
                  v-for="dep in dependencies"
                  :key="dep"
                  @click="router.push(`/apps/${dep}`)"
                  class="group w-full flex items-center justify-between gap-3 rounded-lg border px-3 py-2.5 transition-all bg-gray-50 dark:bg-zinc-900/30"
                  :class="missingDependencies.includes(dep)
                    ? 'border-amber-200 dark:border-amber-900/50 hover:border-amber-300 dark:hover:border-amber-700/50'
                    : 'border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-600'"
                >
                  <div class="flex items-center gap-2">
                    <span class="h-2 w-2 rounded-full"
                      :class="missingDependencies.includes(dep)
                        ? 'bg-amber-500 animate-pulse'
                        : 'bg-green-500'"
                    ></span>
                    <span class="text-xs font-mono uppercase tracking-wider text-gray-900 dark:text-white">{{ dep }}</span>
                  </div>
                  <ExternalLink :size="12" class="text-gray-400 group-hover:text-gray-700 dark:group-hover:text-white transition-colors" />
                </button>
              </div>

              <div v-if="missingDependencies.length > 0" class="mt-4 rounded-lg border border-amber-200/50 dark:border-amber-900/30 bg-amber-50/50 dark:bg-amber-900/10 px-3 py-2 text-[10px] text-amber-700 dark:text-amber-400 flex items-start gap-2">
                <AlertTriangle :size="12" class="mt-0.5 shrink-0" />
                <span>{{ t('appDetail.missingDependenciesWarning') }}</span>
              </div>
            </div>

            <!-- Custom App Notice -->
            <div v-if="app.customapp" class="flex items-start gap-3 rounded-xl border border-purple-200 dark:border-purple-500/20 bg-purple-50 dark:bg-purple-500/10 px-4 py-3">
              <div class="w-2 h-2 rounded-full bg-purple-500 shrink-0 mt-1"></div>
              <div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-purple-600 dark:text-purple-400 mb-0.5">Built by Yantr</div>
                <p class="text-[11px] text-purple-700 dark:text-purple-300 leading-relaxed">This app is custom-built and maintained by the Yantr team. It uses a locally-built image and does not support automatic updates.</p>
              </div>
            </div>

            <!-- Configuration -->
            <div class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-5">
              <div class="flex items-center justify-between mb-5">
                <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">
                  {{ t('appDetail.configuration') }}
                </h2>
                <button
                  v-if="dependencies.length > 0 && app.environment?.length > 0"
                  @click="fillFromDependencies"
                  :disabled="loadingDependencyEnv || missingDependencies.length > 0"
                  class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <svg v-if="loadingDependencyEnv" class="animate-spin h-3 w-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <Download v-else :size="12" />
                  <span>{{ t('appDetail.autofillEnv') }}</span>
                </button>
              </div>

              <div class="space-y-6">
              <!-- Environment Vars -->
              <div v-if="app.environment?.length > 0" class="space-y-4">
                <div v-for="env in app.environment" :key="env.envVar" class="space-y-1.5">
                  <label class="w-full text-[10px] font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-widest flex items-center justify-between">
                    {{ env.name }}
                    <span v-if="env.default" class="text-[9px] font-mono text-gray-400 dark:text-zinc-500 border border-gray-200 dark:border-zinc-800 px-1.5 py-0.5 rounded">{{ env.default }}</span>
                  </label>
                  <input
                    v-model="envValues[env.envVar]"
                    type="text"
                    :placeholder="env.default || t('appDetail.value')"
                    class="w-full bg-transparent border border-gray-200 dark:border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all"
                  />
                  <p v-if="env.description" class="text-[10px] text-gray-500 dark:text-zinc-500 leading-tight">{{ env.description }}</p>
                </div>
              </div>

              <!-- Custom / Extra Environment Variables -->
              <div class="space-y-3 pt-4 border-t border-gray-100 dark:border-zinc-800">
                <div class="flex items-center justify-between">
                  <span class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('appDetail.customVars') }}</span>
                  <button
                    @click="addExtraEnvRow"
                    class="flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
                  >
                    <Plus :size="11" />
                    {{ t('appDetail.addCustomVar') }}
                  </button>
                </div>
                <div v-if="extraEnvRows.length > 0" class="space-y-2">
                  <div v-for="(row, i) in extraEnvRows" :key="i" class="flex items-center gap-2">
                    <input
                      v-model="row.key"
                      type="text"
                      :placeholder="t('appDetail.varName')"
                      class="w-2/5 bg-transparent border border-gray-200 dark:border-zinc-800 rounded-lg px-2.5 py-1.5 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all uppercase"
                    />
                    <span class="text-gray-400 dark:text-zinc-600 text-xs shrink-0">=</span>
                    <input
                      v-model="row.value"
                      type="text"
                      :placeholder="t('appDetail.varValue')"
                      class="flex-1 bg-transparent border border-gray-200 dark:border-zinc-800 rounded-lg px-2.5 py-1.5 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all"
                    />
                    <button
                      @click="removeExtraEnvRow(i)"
                      class="shrink-0 p-1.5 rounded-md text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 transition-all"
                    >
                      <X :size="13" />
                    </button>
                  </div>
                </div>
                <p v-else class="text-[10px] text-gray-400 dark:text-zinc-600">{{ t('appDetail.customVarsHint') }}</p>
              </div>

              <!-- Options Toggles -->
              <div class="space-y-3 pt-4 border-t border-gray-100 dark:border-zinc-800">

                <!-- Temporary Install -->
                <div class="rounded-lg border border-gray-200 dark:border-zinc-800 p-3 transition-colors bg-gray-50 dark:bg-zinc-900/30">
                    <div class="flex items-start gap-3">
                        <input type="checkbox" id="temp-install" v-model="temporaryInstall" class="mt-0.5 w-4 h-4 shrink-0 rounded border-gray-300 dark:border-zinc-700 text-black dark:text-white focus:ring-black dark:focus:ring-white focus:ring-offset-0 cursor-pointer bg-transparent" />
                        <div class="flex-1">
                            <label for="temp-install" class="block text-[11px] font-bold text-gray-900 dark:text-zinc-100 cursor-pointer uppercase tracking-wider">{{ t('appDetail.temporaryInstall') }}</label>
                            <p class="text-[10px] text-gray-500 dark:text-zinc-500 mt-0.5">{{ t('appDetail.temporaryInstallDesc') }}</p>
                        </div>
                    </div>
                    
                    <div v-if="temporaryInstall" class="mt-3 pl-6">
                        <select v-model.number="expirationHours" class="w-full bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-lg p-2 text-[11px] font-bold uppercase tracking-wider text-gray-900 dark:text-white focus:border-blue-500 focus:outline-none transition-colors cursor-pointer">
                            <option :value="1">{{ t('appDetail.1hour') }}</option>
                            <option :value="6">{{ t('appDetail.6hours') }}</option>
                            <option :value="12">{{ t('appDetail.12hours') }}</option>
                            <option :value="24">{{ t('appDetail.1day') }}</option>
                            <option :value="72">{{ t('appDetail.3days') }}</option>
                            <option :value="168">{{ t('appDetail.1week') }}</option>
                            <option :value="336">{{ t('appDetail.2weeks') }}</option>
                            <option :value="720">{{ t('appDetail.1month') }}</option>
                        </select>
                    </div>
                </div>

                <!-- Custom Ports -->
                <div v-if="allPorts.length > 0" class="rounded-lg border border-gray-200 dark:border-zinc-800 p-3 transition-colors bg-gray-50 dark:bg-zinc-900/30">
                   <div class="flex items-start gap-3">
                        <input type="checkbox" id="custom-ports" v-model="customizePorts" class="mt-0.5 w-4 h-4 shrink-0 rounded border-gray-300 dark:border-zinc-700 text-black dark:text-white focus:ring-black dark:focus:ring-white focus:ring-offset-0 cursor-pointer bg-transparent" />
                        <div class="flex-1">
                            <label for="custom-ports" class="block text-[11px] font-bold text-gray-900 dark:text-zinc-100 cursor-pointer uppercase tracking-wider">{{ t('appDetail.portMapping') }}</label>
                            <p class="text-[10px] text-gray-500 dark:text-zinc-500 mt-0.5">{{ t('appDetail.advancedConfig') }}</p>
                        </div>
                   </div>

                    <div v-if="customizePorts" class="mt-4 pl-1 space-y-4">
                        <div v-for="port in allPorts" :key="port.hostPort + '/' + port.protocol" class="space-y-1.5">
                            <div class="flex items-center justify-between text-[10px] font-mono font-medium text-gray-500 uppercase tracking-wider">
                                <span>{{ t('appDetail.internal') }}: {{ port.containerPort }} ({{ port.protocol }})</span>
                            </div>
                            <div class="flex items-center gap-2">
                                <span class="text-gray-400 text-sm">→</span>
                                <input
                                v-model="customPortMappings[port.hostPort + '/' + port.protocol]"
                                type="number"
                                :placeholder="port.hostPort"
                                class="flex-1 bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-lg px-2.5 py-1.5 text-xs font-mono focus:border-blue-500 focus:outline-none transition-colors"
                                />
                            </div>
                            <!-- Port Status -->
                            <div v-if="customPortMappings[port.hostPort + '/' + port.protocol]" class="flex items-center justify-end">
                                <div class="flex items-center gap-1.5 text-[9px] uppercase font-bold tracking-wider">
                                    <span :class="{
                                        'text-red-500': getPortStatus(port).status === 'conflict',
                                        'text-yellow-500': getPortStatus(port).status === 'warning',
                                        'text-green-500': getPortStatus(port).status === 'available'
                                    }">
                                      <span v-if="getPortStatus(port).status === 'conflict'" class="flex items-center gap-1"><AlertTriangle :size="10" /> {{ t('appDetail.portConflict') }}</span>
                                      <span v-else-if="getPortStatus(port).status === 'available'" class="flex items-center gap-1"><Check :size="10" /> {{ t('appDetail.portAvailable') }}</span>
                                      <span v-else>{{ getPortStatus(port).message }}</span>
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
              </div>

              <!-- Deploy Button -->
              <div class="pt-4">
                 <button
                   @click="deployApp"
                   :disabled="!canDeploy"
                   :title="missingDependencies.length > 0 ? `Missing dependencies: ${missingDependencies.join(', ')} (deploy anyway)` : ''"
                   class="w-full flex items-center justify-center gap-2 px-4 py-3 bg-black dark:bg-white text-white dark:text-black hover:bg-gray-800 dark:hover:bg-gray-200 rounded-xl text-xs font-bold uppercase tracking-wider transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                 >
                    <span v-if="deploying" class="flex items-center justify-center gap-2">
                       <span class="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin"></span>
                       {{ t('appDetail.initializing') }}
                    </span>
                    <span v-else class="flex items-center justify-center gap-2">
                       <Play :size="14" fill="currentColor" />
                       {{ instanceCount > 0 ? t('appDetail.deployAnother') : t('appDetail.installApp') }}
                    </span>
                 </button>
                 <div v-if="instanceCount > 0" class="text-center mt-3 text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">
                    {{ instanceCount }} {{ instanceCount !== 1 ? t('appDetail.activeInstancesPlural') : t('appDetail.activeInstances') }} {{ t('appDetail.running') }}
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
