import { computed } from 'vue'

export function usePortConflict(containers) {
  function checkPortConflict(hostPort, protocol) {
    return containers.value.find((c) =>
      c.Ports?.some((p) => p.PublicPort === parseInt(hostPort) && p.Type === protocol)
    )
  }

  function getPortStatus(port, customPortMappings) {
    const hostPort = customPortMappings[port.hostPort + "/" + port.protocol] || port.hostPort
    const conflict = checkPortConflict(hostPort, port.protocol)

    if (conflict) {
      return {
        status: "conflict",
        color: "red",
        message: `Conflict: ${conflict.Names?.[0]?.replace(/^\//, "") || "Container"}`,
      }
    }

    if (parseInt(hostPort) < 1024) {
      return {
        status: "warning",
        color: "yellow",
        message: "Privileged (Root req.)",
      }
    }

    return {
      status: "available",
      color: "green",
      message: "Available",
    }
  }

  return {
    checkPortConflict,
    getPortStatus,
  }
}
