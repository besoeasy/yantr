<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useDeveloperProfile } from "../composables/useDeveloperProfile";
import { usePaymentRails } from "../composables/usePaymentRails";
import AppCard from "../components/AppCard.vue";
import NostrQrCode from "../components/NostrQrCode.vue";
import AppLayout from "../components/AppLayout.vue";
import {
  ArrowLeft,
  Bitcoin,
  Check,
  ExternalLink,
  Link2,
  Package,
  AlertCircle,
  AtSign,
  Copy,
  Zap,
} from "@lucide/vue";
import {
  DEVELOPER_LABEL,
  developerInitials,
  developerProfileUrl,
  nostrDmUri,
} from "../utils/developer";
// `apps` is already the list of apps this developer is credited on, so the
// catalog route builder is aliased rather than shadowing it.
import { appDetail, apps as appsRoute } from "../utils/routes.js";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();

const { developer, apps, profile, loading, notFound, failed, profileUnavailable, load } =
  useDeveloperProfile();

// Installed state, so a developer page shows the same card chrome as the
// catalog instead of a second, subtly different grid.
const containers = ref([]);

const developerRef = computed(() => String(route.params.npub ?? ""));

const key = computed(() => developer.value?.key ?? developerRef.value);

const profileUrl = computed(() => developerProfileUrl(key.value));

const displayName = computed(() => profile.value?.name?.trim() || key.value);
const initials = computed(() => developerInitials(key.value));
const avatarUrl = computed(() => profile.value?.avatarUrl ?? "");

// The NIP-21 URI a Nostr client opens to message this developer. Derived from
// the catalog's own key rather than from the profile request, so the code is
// on screen immediately and survives the relay being unreachable — which is the
// common case for a rootless host on a network that blocks it.
const nostrDm = computed(() => nostrDmUri(key.value));

// Only the key itself goes in the code's accessible name; the surrounding copy
// explains it.
const nostrQrLabel = computed(() =>
  t("developer.nostrQrLabel", { name: displayName.value })
);

const copied = ref(false);
let copiedTimer = null;

// The payment rails under the identity card: same composable the Developer
// of the day card renders, so an address that shows up here is one that
// validated — Lightning address, checksum, and all.
const { rails: paymentRails, copiedRail, copyAddress } = usePaymentRails(
  profile,
  "developer.payments"
);

