<script>
// Two columns, like every other widget on the dashboard. The card was a
// four-column band built around an editorial row — identity, pitch, payment
// rails side by side — which at this width no longer reads; stacked vertically
// it is one column of facts, and the rails sit two across under it.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Check, Copy } from "@lucide/vue";
import AppLogo from "../components/AppLogo.vue";
import NostrQrCode from "../components/NostrQrCode.vue";
import { useDeveloperProfile } from "../composables/useDeveloperProfile";
import { usePaymentRails } from "../composables/usePaymentRails";
import { useCatalog } from "../composables/useDashboardData";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";
import { appDetail, developerDetail } from "../utils/routes.js";
import {
  appsByDeveloper,
  dailyDeveloperPick,
  developerInitials,
  developerKeysFromApps,
  developerProfileUrl,
  developerShortKey,
} from "../utils/developer";

// One developer, one day, in full. The catalog answers who exists; the
// account's kind-0 answers who it is and where its Lightning address lives;
// the widget's own logic is only the pick and the layout. Like the app
// spotlight, nothing is server-chosen and nothing is personalised: the same
// day reads the same everywhere.

const { t } = useI18n();
const router = useRouter();

const { profile, load: loadDeveloper } = useDeveloperProfile();

// The catalog is shared with the other cards that list it, so the band stops
// paying for its own copy of /api/apps. The pick is computed rather than
// fetched: it follows the shared list whenever it lands, which is also what
// makes a cached catalog re-pick without a second request.
const { apps, appsLoaded, appsLoadFailed } = useCatalog();

// A read failure is not rendered as an error card: a promo band that silently
// leaves today's slot to the other widgets is honest about being promotion,
// where "failed to load" would read as the dashboard being broken. The pick
// resolves empty and the section renders nothing.
const pickedKey = computed(() => {
  if (!appsLoaded.value || appsLoadFailed.value) return "";
  return dailyDeveloperPick(developerKeysFromApps(apps.value));
});

const developerApps = computed(() => appsByDeveloper(apps.value, pickedKey.value));

// The chips show the account's catalog apps; four, because a line of five or
// more chips turns a promo band into a shelf.
const SHOWCASE_APP_LIMIT = 4;
const showcaseApps = computed(() => developerApps.value.slice(0, SHOWCASE_APP_LIMIT));
const hiddenAppCount = computed(() => Math.max(0, developerApps.value.length - SHOWCASE_APP_LIMIT));

const displayName = computed(() => profile.value?.name?.trim() || pickedKey.value);
const initials = computed(() => developerInitials(displayName.value));
const avatarUrl = computed(() => profile.value?.avatarUrl ?? "");

// The identity row shows the npub, not the hex: the abbreviated form is what a
// reader compares against one they already have.
const npubShort = computed(() => developerShortKey(pickedKey.value));

// The payment rails this account publishes, built by the shared composable
// the developer page also uses — same validation, same payloads, one place.
const { rails: payments, copiedRail, copyAddress } = usePaymentRails(
  profile,
  "home.dailyDeveloper"
);

// Follow and thank-you both open the account's page on njump: the profile
// landing offers every Nostr client the choice to follow or to message, which
// is "the reader's own client" as a URL, and it works on a desktop too.
const profileUrl = computed(() => developerProfileUrl(pickedKey.value));

function viewApp(appId) {
  router.push(appDetail(appId));
}

function viewDeveloper() {
  router.push(developerDetail(pickedKey.value));
}

// Profile, only once there is a pick: useDeveloperProfile carries the
// IndexedDB kind-0 cache and the verified-read fallbacks — nothing here to
// reimplement, at the cost of one redundant /api/developers request it makes
// itself and which also re-confirms the account really is catalog-credited.
watch(pickedKey, (next) => {
  if (next) loadDeveloper(next);
});

</script>

