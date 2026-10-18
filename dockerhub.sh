#!/usr/bin/env bash
# Build and push the Yantr image to Docker Hub.
#
# Publishes two tags of the same build: `latest` for the head of the tree and a
# date stamp (YYYY.MM.DD) so each release stays pullable by version too.
# Run `podman login docker.io` first if the registry is not already
# authenticated.
set -euo pipefail

IMAGE="docker.io/besoeasy/yantr"
DATE_TAG="$(date -u +%Y.%m.%d)"

echo "Building ${IMAGE}:latest and ${IMAGE}:${DATE_TAG}"
podman build \
  -t "${IMAGE}:latest" \
  -t "${IMAGE}:${DATE_TAG}" \
  .

echo "Pushing ${IMAGE}:latest"
podman push "${IMAGE}:latest"

echo "Pushing ${IMAGE}:${DATE_TAG}"
podman push "${IMAGE}:${DATE_TAG}"

echo "Done: pushed ${IMAGE}:latest and ${IMAGE}:${DATE_TAG}"