async function copyNostrKey() {
  try {
    await navigator.clipboard.writeText(key.value);
    copied.value = true;
    // Reset rather than leaving the label stuck on "Copied"; a second copy of
    // the same value would otherwise look like nothing happened.
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch {
    // Clipboard access needs a secure context and can be refused. The key is
    // printed next to the button, so there is nothing to recover from.
  }
}

// Blog is free text on GitHub and people write it with and without a scheme.
const blogUrl = computed(() => {
  const raw = String(profile.value?.blog ?? "").trim();
  if (!raw) return "";
  return /^https?:\/\//i.test(raw) ? raw : `https://${raw}`;
});

const installedAppIds = computed(() => {
  const ids = new Set(containers.value.map((c) => c?.app?.id).filter(Boolean));
  return ids;
});

const appInstanceCounts = computed(() => {
  const projectsByApp = {};
  containers.value.forEach((c) => {
    const appId = c?.app?.id;
    const projectId = c?.app?.projectId;
    if (appId && projectId) {
      if (!projectsByApp[appId]) projectsByApp[appId] = new Set();
      projectsByApp[appId].add(projectId);
    }
  });
  const counts = {};
  for (const [appId, projects] of Object.entries(projectsByApp)) {
    counts[appId] = projects.size;
  }
  return counts;
});

const developerApps = computed(() =>
  apps.value.map((app) => ({ ...app, isInstalled: installedAppIds.value.has(app.id) }))
);

async function fetchContainers() {
  try {
    const res = await fetch(`${apiUrl.value}/api/containers`);
    const data = await res.json();
    containers.value = Array.isArray(data.containers)
      ? data.containers.filter((c) => c.state === "running")
      : [];
  } catch {
    // Installed badges are a nicety on this page. The catalogue half already
    // decided whether the page renders at all.
    containers.value = [];
  }
}

function viewAppDetail(appId) {
  router.push(appDetail(appId));
}

onMounted(() => {
  load(developerRef.value);
  fetchContainers();
});

// The copy feedback outlives a fast navigation otherwise, and firing a timer
// into an unmounted component is the kind of thing that shows up as a console
// warning in someone else's session.
onUnmounted(() => {
  if (copiedTimer) clearTimeout(copiedTimer);
});

// Navigating between two developer pages reuses this component, so the fetch
// cannot live only in onMounted.
watch(developerRef, (next) => {
  if (next) load(next);
});
</script>

<template>
  <AppLayout width="narrow" gap="loose">
      <!-- Back -->
      <button
        @click="router.push(appsRoute())"
        class="mb-6 inline-flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 transition-colors hover:text-gray-900 dark:text-zinc-500 dark:hover:text-zinc-200"
      >
        <ArrowLeft :size="13" />
        {{ t('developer.backToApps') }}
      </button>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-32">
        <div class="mb-6 h-8 w-8 animate-spin rounded-full border-[3px] border-zinc-200 border-t-blue-500 dark:border-zinc-800"></div>
        <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('developer.loading') }}</div>
      </div>

      <!-- Unknown developer -->
      <div v-else-if="notFound" class="flex flex-col items-center justify-center py-32 text-center">
        <div class="mb-5 rounded-2xl border border-zinc-200 bg-zinc-50 p-5 dark:border-zinc-800 dark:bg-zinc-900/50">
          <AlertCircle :size="28" class="text-zinc-400 dark:text-zinc-500" />
        </div>
        <h3 class="text-base font-semibold tracking-tight text-zinc-900 dark:text-white">{{ key }}</h3>
        <p class="mx-auto mt-2 max-w-sm text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">{{ t('developer.notFound') }}</p>
        <button @click="router.push(appsRoute())" class="mt-6 text-xs font-bold uppercase tracking-wider text-sky-600 transition-colors hover:text-zinc-900 dark:text-sky-400 dark:hover:text-zinc-300">
          {{ t('developer.backToApps') }}
        </button>
      </div>

      <!-- Request failed -->
      <div v-else-if="failed" class="flex flex-col items-center justify-center py-32 text-center">
        <div class="mb-5 rounded-2xl border border-zinc-200 bg-zinc-50 p-5 dark:border-zinc-800 dark:bg-zinc-900/50">
          <AlertCircle :size="28" class="text-zinc-400 dark:text-zinc-500" />
        </div>
        <p class="text-sm text-zinc-500 dark:text-zinc-400">{{ t('developer.loadFailed') }}</p>
      </div>

      <template v-else>
        <!-- Identity header -->
        <div class="yantra-card group relative flex flex-col gap-6 overflow-hidden p-6 transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] sm:flex-row dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
          <div class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
            <img v-if="avatarUrl" :src="avatarUrl" :alt="displayName" class="h-full w-full object-cover" />
            <span v-else class="text-2xl font-black tracking-tight text-zinc-900 dark:text-zinc-100">{{ initials }}</span>
          </div>

          <div class="flex flex-1 flex-col">
            <div class="mb-1 flex flex-wrap items-center gap-3">
              <h1 class="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">{{ displayName }}</h1>
              <span class="inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                {{ DEVELOPER_LABEL }}
              </span>
            </div>
            <p class="font-mono text-xs text-zinc-500">{{ key }}</p>

            <p v-if="profile?.bio" class="mt-4 max-w-2xl text-sm leading-relaxed text-zinc-500">{{ profile.bio }}</p>
            <p v-else-if="!profile" class="mt-4 max-w-2xl text-sm leading-relaxed text-zinc-500">
              {{ t('developer.appsBy', { name: displayName }) }}
            </p>

            <div v-if="profile" class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-zinc-500">
              <span v-if="profile.nip05" class="inline-flex items-center gap-1.5"><AtSign :size="13" /> {{ profile.nip05 }}</span>
              <span v-if="profile.lud16" class="inline-flex items-center gap-1.5"><Zap :size="13" /> {{ profile.lud16 }}</span>
              <span v-if="profile.bitcoin" class="inline-flex items-center gap-1.5"><Bitcoin :size="13" /> {{ profile.bitcoin }}</span>
              <a v-if="blogUrl" :href="blogUrl" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1.5 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200"><Link2 :size="13" /> {{ profile.blog }}</a>
            </div>

            <div class="mt-5 flex flex-wrap gap-x-8 gap-y-3">
              <div>
                <div class="text-base font-bold tabular-nums text-zinc-900 dark:text-white">{{ developerApps.length }}</div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.yantrApps') }}</div>
              </div>
            </div>

            <div class="mt-auto flex flex-wrap gap-3 pt-6">
              <a v-if="profileUrl" :href="profileUrl" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300">
                <ExternalLink :size="14" /> {{ t('developer.viewOn') }}
              </a>
              <button @click="router.push(appsRoute())" class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300">
                <Package :size="14" /> {{ t('developer.browseCatalog') }}
              </button>
            </div>
          </div>
        </div>

        <p v-if="profileUnavailable" class="flex items-start gap-2 text-[11px] leading-relaxed text-zinc-500">
          <AlertCircle :size="13" class="mt-0.5 shrink-0" />
          {{ t('developer.profileUnavailable') }}
        </p>

        <nav class="sticky top-0 z-10 flex flex-wrap items-center gap-2 border-b border-zinc-200 bg-white py-3 backdrop-blur-md dark:border-zinc-800 dark:bg-(--page-bg)">
          <a href="#apps" class="text-[10px] font-bold uppercase tracking-widest text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200">{{ t('developer.appsInYantr') }}</a>
          <a v-if="nostrDm" href="#message" class="text-[10px] font-bold uppercase tracking-widest text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200">{{ t('developer.nostrDm') }}</a>
          <a v-if="paymentRails.length" href="#support" class="text-[10px] font-bold uppercase tracking-widest text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200">{{ t('developer.payments.title') }}</a>
        </nav>

        <!-- Apps -->
        <section id="apps" class="border-t border-zinc-200 pt-6 dark:border-zinc-800">
          <div class="mb-4 flex items-end justify-between gap-4">
            <div>
              <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.appsInYantr') }}</h2>
              <p class="mt-1.5 text-xs text-zinc-500">{{ t('developer.appsInYantrHint') }}</p>
            </div>
            <span class="shrink-0 rounded border border-zinc-200 bg-zinc-50 px-2 py-0.5 font-mono text-[10px] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">{{ developerApps.length }}</span>
          </div>

          <div v-if="developerApps.length === 0" class="flex flex-col items-center gap-4 rounded-xl border border-zinc-200 bg-zinc-50 p-6 text-center dark:border-zinc-800 dark:bg-zinc-900/50">
            <p class="text-sm text-zinc-500">{{ t('developer.noApps') }}</p>
            <button @click="router.push(appsRoute())" class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300">
              <Package :size="14" /> {{ t('developer.browseCatalog') }}
            </button>
          </div>
          <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 anim-stagger">
            <AppCard
              v-for="app in developerApps"
              :key="app.id"
              :app="app"
              :instance-count="appInstanceCounts[app.id] || 0"
              @select="viewAppDetail(app.id)"
            />
          </div>
        </section>

        <!-- Nostr DM -->
        <section v-if="nostrDm" id="message" class="border-t border-zinc-200 pt-6 dark:border-zinc-800">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.nostrDm') }}</h2>
          <p class="mt-2 max-w-2xl text-sm leading-relaxed text-zinc-500">{{ t('developer.nostrDmHint') }}</p>

          <div class="mt-6 flex flex-wrap items-start gap-6 sm:flex-row sm:items-center">
            <div class="shrink-0 rounded-xl border border-zinc-200 bg-white p-2 dark:border-zinc-800 dark:bg-zinc-900">
              <NostrQrCode :value="nostrDm" :label="nostrQrLabel" :size="180" />
            </div>

            <div class="min-w-0 flex-1">
              <p class="break-all font-mono text-[11px] leading-relaxed text-zinc-500">{{ key }}</p>

              <div class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-3">
                <a v-if="profileUrl" :href="profileUrl" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300">
                  <ExternalLink :size="14" /> {{ t('developer.nostrOpen') }}
                </a>
                <button @click="copyNostrKey" class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300">
                  <Copy :size="14" /> {{ copied ? t('developer.nostrCopied') : t('developer.nostrCopy') }}
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- Payments -->
        <section v-if="paymentRails.length" id="support" class="border-t border-zinc-200 pt-6 dark:border-zinc-800">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.payments.title') }}</h2>
          <p class="mt-2 max-w-2xl text-sm leading-relaxed text-zinc-500">{{ t('developer.payments.hint') }}</p>

          <div class="mt-6 flex flex-wrap items-start gap-8 sm:gap-10">
            <div v-for="rail in paymentRails" :key="rail.kind" class="flex min-w-44 flex-col items-start">
              <span class="mb-3 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ rail.label }}</span>

              <div class="shrink-0 rounded-xl border border-zinc-200 bg-white p-2 dark:border-zinc-800 dark:bg-zinc-900">
                <NostrQrCode :value="rail.payload" :label="rail.qrLabel" :size="168" />
              </div>

              <p class="mt-3 max-w-64 break-all font-mono text-[11px] leading-relaxed text-zinc-500">{{ rail.address }}</p>

              <button @click="copyAddress(rail)" class="mt-3 inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300">
                <Check v-if="copiedRail === rail.kind" :size="14" />
                <Copy v-else :size="14" />
                {{ copiedRail === rail.kind ? t('developer.payments.copied') : t('developer.payments.copy') }}
              </button>
            </div>
          </div>
        </section>
      </template>
  </AppLayout>
</template>
