<script setup>
import { ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useNotification } from '../composables/useNotification';
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";
import { useI18n } from "vue-i18n";
import AppMetadata from "../components/AppMetadata.vue";
import AppDeployForm from "../components/AppDeployForm.vue";

const route = useRoute();
const router = useRouter();
const toast = useNotification();
const { apiUrl } = useApiUrl();
const { t } = useI18n();

// State
const app = ref(null);
const containers = ref([]);
const loading = ref(true);
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
        <div class="lg:col-span-8">
          <AppMetadata :app="app" :imageDetails="imageDetails" />
        </div>

        <!-- Right Column: Deployment Configuration -->
        <div class="lg:col-span-4">
          <AppDeployForm :app="app" :containers="containers" />
        </div>

      </div>
    </div>
  </template>

<style scoped></style>
