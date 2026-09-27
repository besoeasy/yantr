#!/usr/bin/env bash
# Rebuild and run Yantr locally.
set -euo pipefail

SOCKET="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock"

podman rm -f yantr 2>/dev/null || true
podman rmi -f yantr:dev 2>/dev/null || true

podman build -t yantr:dev .

podman run --rm \
  --name yantr \
  --replace \
  --network host \
  --security-opt label=disable \
  -e CONTAINER_HOST=unix:///run/podman/podman.sock \
  -v "$SOCKET:/run/podman/podman.sock" \
  yantr:dev
