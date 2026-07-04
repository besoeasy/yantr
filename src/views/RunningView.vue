<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import {
  Activity,
  Play,
  Square,
  RefreshCw,
  Loader,
  Box,
} from '@lucide/vue'
import { useAppStore } from '../stores/app'
import { useAuthStore } from '../stores/auth'

const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(true)
const actionLoading = ref({}) // { [containerId]: 'start' | 'stop' | 'restart' | null }
let refreshInterval = null

async function loadContainers() {
  await appStore.fetchContainers()
  loading.value = false
}

async function containerAction(containerId, action) {
  actionLoading.value = { ...actionLoading.value, [containerId]: action }
  try {
    await authStore.apiFetch(`/api/containers/${containerId}/${action}`, { method: 'POST' })
    await appStore.fetchContainers()
  } catch {
    // silently fail; UI will reflect state on next refresh
  } finally {
    actionLoading.value = { ...actionLoading.value, [containerId]: null }
  }
}

function statusColor(state) {
  const s = (state || '').toLowerCase()
  if (s === 'running') return 'running'
  if (s === 'paused') return 'paused'
  return 'exited'
}

function statusLabel(state) {
  const s = (state || '').toLowerCase()
  if (s === 'running') return 'running'
  if (s === 'paused') return 'paused'
  if (s === 'exited') return 'exited'
  return state || 'unknown'
}

