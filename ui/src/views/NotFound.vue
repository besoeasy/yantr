<script setup>
// Shown for any hash the router does not recognise. Without a catch-all route
// vue-router matched nothing and rendered an empty <main>, so a stale or
// mistyped bookmark looked like a broken app rather than a wrong URL.
//
// The layout follows StackView's load-failure panel on purpose: one shape for
// "this page has nothing to show you", instead of a fourth look.
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { AlertTriangle, LayoutGrid } from "@lucide/vue";
import { dashboard } from "../utils/routes.js";

const route = useRoute();
const { t } = useI18n();
</script>

<template>
  <main class="mx-auto flex max-w-7xl flex-col items-center gap-4 px-6 py-32 text-center">
    <AlertTriangle :size="28" class="yantra-text-danger" />
    <p class="text-sm font-semibold yantra-text-primary">{{ t("notFound.title") }}</p>
    <p class="max-w-md break-all text-xs yantra-text-secondary">
      {{ t("notFound.hint", { path: route.fullPath }) }}
    </p>
    <router-link :to="dashboard()" class="yantra-btn yantra-btn--primary mt-1">
      <LayoutGrid :size="13" />{{ t("notFound.backToDashboard") }}
    </router-link>
  </main>
</template>
