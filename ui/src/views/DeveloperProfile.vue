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
  developerShortKey,
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

const npubShort = computed(() => developerShortKey(key.value));
const displayName = computed(() => profile.value?.name?.trim() || npubShort.value);
const initials = computed(() => developerInitials(profile.value?.name?.trim() || key.value));
const avatarUrl = computed(() => profile.value?.avatarUrl ?? "");
const avatarError = ref(false);

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

const activeAppsCount = computed(() => {
  return developerApps.value.filter((app) => app.isInstalled && (appInstanceCounts.value[app.id] || 0) > 0).length;
});

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

function scrollToSection(id) {
  const el = document.getElementById(id);
  if (el) {
    el.scrollIntoView({ behavior: "smooth", block: "start" });
  }
}

onMounted(() => {
  avatarError.value = false;
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
  if (next) {
    avatarError.value = false;
    load(next);
  }
});
</script>

<template>
  <AppLayout width="reading" gap="loose">
    <!-- Back -->
    <button
      type="button"
      @click="router.push(appsRoute())"
      class="mb-6 inline-flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200"
    >
      <ArrowLeft :size="13" />
      {{ t('developer.backToApps') }}
    </button>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-32">
      <div class="mb-6 h-8 w-8 animate-spin rounded-full border-[3px] border-zinc-200 border-t-zinc-900 dark:border-zinc-800 dark:border-t-white"></div>
      <div class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.loading') }}</div>
    </div>

    <!-- Unknown developer -->
    <div v-else-if="notFound" class="flex flex-col items-center justify-center py-32 text-center">
      <div class="mb-5 rounded-2xl border border-zinc-200 bg-zinc-50 p-5 dark:border-zinc-800 dark:bg-zinc-900/50">
        <AlertCircle :size="28" class="text-zinc-400 dark:text-zinc-500" />
      </div>
      <h3 class="text-base font-semibold tracking-tight text-zinc-900 dark:text-white">{{ key }}</h3>
      <p class="mx-auto mt-2 max-w-sm text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">{{ t('developer.notFound') }}</p>
      <button
        type="button"
        @click="router.push(appsRoute())"
        class="mt-6 text-xs font-bold uppercase tracking-wider text-sky-600 transition-colors hover:text-zinc-900 dark:text-sky-400 dark:hover:text-zinc-300"
      >
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
      <!-- Identity Hero Card -->
      <div class="yantra-card group relative flex flex-col gap-6 overflow-hidden p-6 sm:p-8 transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
        <div
          aria-hidden="true"
          class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-fuchsia-500/30 to-transparent dark:via-fuchsia-500/40"
        ></div>

        <div class="flex flex-col sm:flex-row gap-6 items-start">
          <!-- Avatar -->
          <div class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-2xl border-2 border-fuchsia-200 bg-fuchsia-50 shadow-sm dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20">
            <img
              v-if="avatarUrl && !avatarError"
              :src="avatarUrl"
              :alt="displayName"
              referrerpolicy="no-referrer"
              @error="avatarError = true"
              class="h-full w-full object-cover"
            />
            <span v-else class="text-2xl font-black tracking-tight text-fuchsia-600 dark:text-fuchsia-400">
              {{ initials }}
            </span>
          </div>

          <!-- Main Details -->
          <div class="flex flex-1 flex-col min-w-0">
            <!-- Name & Developer Badge -->
            <div class="flex flex-wrap items-center gap-3">
              <h1 class="text-2xl sm:text-3xl font-black tracking-tight text-zinc-900 dark:text-white">
                {{ displayName }}
              </h1>
              <span class="inline-flex items-center rounded-full border border-fuchsia-200 bg-fuchsia-50 px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider text-fuchsia-700 dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20 dark:text-fuchsia-400">
                {{ DEVELOPER_LABEL }}
              </span>
            </div>

            <!-- Key Pill with Quick Copy & NIP-05 -->
            <div class="mt-2 flex flex-wrap items-center gap-2">
              <button
                type="button"
                @click="copyNostrKey"
                :title="key"
                class="group inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 bg-zinc-50 px-2.5 py-1 font-mono text-[11px] text-zinc-600 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-400 dark:hover:border-zinc-700 dark:hover:text-zinc-100"
              >
                <Check v-if="copied" :size="12" class="text-emerald-500" />
                <Copy v-else :size="12" class="opacity-70 group-hover:opacity-100" />
                <span>{{ copied ? t('developer.nostrCopied') : npubShort }}</span>
              </button>

              <!-- NIP-05 Badge -->
              <span
                v-if="profile?.nip05"
                class="inline-flex items-center gap-1.5 rounded-lg border border-fuchsia-200 bg-fuchsia-50 px-2.5 py-1 text-[11px] font-medium text-fuchsia-700 dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20 dark:text-fuchsia-400"
              >
                <AtSign :size="12" />
                <span>{{ profile.nip05 }}</span>
              </span>
            </div>

            <!-- Bio -->
            <p v-if="profile?.bio" class="mt-4 max-w-2xl text-sm leading-relaxed text-zinc-600 dark:text-zinc-400">
              {{ profile.bio }}
            </p>
            <p v-else class="mt-4 max-w-2xl text-sm leading-relaxed text-zinc-500">
              {{ t('developer.appsBy', { name: displayName }) }}
            </p>

            <!-- Links / Metadata -->
            <div v-if="profile" class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-zinc-500">
              <span v-if="profile.lud16" class="inline-flex items-center gap-1.5 font-mono">
                <Zap :size="13" class="text-amber-500" />
                <span>{{ profile.lud16 }}</span>
              </span>
              <span v-if="profile.bitcoin" class="inline-flex items-center gap-1.5 font-mono">
                <Bitcoin :size="13" class="text-amber-500" />
                <span>{{ profile.bitcoin }}</span>
              </span>
              <a
                v-if="blogUrl"
                :href="blogUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1.5 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200"
              >
                <Link2 :size="13" />
                <span>{{ profile.blog }}</span>
              </a>
            </div>

            <!-- Stats Row -->
            <div class="mt-6 flex flex-wrap items-center gap-6 border-t border-zinc-100 pt-5 dark:border-zinc-800">
              <div>
                <div class="text-xl font-black tabular-nums text-zinc-900 dark:text-white">
                  {{ developerApps.length }}
                </div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                  {{ t('developer.yantrApps') }}
                </div>
              </div>

              <div v-if="activeAppsCount > 0">
                <div class="flex items-center gap-2">
                  <span class="relative flex h-2 w-2">
                    <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
                    <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
                  </span>
                  <span class="text-xl font-black tabular-nums text-emerald-600 dark:text-emerald-400">
                    {{ activeAppsCount }}
                  </span>
                </div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-emerald-600 dark:text-emerald-400">
                  {{ t('home.dailyAppCard.running', { count: activeAppsCount }) }}
                </div>
              </div>
            </div>

            <!-- Actions -->
            <div class="mt-6 flex flex-wrap gap-3">
              <a
                v-if="profileUrl"
                :href="profileUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-2 rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
              >
                <ExternalLink :size="14" />
                {{ t('developer.viewOn') }}
              </a>
              <button
                type="button"
                @click="router.push(appsRoute())"
                class="inline-flex items-center gap-2 rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
              >
                <Package :size="14" />
                {{ t('developer.browseCatalog') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <p v-if="profileUnavailable" class="flex items-start gap-2 text-[11px] leading-relaxed text-zinc-500">
        <AlertCircle :size="13" class="mt-0.5 shrink-0" />
        {{ t('developer.profileUnavailable') }}
      </p>

      <!-- Section Navigation (Clean buttons with smooth scrolling) -->
      <nav class="sticky top-0 z-10 flex flex-wrap items-center gap-2 border-b border-zinc-200 bg-white py-3 backdrop-blur-md dark:border-zinc-800 dark:bg-(--page-bg)">
        <button
          type="button"
          @click="scrollToSection('apps')"
          class="rounded-lg px-3 py-1.5 text-[10px] font-bold uppercase tracking-widest text-zinc-600 transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
        >
          {{ t('developer.appsInYantr') }} ({{ developerApps.length }})
        </button>
        <button
          v-if="nostrDm"
          type="button"
          @click="scrollToSection('message')"
          class="rounded-lg px-3 py-1.5 text-[10px] font-bold uppercase tracking-widest text-zinc-600 transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
        >
          {{ t('developer.nostrDm') }}
        </button>
        <button
          v-if="paymentRails.length"
          type="button"
          @click="scrollToSection('support')"
          class="rounded-lg px-3 py-1.5 text-[10px] font-bold uppercase tracking-widest text-zinc-600 transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
        >
          {{ t('developer.payments.title') }}
        </button>
      </nav>

      <!-- Apps Section -->
      <section id="apps" class="border-t border-zinc-200 pt-6 dark:border-zinc-800">
        <div class="mb-4 flex items-end justify-between gap-4">
          <div>
            <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.appsInYantr') }}</h2>
            <p class="mt-1.5 text-xs text-zinc-500">{{ t('developer.appsInYantrHint') }}</p>
          </div>
          <span class="shrink-0 rounded border border-zinc-200 bg-zinc-50 px-2 py-0.5 font-mono text-[10px] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
            {{ developerApps.length }}
          </span>
        </div>

        <div v-if="developerApps.length === 0" class="flex flex-col items-center gap-4 rounded-xl border border-zinc-200 bg-zinc-50 p-6 text-center dark:border-zinc-800 dark:bg-zinc-900/50">
          <p class="text-sm text-zinc-500">{{ t('developer.noApps') }}</p>
          <button
            type="button"
            @click="router.push(appsRoute())"
            class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
          >
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

      <!-- Nostr DM Section -->
      <section v-if="nostrDm" id="message" class="border-t border-zinc-200 pt-6 dark:border-zinc-800">
        <div class="mb-4">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.nostrDm') }}</h2>
          <p class="mt-1 text-xs text-zinc-500">{{ t('developer.nostrDmHint') }}</p>
        </div>

        <div class="yantra-card overflow-hidden p-6 sm:p-8">
          <div class="flex flex-col sm:flex-row items-center gap-6">
            <div class="w-40 shrink-0 rounded-2xl bg-white p-3 border border-zinc-200 shadow-sm flex items-center justify-center dark:border-zinc-800 dark:bg-white">
              <NostrQrCode :value="nostrDm" :label="nostrQrLabel" :size="150" class="h-full w-full" />
            </div>

            <div class="min-w-0 flex-1">
              <p class="break-all font-mono text-[11px] leading-relaxed text-zinc-500">{{ key }}</p>

              <div class="mt-4 flex flex-wrap items-center gap-3">
                <a
                  v-if="profileUrl"
                  :href="profileUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
                >
                  <ExternalLink :size="14" /> {{ t('developer.nostrOpen') }}
                </a>
                <button
                  type="button"
                  @click="copyNostrKey"
                  class="inline-flex items-center gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
                >
                  <Check v-if="copied" :size="14" class="text-emerald-500" />
                  <Copy v-else :size="14" />
                  {{ copied ? t('developer.nostrCopied') : t('developer.nostrCopy') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Payments Section -->
      <section v-if="paymentRails.length" id="support" class="border-t border-zinc-200 pt-6 dark:border-zinc-800">
        <div class="mb-4">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.payments.title') }}</h2>
          <p class="mt-1 text-xs text-zinc-500">{{ t('developer.payments.hint') }}</p>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
          <div
            v-for="rail in paymentRails"
            :key="rail.kind"
            class="yantra-card p-6 flex flex-col items-start"
          >
            <div class="flex items-center gap-2 mb-4">
              <span class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ rail.label }}</span>
              <span class="rounded-md border border-amber-200 bg-amber-50 px-2 py-0.5 text-[9px] font-bold uppercase tracking-wider text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-400">
                {{ rail.kind === 'lightning' ? 'Lightning' : 'On-Chain' }}
              </span>
            </div>

            <div class="aspect-square w-40 rounded-2xl bg-white p-3 border border-zinc-200 shadow-sm flex items-center justify-center dark:border-zinc-800 dark:bg-white">
              <NostrQrCode :value="rail.payload" :label="rail.qrLabel" :size="140" class="h-full w-full" />
            </div>

            <p class="mt-4 max-w-64 break-all font-mono text-[11px] leading-relaxed text-zinc-500">{{ rail.address }}</p>

            <button
              type="button"
              @click="copyAddress(rail)"
              class="mt-4 inline-flex items-center gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
            >
              <Check v-if="copiedRail === rail.kind" :size="14" class="text-emerald-500" />
              <Copy v-else :size="14" />
              {{ copiedRail === rail.kind ? t('developer.payments.copied') : t('developer.payments.copy') }}
            </button>
          </div>
        </div>
      </section>
    </template>
  </AppLayout>
</template>
