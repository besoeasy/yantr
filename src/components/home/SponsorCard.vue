<script setup>
import { computed, ref, onMounted, onUnmounted } from "vue";
import { useI18n } from "vue-i18n";
import { MessageCircle, Zap, GitPullRequest, ShieldCheck, ArrowUpRight } from "lucide-vue-next";

const { t } = useI18n();

// Eye tracking
const leftEyeRef = ref(null);
const rightEyeRef = ref(null);
const leftPupil = ref({ x: 0, y: 0 });
const rightPupil = ref({ x: 0, y: 0 });

function calcOffset(el, mx, my) {
  const rect = el.getBoundingClientRect();
  const cx = rect.left + rect.width / 2;
  const cy = rect.top + rect.height / 2;
  const dx = mx - cx;
  const dy = my - cy;
  const angle = Math.atan2(dy, dx);
  const dist = Math.min(4, Math.hypot(dx, dy) * 0.1);
  return { x: Math.cos(angle) * dist, y: Math.sin(angle) * dist };
}

function onMouseMove(e) {
  if (leftEyeRef.value) leftPupil.value = calcOffset(leftEyeRef.value, e.clientX, e.clientY);
  if (rightEyeRef.value) rightPupil.value = calcOffset(rightEyeRef.value, e.clientX, e.clientY);
}

// Perk cycling
const currentIndex = ref(0);
let perkInterval = null;

const benefits = computed(() => [
  { icon: MessageCircle, title: t("sponsorCard.benefits.devAccess.title"), desc: t("sponsorCard.benefits.devAccess.desc") },
  { icon: Zap, title: t("sponsorCard.benefits.roadmap.title"), desc: t("sponsorCard.benefits.roadmap.desc") },
  { icon: GitPullRequest, title: t("sponsorCard.benefits.earlyBuilds.title"), desc: t("sponsorCard.benefits.earlyBuilds.desc") },
  { icon: ShieldCheck, title: t("sponsorCard.benefits.badge.title"), desc: t("sponsorCard.benefits.badge.desc") },
]);

const activeBenefit = computed(() => benefits.value[currentIndex.value]);

onMounted(() => {
  document.addEventListener("mousemove", onMouseMove);
  perkInterval = setInterval(() => {
    currentIndex.value = (currentIndex.value + 1) % benefits.value.length;
  }, 2800);
});

onUnmounted(() => {
  document.removeEventListener("mousemove", onMouseMove);
  clearInterval(perkInterval);
});
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-amber-400 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 flex flex-col h-full p-5">

      <!-- Mascot -->
      <div class="flex flex-col items-center mb-4">
        <svg viewBox="0 0 60 64" class="w-28 h-28 mascot-float" xmlns="http://www.w3.org/2000/svg">
          <!-- Antenna -->
          <line x1="30" y1="14" x2="30" y2="6" stroke="#f59e0b" stroke-width="2.2" stroke-linecap="round"/>
          <circle cx="30" cy="4" r="3.5" fill="#fcd34d"/>
          <circle cx="30" cy="4" r="1.8" fill="#f59e0b"/>
          <!-- Body -->
          <ellipse cx="30" cy="39" rx="22" ry="21" fill="#f59e0b"/>
          <!-- Ear bumps -->
          <circle cx="9" cy="33" r="6.5" fill="#f59e0b"/>
          <circle cx="51" cy="33" r="6.5" fill="#f59e0b"/>
          <circle cx="9" cy="33" r="3.5" fill="#fcd34d"/>
          <circle cx="51" cy="33" r="3.5" fill="#fcd34d"/>
          <!-- Eye sockets -->
          <circle ref="leftEyeRef" cx="21" cy="36" r="9" fill="white"/>
          <circle ref="rightEyeRef" cx="39" cy="36" r="9" fill="white"/>
          <!-- Pupils -->
          <circle cx="21" cy="36" r="4.5" fill="#111827" :transform="`translate(${leftPupil.x}, ${leftPupil.y})`"/>
          <circle cx="39" cy="36" r="4.5" fill="#111827" :transform="`translate(${rightPupil.x}, ${rightPupil.y})`"/>
          <!-- Eye shine -->
          <circle cx="24" cy="33" r="2" fill="white" opacity="0.9" :transform="`translate(${leftPupil.x * 0.5}, ${leftPupil.y * 0.5})`"/>
          <circle cx="42" cy="33" r="2" fill="white" opacity="0.9" :transform="`translate(${rightPupil.x * 0.5}, ${rightPupil.y * 0.5})`"/>
          <!-- Blush -->
          <circle cx="11" cy="46" r="5.5" fill="#fb923c" opacity="0.35"/>
          <circle cx="49" cy="46" r="5.5" fill="#fb923c" opacity="0.35"/>
          <!-- Smile -->
          <path d="M 22 47 Q 30 54 38 47" stroke="#92400e" stroke-width="2" fill="none" stroke-linecap="round"/>
        </svg>

        <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors mt-1">{{ t('sponsorCard.title') }}</h3>
        <div class="text-[10px] font-bold text-gray-400 dark:text-zinc-500 uppercase tracking-[0.2em] mt-0.5">{{ t('sponsorCard.label') }}</div>
      </div>

      <!-- Cycling perk -->
      <div class="flex-1 flex flex-col justify-center">
        <transition name="perk" mode="out-in">
          <div
            :key="currentIndex"
            class="flex items-start gap-3 p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50"
          >
            <div class="w-7 h-7 rounded-md bg-amber-50 dark:bg-amber-500/10 border border-amber-100 dark:border-amber-500/20 flex items-center justify-center shrink-0">
              <component :is="activeBenefit.icon" class="w-3.5 h-3.5 text-amber-500" />
            </div>
            <div class="flex flex-col min-w-0">
              <span class="text-xs font-semibold text-gray-900 dark:text-white">{{ activeBenefit.title }}</span>
              <span class="text-[10px] text-gray-500 dark:text-zinc-400 leading-snug mt-0.5">{{ activeBenefit.desc }}</span>
            </div>
          </div>
        </transition>

        <!-- Dot indicators -->
        <div class="flex items-center justify-center gap-1.5 mt-3">
          <button
            v-for="(_, i) in benefits"
            :key="i"
            @click="currentIndex = i"
            class="transition-all duration-300 rounded-full"
            :class="i === currentIndex
              ? 'w-4 h-1.5 bg-amber-400'
              : 'w-1.5 h-1.5 bg-gray-200 dark:bg-zinc-700 hover:bg-amber-300 dark:hover:bg-amber-600'"
          />
        </div>
      </div>

      <!-- CTA -->
      <div class="mt-4 pt-3 border-t border-gray-100 dark:border-zinc-800/80">
        <a
          href="https://sponsor.besoeasy.com/"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta flex items-center justify-between w-full px-4 py-3 rounded-lg bg-gray-950 dark:bg-white text-white dark:text-gray-950 transition-all duration-300 hover:bg-gray-800 dark:hover:bg-gray-100 active:scale-[0.98]"
        >
          <span class="text-xs font-bold tracking-tight">{{ t('sponsorCard.cta') }}</span>
          <ArrowUpRight class="w-4 h-4 opacity-50 transition-all duration-300 group-hover/cta:opacity-100 group-hover/cta:translate-x-0.5 group-hover/cta:-translate-y-0.5" />
        </a>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mascot-float {
  animation: mascotFloat 3.5s ease-in-out infinite;
}

@keyframes mascotFloat {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-6px); }
}

.perk-enter-active {
  transition: opacity 0.35s ease, transform 0.35s ease;
}
.perk-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.perk-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.perk-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
