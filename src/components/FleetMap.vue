<script setup>
import { nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import L from "leaflet";
import "leaflet/dist/leaflet.css";

const props = defineProps({
  countries: {
    type: Array,
    default: () => [],
  },
});

const mapEl = ref(null);
let map = null;
let markers = null;
let tiles = null;
let themeObserver = null;

function isDark() {
  return document.documentElement.classList.contains("dark");
}

function tileUrl() {
  return isDark()
    ? "https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
    : "https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
}

function ensureMap() {
  if (map || !mapEl.value) return;
  map = L.map(mapEl.value, {
    worldCopyJump: true,
    minZoom: 1,
    maxZoom: 6,
    zoomControl: true,
    attributionControl: true,
  }).setView([20, 0], 2);

  tiles = L.tileLayer(tileUrl(), {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>',
  }).addTo(map);

  markers = L.layerGroup().addTo(map);
  requestAnimationFrame(() => map.invalidateSize());
}

function renderMarkers() {
  if (!map || !markers) return;
  markers.clearLayers();
  const points = (props.countries || []).filter(
    (item) => item.code && item.code !== "??" && (item.lat || item.lng)
  );
  const max = points.reduce((n, item) => Math.max(n, item.count || 0), 1);

  for (const item of points) {
    const radius = 8 + ((item.count || 0) / max) * 18;
    L.circleMarker([item.lat, item.lng], {
      radius,
      color: "#0284c7",
      fillColor: "#0ea5e9",
      fillOpacity: 0.55,
      weight: 2,
    })
      .bindTooltip(
        `<strong>${item.name}</strong><br>${item.count} ${item.count === 1 ? "node" : "nodes"}`,
        { sticky: true, direction: "top" }
      )
      .addTo(markers);
  }

  if (points.length === 1) {
    map.setView([points[0].lat, points[0].lng], 3);
  } else if (points.length > 1) {
    map.fitBounds(
      L.latLngBounds(points.map((item) => [item.lat, item.lng])),
      { padding: [28, 28], maxZoom: 4 }
    );
  }
}

function swapTiles() {
  if (!map) return;
  if (tiles) map.removeLayer(tiles);
  tiles = L.tileLayer(tileUrl(), {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>',
  }).addTo(map);
}

onMounted(async () => {
  await nextTick();
  ensureMap();
  renderMarkers();
  themeObserver = new MutationObserver(() => swapTiles());
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
});

onUnmounted(() => {
  themeObserver?.disconnect();
  map?.remove();
  map = null;
  markers = null;
  tiles = null;
});

watch(() => props.countries, renderMarkers, { deep: true });
</script>

<template>
  <div class="overflow-hidden rounded-2xl border border-zinc-200 dark:border-zinc-800">
    <div ref="mapEl" class="h-[420px] w-full bg-zinc-100 dark:bg-zinc-950"></div>
  </div>
</template>

<style>
.leaflet-container {
  font-family: inherit;
  background: transparent;
}
.leaflet-tooltip {
  border: 0;
  border-radius: 8px;
  padding: 6px 10px;
  box-shadow: 0 8px 24px rgb(0 0 0 / 0.12);
}
</style>