function containerName(c) {
  const name = c.Names?.[0] || c.name || c.ID || c.id || 'unknown'
  return name.replace(/^\//, '')
}

function containerImage(c) {
  return c.Image || c.image || '—'
}

function containerState(c) {
  return c.State || c.state || c.status || ''
}

function isRunning(c) {
  return containerState(c).toLowerCase() === 'running'
}

const containers = computed(() => appStore.containers)

onMounted(() => {
  loadContainers()
  refreshInterval = setInterval(loadContainers, 10000)
})

onUnmounted(() => {
  clearInterval(refreshInterval)
})
</script>

<template>
  <div class="running-view">
    <div class="page-header">
      <div>
        <h1>Running Containers</h1>
        <p style="font-size: 0.85rem; margin-top: 4px">
          Auto-refreshes every 10 seconds
        </p>
      </div>
      <button class="btn btn-ghost" id="btn-refresh-running" @click="loadContainers">
        <RefreshCw :size="15" />
        Refresh
      </button>
    </div>

    <!-- Loading skeletons -->
    <div v-if="loading" class="loading-list">
      <div v-for="n in 4" :key="n" class="skeleton-row"></div>
    </div>

    <!-- Empty state -->
    <div v-else-if="containers.length === 0" class="empty-state">
      <Activity :size="44" />
      <h3>No containers found</h3>
      <p>Deploy something from the App Store to see it here.</p>
      <router-link to="/store" class="btn btn-primary" id="btn-goto-store">Browse App Store</router-link>
    </div>

    <!-- Container rows -->
    <div v-else class="containers-list">
      <div
        v-for="c in containers"
        :key="c.Id || c.ID || c.id"
        class="container-row card"
      >
        <!-- Left: status dot + info -->
        <div class="container-info">
          <div class="status-dot" :class="statusColor(containerState(c))"></div>
          <div class="container-meta">
            <h3 class="container-name">{{ containerName(c) }}</h3>
            <span class="container-image">{{ containerImage(c) }}</span>
          </div>
          <span class="status-badge" :class="`status-badge--${statusColor(containerState(c))}`">
            {{ statusLabel(containerState(c)) }}
          </span>
        </div>

        <!-- Right: actions -->
        <div class="container-actions">
          <!-- Start (only when not running) -->
          <button
            v-if="!isRunning(c)"
            class="btn btn-ghost action-btn action-btn--start"
            :disabled="!!actionLoading[c.Id || c.ID || c.id]"
            :id="`btn-start-${c.Id || c.ID || c.id}`"
            @click="containerAction(c.Id || c.ID || c.id, 'start')"
          >
            <Loader
              v-if="actionLoading[c.Id || c.ID || c.id] === 'start'"
              :size="14"
              class="spin"
            />
            <Play v-else :size="14" />
            Start
          </button>

          <!-- Stop (only when running) -->
          <button
            v-if="isRunning(c)"
            class="btn btn-ghost action-btn action-btn--stop"
            :disabled="!!actionLoading[c.Id || c.ID || c.id]"
            :id="`btn-stop-${c.Id || c.ID || c.id}`"
            @click="containerAction(c.Id || c.ID || c.id, 'stop')"
          >
            <Loader
              v-if="actionLoading[c.Id || c.ID || c.id] === 'stop'"
              :size="14"
              class="spin"
            />
            <Square v-else :size="14" />
            Stop
          </button>

          <!-- Restart -->
          <button
            class="btn btn-ghost action-btn"
            :disabled="!!actionLoading[c.Id || c.ID || c.id]"
            :id="`btn-restart-${c.Id || c.ID || c.id}`"
            @click="containerAction(c.Id || c.ID || c.id, 'restart')"
          >
            <Loader
              v-if="actionLoading[c.Id || c.ID || c.id] === 'restart'"
              :size="14"
              class="spin"
            />
            <RefreshCw v-else :size="14" />
            Restart
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.running-view {
  padding: 28px 32px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  overflow-y: auto;
  height: 100%;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.page-header p {
  color: var(--text-muted);
  margin: 0;
}

/* Container list */
.containers-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.container-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  gap: 16px;
  flex-wrap: wrap;
}

.container-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.container-meta {
  flex: 1;
  min-width: 0;
}

.container-name {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.container-image {
  font-size: 0.78rem;
  color: var(--text-muted);
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

/* Status dot */
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.running {
  background: var(--success);
  box-shadow: 0 0 8px var(--success);
  animation: pulse-green 2s ease-in-out infinite;
}
.status-dot.exited {
  background: var(--danger);
}
.status-dot.paused {
  background: var(--warning);
}

@keyframes pulse-green {
  0%, 100% { box-shadow: 0 0 4px var(--success); }
  50%       { box-shadow: 0 0 12px var(--success); }
}

/* Status badge */
.status-badge {
  font-size: 0.72rem;
  font-weight: 500;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  padding: 3px 10px;
  border-radius: 99px;
  flex-shrink: 0;
}
.status-badge--running {
  background: rgba(16, 185, 129, 0.1);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.25);
}
.status-badge--exited {
  background: rgba(239, 68, 68, 0.1);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.25);
}
.status-badge--paused {
  background: rgba(245, 158, 11, 0.1);
  color: #fbbf24;
  border: 1px solid rgba(245, 158, 11, 0.25);
}

/* Actions */
.container-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.action-btn {
  padding: 6px 12px;
  font-size: 0.82rem;
}

.action-btn--stop {
  color: var(--danger);
  border-color: rgba(239, 68, 68, 0.3);
}
.action-btn--stop:hover {
  background: rgba(239, 68, 68, 0.08);
  border-color: rgba(239, 68, 68, 0.5);
}

.action-btn--start {
  color: var(--success);
  border-color: rgba(16, 185, 129, 0.3);
}
.action-btn--start:hover {
  background: rgba(16, 185, 129, 0.08);
  border-color: rgba(16, 185, 129, 0.5);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Skeleton */
.loading-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.skeleton-row {
  height: 70px;
  border-radius: var(--radius-lg);
  background: linear-gradient(
    90deg,
    var(--bg-card) 25%,
    var(--bg-elevated) 50%,
    var(--bg-card) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Empty */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 80px 20px;
  color: var(--text-muted);
  text-align: center;
}
.empty-state h3 {
  color: var(--text-secondary);
}

/* Spinner */
.spin {
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