<template>
  <WidgetSkeleton v-if="!appsLoaded && !appsLoadFailed" />

  <section
    v-else-if="pickedKey"
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden rounded-2xl p-7 transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- One hairline accent, top edge. The card's colour budget is this line
         and nothing else: the identity and payment data should not have to
         compete with decoration. -->
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-sky-500/30 to-transparent"
    ></div>

    <div class="flex flex-1 flex-col gap-5">
      <!-- Identity -->
      <div>
        <p class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("home.dailyDeveloper.title") }}
        </p>

        <div class="mt-3 flex items-center gap-4">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-xl yantra-tile">
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              :alt="displayName"
              referrerpolicy="no-referrer"
              class="h-full w-full object-cover"
            />
            <span v-else class="text-lg font-black tracking-tight yantra-text-primary dark:text-zinc-100">
              {{ initials }}
            </span>
          </div>
          <div class="min-w-0">
            <button
              @click="viewDeveloper"
              class="block max-w-full truncate text-left text-xl font-bold tracking-tight yantra-text-primary transition-opacity hover:opacity-70 yantra-text-primary"
            >
              {{ displayName }}
            </button>
            <p :title="pickedKey" class="mt-0.5 truncate font-mono text-[11px] yantra-text-secondary">
              {{ npubShort }}
            </p>
          </div>
        </div>

        <div class="mt-3 flex items-baseline gap-1.5">
          <span class="text-xl font-bold tabular-nums yantra-text-primary">
            {{ developerApps.length }}
          </span>
          <span class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ t("home.dailyDeveloper.appsCount") }}
          </span>
        </div>
      </div>

      <!-- The pitch, backed by the apps -->
      <div class="flex min-w-0 flex-1 flex-col border-t border-zinc-100 pt-5 dark:border-zinc-900">
        <!-- Two lines of pitch, and the rest of the card is facts. The full text
             stays in the tooltip for anyone who wants to read it. -->
        <p
          class="line-clamp-2 max-w-2xl text-sm leading-relaxed text-zinc-600 yantra-text-secondary"
          :title="t('home.dailyDeveloper.pitch')"
        >
          {{ t("home.dailyDeveloper.pitch") }}
        </p>

        <div class="mt-auto flex flex-wrap items-center gap-2 pt-4">
          <button
            v-for="app in showcaseApps"
            :key="app.id"
            @click="viewApp(app.id)"
            class="inline-flex max-w-full items-center gap-2 rounded-full bg-zinc-50 py-1 pl-1 pr-3 text-xs font-medium text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
          >
            <span class="flex h-5 w-5 shrink-0 items-center justify-center overflow-hidden rounded-full">
              <AppLogo
                :logo="app.logo"
                :name="app.name"
                :seed="app.id"
                img-class="h-full w-full object-contain"
                icon-class="h-4 w-4 text-zinc-300 dark:text-zinc-700"
              />
            </span>
            <span class="truncate">{{ app.name }}</span>
          </button>
          <span
            v-if="hiddenAppCount > 0"
            :title="t('home.dailyDeveloper.appsCount')"
            class="rounded-full border-dashed px-3 py-1 text-xs font-medium yantra-text-secondary dark:border-zinc-800"
          >
            {{ t("home.dailyDeveloper.appsMore", { count: hiddenAppCount }) }}
          </span>
        </div>
      </div>

      <!-- Payments: every rail the account published, side by side. -->
      <div class="flex flex-col border-t border-zinc-100 pt-5 dark:border-zinc-900">
        <template v-if="payments.length">
          <p class="text-sm leading-relaxed text-zinc-600 yantra-text-secondary">
            {{ t("home.dailyDeveloper.zap") }}
          </p>

          <div class="mt-3 flex flex-row items-start gap-4">
            <div v-for="rail in payments" :key="rail.kind" class="flex min-w-0 flex-1 flex-col">
              <span class="mb-1.5 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
                {{ rail.label }}
              </span>

              <!-- The QR is black on white and stays that way in both colour
                   schemes — a themed code is the classic way to ship one that
                   never scans. -->
              <div class="w-fit rounded-lg p-2 dark:border-zinc-700">
                <NostrQrCode :value="rail.payload" :label="rail.qrLabel" :size="88" />
              </div>

              <!-- The address is not printed. The code is what you scan and the
                   button is what you copy, and a rail used to spend a
                   wrapped row of the card on text nobody reads by eye — the
                   Lightning address especially, since what the code carries
                   is an LNURL, not the address. It is still one hover away:
                   the button's tooltip is the address it will put on your
                   clipboard, which is also the answer to "what does this QR
                   actually pay?". -->
              <button
                @click="copyAddress(rail)"
                :title="rail.address"
                :aria-label="`${copiedRail === rail.kind ? t('home.dailyDeveloper.copied') : t('home.dailyDeveloper.copy')} ${rail.label}`"
                class="mt-1.5 inline-flex w-fit items-center gap-1.5 rounded-full bg-zinc-900 px-3 py-1.5 text-[11px] font-semibold yantra-text-primary transition-opacity hover:opacity-90 dark:bg-zinc-100 yantra-text-primary"
              >
                <Check v-if="copiedRail === rail.kind" :size="13" />
                <Copy v-else :size="13" />
                {{ copiedRail === rail.kind ? t("home.dailyDeveloper.copied") : t("home.dailyDeveloper.copy") }}
              </button>
            </div>
          </div>
        </template>

        <template v-else>
          <p class="text-sm leading-relaxed text-zinc-600 yantra-text-secondary">
            {{ t("home.dailyDeveloper.noAddress") }}
          </p>
        </template>

        <!-- One link, and it says what it does. This used to be two pills —
             "Follow on Nostr" and "Send a thank-you" — with the same href
             behind both, which spent 36px of the card's height saying the
             same thing twice. -->
        <a
          v-if="profileUrl"
          :href="profileUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="mt-3 inline-flex w-full items-center justify-center gap-1.5 rounded-full border border-zinc-200 px-3.5 py-2 text-[11px] font-semibold yantra-text-secondary transition-colors hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-800 dark:hover:border-zinc-700 dark:hover:text-zinc-100"
        >
          {{ t("home.dailyDeveloper.follow") }}
          <ArrowUpRight :size="13" />
        </a>
      </div>
    </div>
  </section>
</template>
