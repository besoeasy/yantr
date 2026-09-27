export function formatBytes(bytes) {
  const value = Number(bytes) || 0
  if (value === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(value) / Math.log(k))
  return `${(value / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

export function formatDuration(ms) {
  const value = Number(ms)
  if (!Number.isFinite(value)) return 'N/A'
  if (value <= 0) return 'Expired'

  const totalSeconds = Math.floor(value / 1000)
  const totalMinutes = Math.floor(totalSeconds / 60)
  const days = Math.floor(totalMinutes / (60 * 24))
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60)
  const minutes = totalMinutes % 60
  const seconds = totalSeconds % 60

  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

export function formatUptime(ms) {
  if (ms === null) return '—'
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ${m % 60}m`
  const d = Math.floor(h / 24)
  return `${d}d ${h % 24}h`
}

