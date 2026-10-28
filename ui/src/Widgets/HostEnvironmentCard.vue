<script>
// Dashboard engine: one column — temperature as the live headline, fan speed
// and paired sensor rows beneath it. Uptime used to ride along as a row; it
// was dropped because the Host System spec sheet already renders it, and a
// barely-moving number does not need two homes. Widening the card would only
// stretch strings. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Thermometer, Fan } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metrics,
  metricsStale,
  isDegraded,
  tempText,
  fanText,
  sensors,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// Only true pairs get a row: a chip where fan i and temp i are the same slot
// is the one case the aggregate rows cannot express — the aggregate knows
// "52°C somewhere and 3000 RPM somewhere", the pair says which is which. A
// laptop whose fan chips carry no temperature shows no rows here and falls
// back on the fans row above, which is exactly what that hardware can claim.
const pairedSensors = computed(() =>
  sensors.value
    .filter((s) => s.fanRPM !== null && s.tempC !== null)
    .slice(0, 2)
);

// Heat is judged against the chip's own trip when the board ships one (85%
// of the threshold is where the margin starts feeling thin); the fixed 75°C
// stays as the stand-alone baseline for chips that ship no threshold, like
// k10temp does on many boards. The exact number matters less than a hot slot
// looking hot.
const isHot = (celsius, tripC) => (tripC ? celsius >= tripC * 0.85 : celsius >= 75);

const fmtTemp = (celsius) => `${Math.round(celsius)}°C`;

// The headline value reads the aggregate TempC, which carries no trip of its
// own; the strictest threshold it races is the hottest chip trip the snapshot
// saw. A null threshold falls back to the fixed 75°C baseline.
const headlineHot = computed(() => {
  const celsius = metrics.value?.tempC;
  if (!Number.isFinite(celsius)) return false;
  let trip = null;
  for (const s of sensors.value) {
    if (Number.isFinite(s?.tripC) && (trip === null || s.tripC > trip)) trip = s.tripC;
  }
  return isHot(celsius, trip);
});

// Temperature is the card's headline: it is the one live number here and the
// usual reason to look on a homelab box. The value came out of
// readTemperature filtered through the plausibility window, so a dead sensor
// renders the unavailable state instead of a convincing zero.
const headline = computed(() => (isDegraded("temperature") ? unavailable : (tempText.value ?? unavailable)));
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-indigo-200 bg-indigo-50 transition-transform duration-200 group-hover:scale-105 dark:border-indigo-900/50 dark:bg-indigo-900/20">
          <Thermometer class="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.environment") }}
          </h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">{{ t("quickMetrics.hostMetrics.live") }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="relative z-10 mt-auto flex flex-col pt-6 transition-opacity duration-200" :class="metricsStale ? 'opacity-50' : ''">
      <!-- Temperature headline: a live reading first, identity second. The
           value turns rose once some chip's threshold margin thins. -->
      <div class="mb-5 flex items-baseline justify-between gap-3">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <Thermometer class="h-3 w-3 shrink-0 text-indigo-600" />
          {{ t("quickMetrics.hostMetrics.temperature") }}
        </span>
        <span
          class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl"
          :class="headlineHot ? 'text-rose-600 dark:text-rose-400' : ''"
        >
          {{ headline }}
        </span>
      </div>

      <!-- Fans: the physical response to the temperature above. A card that
           shows heat but not what answers it stops half the story. -->
      <div
        class="rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50"
        :class="pairedSensors.length ? 'mb-2' : ''"
      >
        <div class="flex items-center justify-between gap-3">
          <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            <Fan class="h-3 w-3 shrink-0 text-sky-500" />
            {{ t("quickMetrics.hostMetrics.fans") }}
          </span>
          <span class="text-sm font-black tabular-nums tracking-tight yantra-text-primary">
            {{ isDegraded("fans") ? unavailable : (fanText ?? "—") }}
          </span>
        </div>
      </div>

      <!-- Paired slots: chip name left, its own temperature and fan right.
           The hot tint rides on the temperature against the chip's own trip
           when it ships one, so a burning NVMe reads hot no matter how cool
           the CPU headline sits. The title exposes the threshold a hover
           answers honestly: what number is this 52°C racing? -->
      <div
        v-for="sensor in pairedSensors"
        :key="sensor.chip + (sensor.label ?? '')"
        class="mb-2 flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 last:mb-0 dark:border-zinc-800 dark:bg-zinc-900/50"
        :title="sensor.tripC ? t('quickMetrics.hostMetrics.tempTripsAt', { trip: Math.round(sensor.tripC) }) : undefined"
      >
        <span class="flex min-w-0 items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <Fan class="h-3 w-3 shrink-0 text-sky-500" />
          <span class="truncate">{{ sensor.label || sensor.chip }}</span>
        </span>
        <span class="shrink-0 text-sm font-black tabular-nums tracking-tight" :class="isHot(sensor.tempC, sensor.tripC) ? 'text-rose-600 dark:text-rose-400' : 'yantra-text-primary'">
          {{ fmtTemp(sensor.tempC) }} · {{ Math.round(sensor.fanRPM) }} RPM
        </span>
      </div>
    </div>
  </div>
</template>
