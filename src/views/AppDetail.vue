<script setup>
import { ref, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useNotification } from '../composables/useNotification';
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess, getApiErrorMessage, readJsonResponse } from "../composables/useApiResponse";
import { usePortConflict } from "../composables/usePortConflict";
import { useI18n } from "vue-i18n";
import { Globe, FileCode, Package, Clock, Tag, ExternalLink, Activity, Info, AlertTriangle, Check, Terminal, Play, CreditCard, Plus, X } from "lucide-vue-next";
import AppLogo from "../components/AppLogo.vue";
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
const temporaryInstall = ref(false);
const expirationHours = ref(24);
const customizePorts = ref(false);
const customPortMappings = ref({});
const extraEnvRows = ref([]);

// Auth state
const enableAuth = ref(false);
const authPort = ref(3002);
const authUsername = ref("admin");
const authPassword = ref("");

function generateAuthPassword() {
  authPassword.value = randomStringFromCharacters("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+[]{}:,.?", 16);
  toast.success(t('appDetail.generatedValue', { envVar: 'Password' }));
}

function addExtraEnvRow() {
  extraEnvRows.value.push({ key: '', value: '' });
}

function removeExtraEnvRow(index) {
  extraEnvRows.value.splice(index, 1);
}
const imageDetails = ref(null);
const loadingImages = ref(false);
const loadErrorState = {
  app: false,
  containers: false,
  images: false,
};

function notifyLoadErrorOnce(key, message) {
  if (loadErrorState[key]) return;
  loadErrorState[key] = true;
  toast.error(message);
}

function clearLoadError(key) {
  loadErrorState[key] = false;
}

// Port conflict detection
const { getPortStatus: getPortStatusFn } = usePortConflict(containers);

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

const canDeploy = computed(() => {
  return !deploying.value;
});

const chatGptUrl = computed(() => {
  if (!app.value) return "";

  return buildChatGptExplainUrl(app.value.id);
});

function getGeneratorRule(envVar) {
  const rules = app.value?.envGenerators;
  if (!rules || typeof rules !== "object") return null;
  const rule = rules[envVar];
  if (!rule || typeof rule !== "object" || Array.isArray(rule)) return null;
  return rule;
}

function randomInt(max) {
  if (!Number.isInteger(max) || max <= 0) return 0;
  if (window.crypto?.getRandomValues) {
    const arr = new Uint32Array(1);
    window.crypto.getRandomValues(arr);
    return arr[0] % max;
  }
  return Math.floor(Math.random() * max);
}

function randomStringFromCharacters(characters, length) {
  if (!characters || length <= 0) return "";
  let result = "";
  for (let i = 0; i < length; i += 1) {
    result += characters[randomInt(characters.length)];
  }
  return result;
}

function resolveGeneratorCharacters(rule) {
  if (typeof rule.characters === "string" && rule.characters.length > 0) {
    return rule.characters;
  }

  const charsets = {
    alnum: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
    hex: "0123456789abcdef",
    numeric: "0123456789",
    alpha: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
    base64url: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_",
    alnum_symbols: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+[]{}:,.?",
  };

  const selected = typeof rule.charset === "string" ? rule.charset : "alnum";
  return charsets[selected] || charsets.alnum;
}

function resolveGeneratorLength(rule) {
  const parsed = Number.parseInt(rule.length, 10);
  if (!Number.isInteger(parsed)) return 32;
  return Math.max(8, Math.min(parsed, 256));
}

function buildGeneratorRegex(rule) {
  const pattern = typeof rule.regex === "string"
    ? rule.regex
    : typeof rule.pattern === "string"
      ? rule.pattern
      : null;

  if (!pattern) return null;

  try {
    return new RegExp(pattern);
  } catch {
    return false;
  }
}

function generateValueForRule(rule) {
  const length = resolveGeneratorLength(rule);
  const characters = resolveGeneratorCharacters(rule);
  const regex = buildGeneratorRegex(rule);

  if (regex === false) {
    throw new Error(t('appDetail.invalidGeneratorRule'));
  }

  const maxAttempts = 32;
  for (let i = 0; i < maxAttempts; i += 1) {
    const candidate = randomStringFromCharacters(characters, length);
    if (!regex || regex.test(candidate)) {
      return candidate;
    }
  }

  throw new Error(t('appDetail.generationFailed'));
}

function generateEnvValue(envVar) {
  const rule = getGeneratorRule(envVar);
  if (!rule) {
    toast.info(t('appDetail.noGeneratorForVariable'));
    return;
  }

  try {
    const generated = generateValueForRule(rule);
    envValues.value[envVar] = generated;
    toast.success(t('appDetail.generatedValue', { envVar }));
  } catch (error) {
    toast.error(error.message || t('appDetail.generationFailed'));
  }
}

