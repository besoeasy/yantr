<script setup>
import { computed } from "vue";
import { buildQrSymbol, QR_QUIET_ZONE } from "../utils/qrcode";

// A scannable QR code, drawn from the symbol buildQrSymbol encodes.
//
// The drawing is done here rather than through the library's createSvgTag so the
// symbol stays a live SVG the page can size with CSS, nothing is handed to
// v-html, and the colours are ours to fix.
//
// Colour is the part worth being strict about: a reader wants dark modules on a
// light field, so this is black on white in both colour schemes, and the quiet
// zone is painted rather than inherited from the card behind it. A themed QR
// code — light modules on a dark card — is the classic way to ship one that
// never scans.

const props = defineProps({
  // The payload, a NIP-21 URI such as "nostr:npub1…".
  value: { type: String, required: true },
  // Rendered width and height in CSS pixels. The drawing scales, so this is a
  // layout hint rather than a raster size.
  size: { type: Number, default: 168 },
  // Accessible name; only the caller knows whose key this is.
  label: { type: String, default: "" },
});

const symbol = computed(() => buildQrSymbol(props.value));
</script>

<template>
  <svg
    v-if="symbol"
    :viewBox="`${-QR_QUIET_ZONE} ${-QR_QUIET_ZONE} ${symbol.extent} ${symbol.extent}`"
    :width="size"
    :height="size"
    role="img"
    :aria-label="label"
    shape-rendering="crispEdges"
    class="block"
  >
    <rect
      :x="-QR_QUIET_ZONE"
      :y="-QR_QUIET_ZONE"
      :width="symbol.extent"
      :height="symbol.extent"
      fill="#ffffff"
    />
    <path :d="symbol.path" fill="#000000" />
  </svg>
</template>
