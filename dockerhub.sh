#!/usr/bin/env bash
# Build and push the multi-arch Yantr image (linux/amd64 + linux/arm64) to Docker Hub.
#
# Publishes two tags of the same build: `latest` for the head of the tree and a
# date stamp (YYYY.MM.DD) so each release stays pullable by version too.
# Run `podman login docker.io` first if the registry is not already
# authenticated.
set -euo pipefail

IMAGE="docker.io/besoeasy/yantr"
MANIFEST="yantr-multiarch"
DATE_TAG="$(date -u +%Y.%m.%d)"

# The arm64 leg emulates through qemu, so binfmt must be registered first.
if ! podman run --rm --platform linux/arm64 docker.io/library/alpine true >/dev/null 2>&1; then
  echo "Error: linux/arm64 emulation (qemu binfmt) is not registered on this host." >&2
  echo "Enable it, then re-run:" >&2
  echo "  sudo docker run --privileged --rm tonistiigi/binfmt --install arm64" >&2
  exit 1
fi

echo "Building ${IMAGE}:latest and ${IMAGE}:${DATE_TAG} for linux/amd64 and linux/arm64"

# Start fresh in case a previous run left the manifest behind.
podman manifest rm "$MANIFEST" >/dev/null 2>&1 || true
podman manifest create "$MANIFEST"

podman build \
  --platform linux/amd64,linux/arm64 \
  --manifest "$MANIFEST" \
  .

echo "Platforms in manifest:"
podman manifest inspect "$MANIFEST" | grep -o '"architecture": "[^"]*"' | sort -u

echo "Pushing ${IMAGE}:latest"
podman manifest push "$MANIFEST" "${IMAGE}:latest"

echo "Pushing ${IMAGE}:${DATE_TAG}"
podman manifest push "$MANIFEST" "${IMAGE}:${DATE_TAG}"

podman manifest rm "$MANIFEST" >/dev/null

echo "Done: pushed ${IMAGE}:latest and ${IMAGE}:${DATE_TAG} (linux/amd64, linux/arm64)"
