<script setup>
import { Box, ClipboardList, HardDrive, Home, Layers, Check, ChevronRight, Heart, Github, Send, Bug } from "lucide-vue-next";

const navItems = [
  {
    title: "Images",
    to: "/images",
    description: "Inspect local Docker images",
    icon: Layers,
    accent: {
      ring: "focus:ring-emerald-500",
      glow: "bg-emerald-200",
      iconBg: "bg-emerald-900/30",
      iconHover: "group-hover:bg-emerald-800/40",
      textHover: "group-hover:text-emerald-400",
      check: "text-emerald-400",
      arrow: "text-emerald-400",
    },
    features: ["List images", "Cleanup helpers"],
  },
  {
    title: "Volumes",
    to: "/volumes",
    description: "View and manage named volumes",
    icon: HardDrive,
    accent: {
      ring: "focus:ring-amber-500",
      glow: "bg-amber-200",
      iconBg: "bg-amber-900/30",
      iconHover: "group-hover:bg-amber-800/40",
      textHover: "group-hover:text-amber-400",
      check: "text-amber-400",
      arrow: "text-amber-400",
    },
    features: ["Safe storage", "Portable data"],
  },
  {
    title: "Logs",
    to: "/logs",
    description: "Read container logs and recent output",
    icon: ClipboardList,
    accent: {
      ring: "focus:ring-rose-500",
      glow: "bg-rose-200",
      iconBg: "bg-rose-900/30",
      iconHover: "group-hover:bg-rose-800/40",
      textHover: "group-hover:text-rose-400",
      check: "text-rose-400",
      arrow: "text-rose-400",
    },
    features: ["Tail logs", "Debug quickly"],
  },
];

const externalItems = [
  {
    title: "Sponsor",
    href: "https://sponsor.besoeasy.com/",
    description: "Support Yantra development",
    icon: Heart,
    accent: {
      ring: "focus:ring-red-500",
      glow: "bg-red-200",
      iconBg: "bg-red-900/30",
      iconHover: "group-hover:bg-red-800/40",
      textHover: "group-hover:text-red-400",
      check: "text-red-400",
      arrow: "text-red-400",
    },
    features: ["Keep the project alive", "Fund new features"],
  },
  {
    title: "GitHub",
    href: "https://github.com/besoeasy/Yantra",
    description: "Source code, issues, and releases",
    icon: Github,
    accent: {
      ring: "focus:ring-slate-400",
      glow: "bg-slate-200",
      iconBg: "bg-slate-900/30",
      iconHover: "group-hover:bg-slate-800/40",
      textHover: "group-hover:text-slate-200",
      check: "text-slate-300",
      arrow: "text-slate-200",
    },
    features: ["Track updates", "Report bugs"],
  },
  {
    title: "Telegram",
    href: "https://t.me/+Qu06yCZHBAU3NTk1",
    description: "Join the community chat",
    icon: Send,
    accent: {
      ring: "focus:ring-sky-500",
      glow: "bg-sky-200",
      iconBg: "bg-sky-900/30",
      iconHover: "group-hover:bg-sky-800/40",
      textHover: "group-hover:text-sky-400",
      check: "text-sky-400",
      arrow: "text-sky-400",
    },
    features: ["Get help", "Share feedback"],
  },
  {
    title: "Report Issue",
    href: "https://github.com/besoeasy/yantra/issues",
    description: "Open a bug report or request a feature",
    icon: Bug,
    accent: {
      ring: "focus:ring-orange-500",
      glow: "bg-orange-200",
      iconBg: "bg-orange-900/30",
      iconHover: "group-hover:bg-orange-800/40",
      textHover: "group-hover:text-orange-400",
      check: "text-orange-400",
      arrow: "text-orange-400",
    },
    features: ["Bug reports", "Feature requests"],
  },
];
</script>

