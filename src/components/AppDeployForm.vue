<script setup>
import { ref, computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useNotification } from "../composables/useNotification";
import { useApiUrl } from "../composables/useApiUrl";
import { getApiErrorMessage, readJsonResponse } from "../composables/useApiResponse";
import { usePortConflict } from "../composables/usePortConflict";
import { AlertTriangle, Check, Play, Plus, X } from "lucide-vue-next";

const props = defineProps({
  app: {
    type: Object,
    required: true
  },
  containers: {
    type: Array,
    required: true
  }
});

const router = useRouter();
const toast = useNotification();
const { apiUrl } = useApiUrl();
const { t } = useI18n();

// State
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
const authTargetPort = ref(0);

// Computed
const canDeploy = computed(() => !deploying.value);

const instanceCount = computed(() => {
  return props.containers.filter((c) => c.app.id === props.app.id).length;
});

const nextInstanceNumber = computed(() => instanceCount.value + 1);

const infoPorts = computed(() => {
  return Array.isArray(props.app?.ports) ? props.app.ports : [];
});

const allPorts = computed(() => {
  return Array.isArray(props.app?.composePorts) ? props.app.composePorts : [];
});

// Port conflict detection
const { getPortStatus: getPortStatusFn } = usePortConflict(computed(() => props.containers));

function getPortStatus(port) {
  return getPortStatusFn(port, customPortMappings.value);
}

// Helpers
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

function getGeneratorRule(envVar) {
  const rules = props.app?.envGenerators;
  if (!rules || typeof rules !== "object") return null;
  const rule = rules[envVar];
  if (!rule || typeof rule !== "object" || Array.isArray(rule)) return null;
  return rule;
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
  toast.info(t('appDetail.deployingApp', { name: props.app.name, suffix: instanceSuffix }));

  try {
    const extraEnv = {};
    for (const row of extraEnvRows.value) {
      const k = row.key.trim();
      if (k) extraEnv[k] = row.value;
    }

    const requestBody = {
      appId: props.app.id,
      environment: envValues.value,
      extraEnv,
      instanceId: instanceNum,
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
        targetPort: authTargetPort.value || 0,
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
        toast.success(t('appDetail.deployedAsTemporary', { name: props.app.name, hours: expirationHours.value }));
      } else {
        toast.success(t('appDetail.installedSuccessfully', { name: props.app.name }));
      }

      setTimeout(() => {
        router.push("/");
      }, 1500);
    } else {
      throw new Error(getApiErrorMessage(result, t('appDetail.deploymentFailed')));
    }
  } catch (error) {
    if (error.message.includes("timeout")) {
      toast.error(t('appDetail.deploymentTimeout', { name: props.app.name }));
    } else {
      toast.error(t('appDetail.deploymentFailedMessage', { message: error.message }));
    }
  } finally {
    deploying.value = false;
  }
}
</script>

<template>
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
                      <label class="w-full text-[10px] font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-widest flex items-center justify-between">Proxy Port
                        <span class="text-[9px] font-normal normal-case tracking-normal text-gray-400 dark:text-zinc-500">Caddy listens here (public)</span>
                      </label>
                      <input v-model.number="authPort" type="number" placeholder="e.g. 3002" class="w-full bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all" />
                  </div>
                  <!-- Target Port: shown when app exposes multiple ports -->
                  <div v-if="infoPorts.length > 0" class="space-y-1.5">
                      <label class="w-full text-[10px] font-bold text-gray-700 dark:text-zinc-300 uppercase tracking-widest flex items-center justify-between">Target Port
                        <span class="text-[9px] font-normal normal-case tracking-normal text-gray-400 dark:text-zinc-500">App's internal port to proxy to</span>
                      </label>
                      <select v-model.number="authTargetPort" class="w-full bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-gray-900 dark:text-white focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all cursor-pointer">
                        <option :value="0">Auto-detect (first HTTP port)</option>
                        <option v-for="p in infoPorts" :key="p.port" :value="p.port">
                          :{{ p.port }} — {{ p.label }} ({{ p.protocol }})
                        </option>
                      </select>
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
</template>