// Functions
async function fetchApp() {
  try {
    const response = await fetch(`${apiUrl.value}/api/apps`);
    const data = await expectApiSuccess(response, t('appDetail.failedToLoadApps'));

    if (data.apps) {
      app.value = data.apps.find((a) => a.id === route.params.appname);

      if (!app.value) {
        toast.error(t('appDetail.appNotFound'));
        router.push("/apps");
      }
      clearLoadError('app');
    }
  } catch (error) {
    notifyLoadErrorOnce('app', error.message || t('appDetail.failedToLoadAppDetails'));
  }
}

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`);
    const data = await expectApiSuccess(response, "Failed to load app containers");
    containers.value = Array.isArray(data.containers) ? data.containers : [];
    clearLoadError('containers');
  } catch (error) {
    notifyLoadErrorOnce('containers', error.message || "Failed to load app containers");
  }
}

async function fetchImageDetails() {
  if (!app.value) return;

  try {
    loadingImages.value = true;
    const response = await fetch(`${apiUrl.value}/api/image-details/${app.value.id}`);
    const data = await expectApiSuccess(response, "Failed to load image details");
    imageDetails.value = Array.isArray(data.images) ? data.images : [];
    clearLoadError('images');
  } catch (error) {
    notifyLoadErrorOnce('images', error.message || "Failed to load image details");
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

    if (enableAuth.value) {
      if (!authPort.value || !authUsername.value || !authPassword.value) {
        toast.error("Please fill all authentication fields");
        deploying.value = false;
        return;
      }
      requestBody.auth = {
        enabled: true,
        port: authPort.value,
        username: authUsername.value,
        password: authPassword.value,
      };
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

    const result = await readJsonResponse(response);

    if (response.ok && result.success) {
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
      throw new Error(getApiErrorMessage(result, t('appDetail.deploymentFailed')));
    }
  } catch (error) {
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

        <!-- Right Column: Deployment Configuration -->
        <div class="lg:col-span-4">
          <div class="space-y-6 sticky top-24">
            
            <!-- Custom App Notice -->
            <div v-if="app.customapp" class="flex items-start gap-3 rounded-xl border border-purple-200 dark:border-purple-500/20 bg-purple-50 dark:bg-purple-500/10 px-4 py-3">
              <div class="w-2 h-2 rounded-full bg-purple-500 shrink-0 mt-1"></div>
              <div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-purple-600 dark:text-purple-400 mb-0.5">Custom App Built by Yantr Team</div>
                <p class="text-[11px] text-purple-700 dark:text-purple-300 leading-relaxed">This is a custom app created and maintained by the Yantr team. It uses a locally-built image, so automatic updates are disabled.</p>
              </div>
            </div>

            <!-- Configuration -->
            <div class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-5">
              <div class="flex items-center justify-between mb-5">
                <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">
                  {{ t('appDetail.configuration') }}
                </h2>
              </div>

              <div class="space-y-6">
              <!-- Environment Vars -->
              <div v-if="app.environment?.length > 0" class="space-y-4">
                <div v-for="env in app.environment" :key="env.envVar" class="space-y-1.5">
                  <label class="w-full text-[10px] font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-widest flex items-center justify-between">
                    {{ env.name }}
                    <div class="flex items-center gap-2">
                      <button
                        v-if="getGeneratorRule(env.envVar)"
                        @click="generateEnvValue(env.envVar)"
                        type="button"
                        class="text-[9px] font-bold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 transition-colors"
                      >
                        {{ t('appDetail.generate') }}
                      </button>
                      <span v-if="env.default" class="text-[9px] font-mono text-gray-400 dark:text-zinc-500 border border-gray-200 dark:border-zinc-800 px-1.5 py-0.5 rounded">{{ env.default }}</span>
                    </div>
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

                <!-- Basic Authentication -->
                <div class="rounded-lg border border-gray-200 dark:border-zinc-800 p-3 transition-colors bg-gray-50 dark:bg-zinc-900/30">
                   <div class="flex items-start gap-3">
                        <input type="checkbox" id="enable-auth" v-model="enableAuth" class="mt-0.5 w-4 h-4 shrink-0 rounded border-gray-300 dark:border-zinc-700 text-black dark:text-white focus:ring-black dark:focus:ring-white focus:ring-offset-0 cursor-pointer bg-transparent" />
                        <div class="flex-1">
                            <label for="enable-auth" class="block text-[11px] font-bold text-gray-900 dark:text-zinc-100 cursor-pointer uppercase tracking-wider">Enable Basic Auth</label>
                            <p class="text-[10px] text-gray-500 dark:text-zinc-500 mt-0.5">Protect this app with a username and password</p>
                        </div>
                   </div>

                    <div v-if="enableAuth" class="mt-4 pl-1 space-y-4">
                        <div class="space-y-1.5">
                            <label class="w-full text-[10px] font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-widest flex items-center justify-between">Proxy Port</label>
                            <input v-model.number="authPort" type="number" placeholder="e.g. 3002" class="w-full bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all" />
                        </div>
                        <div class="space-y-1.5">
                            <label class="w-full text-[10px] font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-widest flex items-center justify-between">Username</label>
                            <input v-model="authUsername" type="text" placeholder="admin" class="w-full bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all" />
                        </div>
                        <div class="space-y-1.5">
                            <label class="w-full text-[10px] font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-widest flex items-center justify-between">
                              Password
                              <button @click="generateAuthPassword" type="button" class="text-[9px] font-bold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 transition-colors">Generate</button>
                            </label>
                            <input v-model="authPassword" type="text" placeholder="Secret password" class="w-full bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all" />
                        </div>
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