<template>
  <main class="flex flex-col items-center justify-center min-h-screen bg-slate-50 text-slate-900 dark:bg-black dark:text-white px-4 py-12">
    <div class="w-full mx-auto animate-fade-in-up">
      <!-- Navigation Grid -->
      <section aria-label="Navigation" class="grid grid-cols-1 md:grid-cols-3 gap-6 md:gap-8">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="group relative overflow-hidden bg-white dark:bg-gray-900 rounded-3xl p-8 border border-slate-200 dark:border-gray-700 hover:shadow-2xl hover:-translate-y-2 transition-all duration-300 focus:outline-none focus:ring-4"
          :class="item.accent.ring"
          tabindex="0"
          :aria-label="`${item.title}: ${item.description}`"
        >
          <!-- Gradient Glow -->
          <div
            class="absolute top-0 right-0 -mt-8 -mr-8 w-40 h-40 rounded-full blur-3xl opacity-0 group-hover:opacity-70 transition-opacity duration-500"
            :class="item.accent.glow"
          ></div>

          <div class="relative z-10">
            <!-- Icon -->
            <div
              class="flex items-center justify-center w-20 h-20 rounded-2xl mb-6 transition-all duration-300 group-hover:scale-110"
              :class="[item.accent.iconBg, item.accent.iconHover]"
            >
              <component :is="item.icon" :size="40" class="text-slate-900 dark:text-white" />
            </div>

            <!-- Content -->
            <h2 class="text-2xl font-bold text-slate-900 dark:text-white mb-2 transition-colors" :class="item.accent.textHover">
              {{ item.title }}
            </h2>
            <p class="text-slate-600 dark:text-gray-300 leading-relaxed mb-6">
              {{ item.description }}
            </p>

            <!-- Features -->
            <ul class="space-y-2 text-sm text-slate-500 dark:text-gray-400">
              <li v-for="feature in item.features" :key="feature" class="flex items-center gap-2">
                <Check :size="16" :class="item.accent.check" />
                {{ feature }}
              </li>
            </ul>

            <!-- Arrow -->
            <div class="mt-6 flex items-center font-semibold text-sm group-hover:translate-x-2 transition-transform" :class="item.accent.arrow">
              Open
              <ChevronRight :size="16" class="ml-1" />
            </div>
          </div>
        </router-link>

        <a
          v-for="item in externalItems"
          :key="item.href"
          :href="item.href"
          target="_blank"
          rel="noopener noreferrer"
          class="group relative overflow-hidden bg-white dark:bg-gray-900 rounded-3xl p-8 border border-slate-200 dark:border-gray-700 hover:shadow-2xl hover:-translate-y-2 transition-all duration-300 focus:outline-none focus:ring-4"
          :class="item.accent.ring"
          tabindex="0"
          :aria-label="`${item.title}: ${item.description}`"
        >
          <!-- Gradient Glow -->
          <div
            class="absolute top-0 right-0 -mt-8 -mr-8 w-40 h-40 rounded-full blur-3xl opacity-0 group-hover:opacity-70 transition-opacity duration-500"
            :class="item.accent.glow"
          ></div>

          <div class="relative z-10">
            <!-- Icon -->
            <div
              class="flex items-center justify-center w-20 h-20 rounded-2xl mb-6 transition-all duration-300 group-hover:scale-110"
              :class="[item.accent.iconBg, item.accent.iconHover]"
            >
              <component :is="item.icon" :size="40" class="text-slate-900 dark:text-white" />
            </div>

            <!-- Content -->
            <h2 class="text-2xl font-bold text-slate-900 dark:text-white mb-2 transition-colors" :class="item.accent.textHover">
              {{ item.title }}
            </h2>
            <p class="text-slate-600 dark:text-gray-300 leading-relaxed mb-6">
              {{ item.description }}
            </p>

            <!-- Features -->
            <ul class="space-y-2 text-sm text-slate-500 dark:text-gray-400">
              <li v-for="feature in item.features" :key="feature" class="flex items-center gap-2">
                <Check :size="16" :class="item.accent.check" />
                {{ feature }}
              </li>
            </ul>

            <!-- Arrow -->
            <div class="mt-6 flex items-center font-semibold text-sm group-hover:translate-x-2 transition-transform" :class="item.accent.arrow">
              Visit
              <ChevronRight :size="16" class="ml-1" />
            </div>
          </div>
        </a>
      </section>

      <!-- Back Button -->
      <div class="text-center mt-10">
        <router-link
          to="/home"
          class="inline-flex items-center text-slate-500 hover:text-purple-600 dark:text-gray-400 dark:hover:text-purple-400 transition-colors font-semibold"
        >
          <ChevronRight :size="18" class="mr-2 rotate-180" />
          Back to Home
        </router-link>
      </div>
    </div>
  </main>
</template>

<style scoped>
.animate-fade-in-up {
  animation: fadeInUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
