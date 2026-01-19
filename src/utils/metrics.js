export function formatBytes(bytes) {
  const value = Number(bytes) || 0
  if (value === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(value) / Math.log(k))
  return `${(value / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

export function formatCategory(category) {
  if (!category) return 'Unknown'
  const cleaned = String(category).trim().replace(/_/g, ' ').replace(/\s+/g, ' ')
  return cleaned
    .split(/\s|-/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

export function formatDuration(ms) {
  const value = Number(ms)
  if (!Number.isFinite(value)) return 'N/A'
  if (value <= 0) return 'Expired'

  const totalMinutes = Math.floor(value / 60000)
  const days = Math.floor(totalMinutes / (60 * 24))
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60)
  const minutes = totalMinutes % 60

  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

export function formatMinutesAsDuration(minutes) {
  const value = Number(minutes)
  if (!Number.isFinite(value)) return 'N/A'
  return formatDuration(value * 60000)
}

export function pickImageDisplayName(img) {
  const tags = Array.isArray(img?.tags)
    ? img.tags
    : (Array.isArray(img?.RepoTags) ? img.RepoTags : (Array.isArray(img?.repoTags) ? img.repoTags : []))

  const bestTag = tags.find((t) => typeof t === 'string' && t.trim() && t !== '<none>:<none>')
  if (bestTag) return bestTag

  const shortId = img?.shortId || (typeof img?.id === 'string' ? img.id.replace(/^sha256:/, '').slice(0, 12) : null)
  return shortId || img?.id || img?.Id || 'Unknown'
}

// Counts UNIQUE apps per category (an app can belong to multiple categories).
export function computeCategoryStats(containers) {
  const list = Array.isArray(containers) ? containers : []
  const appsById = new Map()

  list.forEach((container) => {
    const app = container?.app
    if (!app) return
    const id = app.id || app.projectId || app.name || container.id
    if (!appsById.has(id)) appsById.set(id, app)
  })

  const categoryToApps = new Map() // categoryKey -> { label: string, apps: Set<string> }

  for (const [appId, app] of appsById.entries()) {
    const raw = (app?.category || 'uncategorized').toString()
    const parts = raw
      .split(',')
      .map((c) => c.trim())
      .filter(Boolean)

    const categories = parts.length > 0 ? parts : ['uncategorized']

    for (const category of categories) {
      const key = category.trim().toLowerCase()
      if (!categoryToApps.has(key)) categoryToApps.set(key, { label: category, apps: new Set() })
      categoryToApps.get(key).apps.add(appId)
    }
  }

  const entries = Array.from(categoryToApps.values()).map((v) => [v.label, v.apps.size])
  if (entries.length === 0) {
    return { mostUsed: null, leastUsed: null, total: 0, all: [], appsCount: appsById.size }
  }

  const sorted = entries.sort((a, b) => (Number(b?.[1]) || 0) - (Number(a?.[1]) || 0))
  const most = sorted[0]
  const least = sorted[sorted.length - 1]

  return {
    mostUsed: most ? { category: most[0], count: most[1] } : null,
    leastUsed: least ? { category: least[0], count: least[1] } : null,
    total: sorted.length,
    all: sorted,
    appsCount: appsById.size
  }
}
