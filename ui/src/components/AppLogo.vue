<script setup>
import { computed, ref, watch } from "vue";
import { Bot, Box, Database, Globe, Package, Server } from "@lucide/vue";

const props = defineProps({
  logo: {
    type: String,
    default: null,
  },
  name: {
    type: String,
    default: "App",
  },
  seed: {
    type: [String, Number],
    default: "",
  },
  imgClass: {
    type: String,
    default: "w-full h-full object-contain",
  },
  iconClass: {
    type: String,
    default: "w-full h-full text-[var(--text-secondary)]",
  },
  loading: {
    type: String,
    default: "lazy",
  },
});

const fallbackIcons = [Bot, Box, Database, Globe, Package, Server];
const imageFailed = ref(false);

watch(
  () => props.logo,
  () => {
    imageFailed.value = false;
  }
);

function normalizeLogoUrl(value) {
  if (typeof value !== "string") return null;
  const trimmed = value.trim();
  if (!trimmed) return null;

  // Relative paths (e.g. /api/apps/{id}/logo) — use as-is
  if (trimmed.startsWith("/")) return trimmed;

  return null;
}

function pickFallbackIndex(seedValue) {
  const source = String(seedValue || props.name || props.logo || "app");
  let hash = 0;
  for (let index = 0; index < source.length; index += 1) {
    hash = ((hash << 5) - hash + source.charCodeAt(index)) | 0;
  }
  return Math.abs(hash) % fallbackIcons.length;
}

const resolvedLogo = computed(() => normalizeLogoUrl(props.logo));
const fallbackIcon = computed(() => fallbackIcons[pickFallbackIndex(props.seed)]);

function handleError() {
  imageFailed.value = true;
}
</script>

<template>
  <img
    v-if="resolvedLogo && !imageFailed"
    :src="resolvedLogo"
    :alt="name"
    :class="imgClass"
    :loading="loading"
    @error="handleError"
  />
  <component
    :is="fallbackIcon"
    v-else
    :class="iconClass"
    aria-hidden="true"
  />
</template>