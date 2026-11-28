// Group Podman containers back under their Compose service.
//
// `/api/stacks/:id` returns one row per container because container-level
// diagnostics live on ContainerDetail. The stack page, however, should present
// the app's shape at the service level: `web`, `db`, `worker`, etc. Compose can
// scale a service to multiple containers; those containers are children of the
// service, not sibling services.

export function stackServiceKey(svc) {
  return svc?.composeService || svc?.service || svc?.name || svc?.id || 'unknown'
}

export function stackServiceState(containers) {
  const list = containers ?? []
  if (list.length === 0) return 'stopped'
  if (list.every((c) => c.state === 'running')) return 'running'
  if (list.some((c) => c.state === 'running')) return 'partial'
  return 'stopped'
}

export function groupStackServices(services = []) {
  const groups = new Map()

  for (const svc of services) {
    const key = stackServiceKey(svc)
    if (!groups.has(key)) {
      groups.set(key, {
        key,
        serviceName: key,
        containers: [],
        hasYantrLabel: false,
      })
    }
    const group = groups.get(key)
    group.containers.push(svc)
    group.hasYantrLabel = group.hasYantrLabel || svc.hasYantrLabel === true
  }

  return [...groups.values()]
    .map((group) => {
      const first = group.containers[0] ?? {}
      const images = [...new Set(group.containers.map((c) => c.image).filter(Boolean))]
      return {
        ...group,
        state: stackServiceState(group.containers),
        image: images[0] ?? '',
        images,
        id: group.containers.length === 1 ? first.id : null,
        containerNames: group.containers.map((c) => c.name || c.id).filter(Boolean),
      }
    })
    .sort((a, b) => a.serviceName.localeCompare(b.serviceName))
}
