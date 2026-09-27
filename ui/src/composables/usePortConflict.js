import { computed } from 'vue'

/**
 * usePortConflict — reports whether a requested host port is usable.
 *
 * Field-name note: this reads `container.ports` / `container.name`, which is the
 * shape `/api/containers` returns. It previously read `c.Ports` / `c.Names` —
 * the Podman/Docker SDK shape — so on every real response those were undefined,
 * `.some()` never ran, and *no port was ever reported as conflicting*. Every
 * field silently said "Available". Both spellings are accepted below so the
 * composable is correct against either shape.
 */
export function usePortConflict(containers) {
  function portBindingsOf(container) {
    return container?.ports ?? container?.Ports ?? []
  }

  function containerName(container) {
    const name = container?.name ?? container?.Names?.[0] ?? 'Container'
    return String(name).replace(/^\//, '')
  }

  function checkPortConflict(hostPort, protocol, excludeContainerId = null) {
    const wanted = parseInt(hostPort, 10)
    if (!wanted) return null

    return (
      containers.value.find((c) => {
        if (excludeContainerId && c.id === excludeContainerId) return false
        return portBindingsOf(c).some(
          (p) => p.PublicPort === wanted && (!protocol || p.Type === protocol)
        )
      }) ?? null
    )
  }

  /**
   * Finds another field of the same form already bound to the same host port.
   *
   * `mappings` is the host-port map keyed by "<containerPort>/<protocol>". This
   * field's own key is excluded by identity rather than by value — comparing
   * values would make every field skip a genuine duplicate, since the field
   * being checked and the duplicate hold the same value.
   *
   * Protocol is respected because a tcp and a udp binding may legitimately
   * share a host port.
   *
   * Previously only ports held by *other running containers* were considered, so
   * assigning one host port to two fields of a single deploy was reported as
   * "Available" twice and the deploy then failed at `podman compose up`.
   */
  function checkInFormDuplicate(mappings, currentKey, protocol, wantedPort) {
    if (!mappings || typeof mappings !== 'object') return null
    const wanted = parseInt(wantedPort, 10)
    if (!wanted) return null

    for (const [key, value] of Object.entries(mappings)) {
      if (key === currentKey) continue
      const keyProtocol = key.includes('/') ? key.slice(key.lastIndexOf('/') + 1) : 'tcp'
      if (protocol && keyProtocol !== protocol) continue
      if (parseInt(value, 10) === wanted) return key
    }
    return null
  }

  function getPortStatus(port, customPortMappings) {
    const mappings = customPortMappings || {}
    const currentKey = port.hostPort + '/' + port.protocol
    const hostPort = mappings[currentKey] || port.hostPort
    const conflict = checkPortConflict(hostPort, port.protocol)

    if (conflict) {
      return {
        status: 'conflict',
        color: 'red',
        message: `Conflict: ${containerName(conflict)}`,
      }
    }

    const duplicate = checkInFormDuplicate(mappings, currentKey, port.protocol, hostPort)
    if (duplicate) {
      return {
        status: 'conflict',
        color: 'red',
        message: `Conflict: ${duplicate} is already used in this form`,
      }
    }

    if (parseInt(hostPort, 10) < 1024) {
      return {
        status: 'warning',
        color: 'yellow',
        message: 'Privileged (Root req.)',
      }
    }

    return {
      status: 'available',
      color: 'green',
      message: 'Available',
    }
  }

  return {
    checkPortConflict,
    checkInFormDuplicate,
    getPortStatus,
  }
}
