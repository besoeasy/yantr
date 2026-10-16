<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { MessageSquare, RefreshCw, Send, KeyRound, AlertCircle, ExternalLink } from "@lucide/vue";
import { useAppComments } from "../composables/useAppComments";
import { developerInitials, developerProfileUrl, developerShortKey } from "../utils/developer";
import { hexToNpub } from "../utils/nostr";

const props = defineProps({
  app: { type: Object, required: true },
});

const { t } = useI18n();

const appId = computed(() => String(props.app?.id ?? "").trim().toLowerCase());

const {
  comments,
  loading,
  loadFailed,
  relaysReached,
  relaysTotal,
  publishing,
  publishError,
  pubkeyHex,
  signerMissing,
  shortAuthor,
  maxLength,
  load,
  connect,
  publish,
  eventTag,
} = useAppComments(appId.value);

const draft = ref("");
const published = ref(false);

const remaining = computed(() => maxLength - draft.value.length);
const canPost = computed(
  () => draft.value.trim().length > 0 && draft.value.length <= maxLength && !publishing.value
);

const errorText = computed(() => {
  switch (publishError.value) {
    case "empty":
      return t("appComments.emptyComment");
    case "too-long":
      return t("appComments.tooLong", { max: maxLength });
    case "sign-cancelled":
      return t("appComments.signCancelled");
    case "sign-error":
    case "signer-error":
      return t("appComments.signerError");
    case "publish-failed":
      return t("appComments.publishFailed");
    default:
      return publishError.value ? t("appComments.publishFailed") : "";
  }
});

function formatTime(createdAt) {
  try {
    return new Date(Number(createdAt) * 1000).toLocaleString();
  } catch {
    return "";
  }
}

function authorNpubOf(event) {
  return hexToNpub(event?.pubkey);
}

function authorShortOf(event) {
  const npub = authorNpubOf(event);
  return npub ? developerShortKey(npub) : t("appComments.anonymous");
}

function authorUrlOf(event) {
  return developerProfileUrl(authorNpubOf(event));
}

async function submit() {
  published.value = false;
  const ok = await publish(draft.value);
  if (ok) {
    draft.value = "";
    published.value = true;
  }
}

onMounted(() => {
  if (appId.value) load();
});

// Navigating between two app pages reuses the view, so the fetch cannot live
// only in onMounted.
watch(appId, () => {
  draft.value = "";
  published.value = false;
  pubkeyHex.value = null;
  if (appId.value) load();
});
</script>

