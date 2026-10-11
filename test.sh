#!/usr/bin/env bash
# Rebuild and run Yantr locally.
#
# Mirrors what install.sh generates, so a dev run exercises the same
# socket contract as production: the host path is passed in explicitly via
# HOST_PODMAN_SOCKET instead of being rediscovered at runtime.
set -euo pipefail

RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
HOST_SOCK="$RUNTIME_DIR/podman/podman.sock"

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
  yantr:dev