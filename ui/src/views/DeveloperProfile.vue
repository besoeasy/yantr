<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useDeveloperProfile } from "../composables/useDeveloperProfile";
import AppCard from "../components/AppCard.vue";
import NostrQrCode from "../components/NostrQrCode.vue";
import {
  ArrowLeft,
  ExternalLink,
  MapPin,
  Building2,
  Link2,
  Star,
  GitFork,
  Package,
  AlertCircle,
  Code2,
  AtSign,
  Copy,
  Zap,
} from "@lucide/vue";
import {
  developerInitials,
  developerLabel,
  developerProfileUrl,
  nostrDmUri,
} from "../utils/developer";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();

const { developer, apps, profile, repos, loading, notFound, failed, profileUnavailable, load } =
  useDeveloperProfile();

// Installed state, so a developer page shows the same card chrome as the
// catalog instead of a second, subtly different grid.
const containers = ref([]);

const developerRef = computed(() => String(route.params.handle ?? ""));

const platform = computed(() => developer.value?.platform ?? "");
const handle = computed(() => developer.value?.handle ?? developerRef.value);
const platformLabel = computed(() => developerLabel(platform.value));

const profileUrl = computed(() => {
  if (profile.value?.htmlUrl) return profile.value.htmlUrl;
  return developerProfileUrl(platform.value, handle.value);
});

const displayName = computed(() => profile.value?.name?.trim() || handle.value);
const initials = computed(() => developerInitials(handle.value));
const avatarUrl = computed(() => profile.value?.avatarUrl ?? "");

// The NIP-21 URI a Nostr client opens to message this developer. Derived from
// the catalog's own handle rather than from the profile request, so the code is
// on screen immediately and survives the relay being unreachable — which is the
// common case for a rootless host on a network that blocks it.
const nostrDm = computed(() => (platform.value === "nostr" ? nostrDmUri(handle.value) : ""));

// Only the key itself goes in the code's accessible name; the surrounding copy
// explains it.
const nostrQrLabel = computed(() =>
  t("developer.nostrQrLabel", { name: displayName.value })
);

const copied = ref(false);
let copiedTimer = null;

