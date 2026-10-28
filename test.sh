#!/usr/bin/env bash
# Rebuild and run Yantr locally.
#
# Mirrors what install.sh generates, so a dev run exercises the same
# socket contract as production: the host path is passed in explicitly via
# HOST_PODMAN_SOCKET instead of being rediscovered at runtime. The two :ro
# binds are the host metrics mounts, same as the generated Quadlet.
set -euo pipefail

RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
export XDG_RUNTIME_DIR="$RUNTIME_DIR"
export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=$RUNTIME_DIR/bus}"

HOST_SOCK="$RUNTIME_DIR/podman/podman.sock"

# Ensure the rootless Podman user socket exists — mirrors install.sh. Without it
# the bind mount below fails with "statfs .../podman.sock: no such file".
if [ ! -S "$HOST_SOCK" ]; then
  if command -v systemctl >/dev/null 2>&1; then
    systemctl --user --no-ask-password enable --now podman.socket >/dev/null 2>&1 || true
  fi
fi
if [ ! -S "$HOST_SOCK" ]; then
  echo "Error: Podman socket not found at $HOST_SOCK" >&2
  echo "       Enable it with: systemctl --user enable --now podman.socket" >&2
  exit 1
fi

podman rm -f yantr 2>/dev/null || true
podman rmi -f yantr:dev 2>/dev/null || true

podman build -t yantr:dev .

podman run --rm \
  --name yantr \
  --replace \
  --network host \
  --security-opt label=disable \
  -e CONTAINER_HOST=unix:///run/podman/podman.sock \
  -e HOST_PODMAN_SOCKET="$HOST_SOCK" \
  -v "$HOST_SOCK:/run/podman/podman.sock" \
  -v /:/host:ro \
  -v /sys:/sys:ro \
  yantr:dev