<template>
  <section
    class="space-y-4 rounded-2xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-[#0A0A0A]"
  >
    <div class="flex items-center justify-between gap-4">
      <div class="flex items-center gap-2.5">
        <MessageSquare :size="15" class="text-zinc-400" />
        <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
          {{ t("appComments.title") }}
        </h3>
        <span
          v-if="!loading"
          class="rounded border border-zinc-200 bg-zinc-50 px-2 py-0.5 font-mono text-[10px] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
        >
          {{ comments.length }}
        </span>
      </div>
      <button
        @click="load"
        :disabled="loading"
        class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors hover:text-zinc-900 disabled:opacity-50 dark:hover:text-zinc-300"
      >
        <RefreshCw :size="13" :class="{ 'animate-spin': loading }" />
        {{ t("appComments.refresh") }}
      </button>
    </div>

    <p class="text-xs leading-relaxed text-zinc-500">
      {{ t("appComments.subtitle", { tag: `#yantr #${eventTag()}` }) }}
    </p>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-8">
      <div
        class="mb-4 h-6 w-6 animate-spin rounded-full border-[3px] border-zinc-200 border-t-zinc-900 dark:border-zinc-800 dark:border-t-white"
      ></div>
      <div class="text-[10px] font-bold uppercase tracking-widest text-zinc-400">
        {{ t("appComments.loading") }}
      </div>
    </div>

    <!-- Relays unreachable -->
    <div
      v-else-if="loadFailed"
      class="flex items-start gap-2 rounded-xl border border-zinc-200 bg-zinc-50 p-4 text-xs leading-relaxed text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
    >
      <AlertCircle :size="14" class="mt-0.5 shrink-0" />
      <span>{{ t("appComments.loadFailed") }}</span>
    </div>

    <template v-else>
      <!-- Empty state -->
      <p v-if="comments.length === 0" class="py-4 text-center text-sm text-zinc-500">
        {{ t("appComments.empty") }}
      </p>

      <!-- Comment list. Content renders as plain text (interpolation escapes),
           never v-html: a relay can serve any bytes and must not become script. -->
      <ul v-else class="divide-y divide-zinc-100 dark:divide-zinc-800">
        <li v-for="c in comments" :key="c.id" class="flex gap-3 py-4 first:pt-0 last:pb-0">
          <span
            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-zinc-900 text-[10px] font-black text-white dark:bg-zinc-100 dark:text-zinc-900"
          >
            {{ developerInitials(authorNpubOf(c) || c.pubkey) }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-baseline gap-x-2">
              <a
                v-if="authorUrlOf(c)"
                :href="authorUrlOf(c)"
                target="_blank"
                rel="noopener noreferrer"
                :title="authorNpubOf(c)"
                class="inline-flex items-center gap-1 truncate font-mono text-[11px] font-semibold text-zinc-700 hover:text-zinc-900 dark:text-zinc-300 dark:hover:text-zinc-100"
              >
                {{ authorShortOf(c) }}
                <ExternalLink :size="11" class="shrink-0 opacity-50" />
              </a>
              <span v-else class="truncate font-mono text-[11px] font-semibold text-zinc-700 dark:text-zinc-300">
                {{ authorShortOf(c) }}
              </span>
              <span class="shrink-0 font-mono text-[10px] text-zinc-400">
                {{ formatTime(c.created_at) }}
              </span>
            </div>
            <p class="mt-1 whitespace-pre-wrap break-words text-sm leading-relaxed text-zinc-600 dark:text-zinc-400">
              {{ c.content }}
            </p>
          </div>
        </li>
      </ul>

      <p v-if="!loadFailed && !loading" class="font-mono text-[10px] text-zinc-400">
        {{ t("appComments.relayStatus", { reached: relaysReached, total: relaysTotal }) }}
      </p>
    </template>

    <!-- Composer -->
    <div class="border-t border-zinc-100 pt-4 dark:border-zinc-900">
      <div v-if="!pubkeyHex">
        <button
          @click="connect"
          class="inline-flex items-center gap-1.5 rounded-xl border border-zinc-200 bg-zinc-900 px-4 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition-colors hover:bg-zinc-700 dark:border-zinc-700 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
        >
          <KeyRound :size="14" />
          {{ t("appComments.connect") }}
        </button>
        <p class="mt-2 text-[11px] leading-relaxed text-zinc-500">
          {{ t("appComments.connectHint") }}
        </p>
        <p
          v-if="signerMissing"
          class="mt-2 flex items-start gap-2 text-[11px] leading-relaxed text-zinc-500"
        >
          <AlertCircle :size="13" class="mt-0.5 shrink-0" />
          {{ t("appComments.noSigner") }}
        </p>
      </div>

      <div v-else class="space-y-3">
        <p class="font-mono text-[11px] text-zinc-500">
          {{ t("appComments.connectedAs", { key: shortAuthor }) }}
        </p>
        <label :for="`comment-${appId}`" class="sr-only">{{ t("appComments.placeholder") }}</label>
        <textarea
          :id="`comment-${appId}`"
          v-model="draft"
          :maxlength="maxLength"
          rows="3"
          :placeholder="t('appComments.placeholder')"
          class="w-full resize-y rounded-xl border border-zinc-200 bg-white p-3 text-sm leading-relaxed text-zinc-900 placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-100 dark:focus:border-zinc-600"
        />
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="font-mono text-[10px] text-zinc-400">{{ remaining }}</span>
          <button
            @click="submit"
            :disabled="!canPost"
            class="inline-flex items-center gap-1.5 rounded-xl border border-zinc-200 bg-zinc-900 px-4 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition-colors hover:bg-zinc-700 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
          >
            <Send :size="14" />
            {{ publishing ? t("appComments.posting") : t("appComments.post") }}
          </button>
        </div>
        <p v-if="errorText" class="flex items-start gap-2 text-[11px] leading-relaxed text-red-500">
          <AlertCircle :size="13" class="mt-0.5 shrink-0" />
          {{ errorText }}
        </p>
        <p v-if="published" class="text-[11px] leading-relaxed text-green-600 dark:text-green-400">
          {{ t("appComments.published") }}
        </p>
      </div>

      <p class="mt-3 text-[11px] leading-relaxed text-zinc-400">
        {{ t("appComments.expiryNote") }}
      </p>
    </div>
  </section>
</template>