async function copyNostrKey() {
  try {
    await navigator.clipboard.writeText(handle.value);
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
  router.push(`/apps/${appId}`);
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
  <div class="min-h-screen bg-white text-gray-900 dark:bg-[#0A0A0A] dark:text-zinc-100 font-sans">
    <div class="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 sm:py-10">

      <!-- Back -->
      <button
        @click="router.push('/apps')"
        class="mb-6 inline-flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 transition-colors hover:text-gray-900 dark:text-zinc-500 dark:hover:text-zinc-200"
      >
        <ArrowLeft :size="13" />
        {{ t('developer.backToApps') }}
      </button>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-32">
        <div class="mb-6 h-8 w-8 animate-spin rounded-full border-[3px] border-gray-200 border-t-blue-500 dark:border-zinc-800"></div>
        <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('developer.loading') }}</div>
      </div>

      <!-- Unknown developer -->
      <div v-else-if="notFound" class="flex flex-col items-center justify-center py-32 text-center">
        <div class="mb-5 rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-zinc-800 dark:bg-zinc-900/50">
          <AlertCircle :size="28" class="text-gray-400 dark:text-zinc-500" />
        </div>
        <h3 class="text-base font-semibold tracking-tight text-gray-900 dark:text-white">{{ handle }}</h3>
        <p class="mx-auto mt-2 max-w-sm text-sm leading-relaxed text-gray-500 dark:text-zinc-400">{{ t('developer.notFound') }}</p>
        <button @click="router.push('/apps')" class="mt-6 text-xs font-bold uppercase tracking-wider text-blue-600 transition-colors hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300">
          {{ t('developer.backToApps') }}
        </button>
      </div>

      <!-- Request failed -->
      <div v-else-if="failed" class="flex flex-col items-center justify-center py-32 text-center">
        <div class="mb-5 rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-zinc-800 dark:bg-zinc-900/50">
          <AlertCircle :size="28" class="text-gray-400 dark:text-zinc-500" />
        </div>
        <p class="text-sm text-gray-500 dark:text-zinc-400">{{ t('developer.loadFailed') }}</p>
      </div>

      <template v-else>
        <!-- Identity -->
        <div class="flex flex-col gap-6 rounded-2xl border border-zinc-200 bg-white p-6 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] sm:flex-row dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
          <div class="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
            <img v-if="avatarUrl" :src="avatarUrl" :alt="displayName" class="h-full w-full object-cover" />
            <span v-else class="text-2xl font-black tracking-tight text-zinc-900 dark:text-zinc-100">{{ initials }}</span>
          </div>

          <div class="flex flex-1 flex-col">
            <div class="mb-1 flex flex-wrap items-center gap-3">
              <h1 class="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">{{ displayName }}</h1>
              <span class="inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                {{ platformLabel }}
              </span>
            </div>
            <p class="font-mono text-xs text-zinc-500">{{ platform }}:{{ handle }}</p>

            <p v-if="profile?.bio" class="mt-4 max-w-2xl text-sm leading-relaxed text-zinc-500">{{ profile.bio }}</p>
            <p v-else-if="!profile" class="mt-4 max-w-2xl text-sm leading-relaxed text-zinc-500">
              {{ t('developer.appsBy', { name: displayName }) }}
            </p>

            <!-- Profile facts, when the platform gave us any -->
            <div v-if="profile" class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-zinc-500">
              <span v-if="profile.company" class="inline-flex items-center gap-1.5">
                <Building2 :size="13" /> {{ profile.company }}
              </span>
              <span v-if="profile.location" class="inline-flex items-center gap-1.5">
                <MapPin :size="13" /> {{ profile.location }}
              </span>
              <!-- NIP-05 is the Nostr equivalent of a verified handle, so it is
                   shown as written rather than linked: verifying it means asking
                   the domain, which is not this page's job. -->
              <span v-if="profile.nip05" class="inline-flex items-center gap-1.5">
                <AtSign :size="13" /> {{ profile.nip05 }}
              </span>
              <span v-if="profile.lud16" class="inline-flex items-center gap-1.5">
                <Zap :size="13" /> {{ profile.lud16 }}
              </span>
              <a v-if="blogUrl" :href="blogUrl" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1.5 transition-colors hover:text-zinc-900 dark:hover:text-zinc-200">
                <Link2 :size="13" /> {{ profile.blog }}
              </a>
            </div>

            <!-- Counters. The repository and follower ones are GitHub's; a Nostr
                 profile has no equivalent, and normaliseNostrProfile leaves them
                 undefined so they stay hidden rather than reading zero. -->
            <div class="mt-5 flex flex-wrap gap-x-8 gap-y-3">
              <div>
                <div class="text-base font-bold tabular-nums text-zinc-900 dark:text-white">{{ developerApps.length }}</div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.yantrApps') }}</div>
              </div>
              <div v-if="profile?.publicRepos != null">
                <div class="text-base font-bold tabular-nums text-zinc-900 dark:text-white">{{ profile.publicRepos }}</div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.repositories') }}</div>
              </div>
              <div v-if="profile?.followers != null">
                <div class="text-base font-bold tabular-nums text-zinc-900 dark:text-white">{{ profile.followers }}</div>
                <div class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.followers') }}</div>
              </div>
            </div>

            <div class="mt-auto flex flex-wrap gap-3 pt-6">
              <a
                v-if="profileUrl"
                :href="profileUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
              >
                <ExternalLink :size="14" /> {{ t('developer.viewOn', { platform: platformLabel }) }}
              </a>
              <button
                @click="router.push('/apps')"
                class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
              >
                <Package :size="14" /> {{ t('developer.browseCatalog') }}
              </button>
            </div>
          </div>
        </div>

        <!-- Profile enrichment unavailable: the page keeps working, and says why -->
        <p
          v-if="profileUnavailable"
          class="mt-4 flex items-start gap-2 text-[11px] leading-relaxed text-zinc-500"
        >
          <AlertCircle :size="13" class="mt-0.5 shrink-0" />
          {{ t('developer.profileUnavailable') }}
        </p>

        <!-- Direct message on Nostr. The code is built from the catalog's own
             key, so it is on screen as soon as the page is and stays there when
             the relay is unreachable — which is the common case for a rootless
             host on a network that blocks it. -->
        <section
          v-if="nostrDm"
          class="mt-6 overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-[#0A0A0A]"
        >
          <div class="flex flex-col items-start gap-6 p-6 sm:flex-row sm:items-center">
            <div class="shrink-0 rounded-xl border border-zinc-200 bg-white p-2 dark:border-zinc-800">
              <!-- 180px over a 45-module drawing is four pixels per module, which
                   is what keeps it readable off a screen at arm's length. -->
              <NostrQrCode :value="nostrDm" :label="nostrQrLabel" :size="180" />
            </div>

            <div class="min-w-0 flex-1">
              <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.nostrDm') }}</h2>
              <p class="mt-2 max-w-md text-sm leading-relaxed text-zinc-500">{{ t('developer.nostrDmHint') }}</p>

              <p class="mt-3 break-all font-mono text-[11px] leading-relaxed text-zinc-500">{{ handle }}</p>

              <div class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-3">
                <!-- The non-scanning path, for anyone reading this on a desktop:
                     njump offers the reader's own client and a message link. -->
                <a
                  v-if="profileUrl"
                  :href="profileUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
                >
                  <ExternalLink :size="14" /> {{ t('developer.nostrOpen') }}
                </a>
                <button
                  @click="copyNostrKey"
                  class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 dark:hover:text-zinc-300"
                >
                  <Copy :size="14" /> {{ copied ? t('developer.nostrCopied') : t('developer.nostrCopy') }}
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- Apps in Yantr -->
        <section class="mt-10">
          <div class="mb-4 flex items-end justify-between gap-4">
            <div>
              <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.appsInYantr') }}</h2>
              <p class="mt-1.5 text-xs text-zinc-500">{{ t('developer.appsInYantrHint') }}</p>
            </div>
            <span class="shrink-0 rounded border border-zinc-200 bg-zinc-50 px-2 py-0.5 font-mono text-[10px] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">{{ developerApps.length }}</span>
          </div>

          <p v-if="developerApps.length === 0" class="text-sm text-zinc-500">{{ t('developer.noApps') }}</p>
          <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            <AppCard
              v-for="app in developerApps"
              :key="app.id"
              :app="app"
              :instance-count="appInstanceCounts[app.id] || 0"
              @select="viewAppDetail(app.id)"
            />
          </div>
        </section>

        <!-- Other repositories -->
        <section v-if="repos.length > 0" class="mt-12">
          <div class="mb-4">
            <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('developer.otherRepos') }}</h2>
            <p class="mt-1.5 text-xs text-zinc-500">{{ t('developer.otherReposHint', { platform: platformLabel }) }}</p>
          </div>

          <div class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-[#0A0A0A]">
            <a
              v-for="repo in repos"
              :key="repo.name"
              :href="repo.htmlUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="flex flex-col gap-2 border-b border-zinc-100 p-5 transition-colors last:border-b-0 hover:bg-zinc-50 sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800 dark:hover:bg-zinc-900/50"
            >
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-mono text-sm font-bold text-zinc-900 dark:text-white">{{ repo.name }}</span>
                  <span v-if="repo.archived" class="rounded bg-zinc-100 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-widest text-zinc-500 dark:bg-zinc-800">{{ t('developer.archived') }}</span>
                </div>
                <p v-if="repo.description" class="mt-1 truncate text-xs text-zinc-500">{{ repo.description }}</p>
              </div>
              <div class="flex shrink-0 items-center gap-4 text-[11px] text-zinc-500">
                <span v-if="repo.language" class="inline-flex items-center gap-1 font-mono">
                  <Code2 :size="12" /> {{ repo.language }}
                </span>
                <span class="inline-flex items-center gap-1 tabular-nums">
                  <Star :size="12" /> {{ repo.stars }}
                </span>
                <span class="inline-flex items-center gap-1 tabular-nums">
                  <GitFork :size="12" /> {{ repo.forks }}
                </span>
              </div>
            </a>
          </div>
        </section>
      </template>
    </div>
  </div>
</template>
