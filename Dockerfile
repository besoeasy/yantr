# Yantr builds without emulation: every step that executes binaries runs on the
# builder's own platform (--platform=$BUILDPLATFORM), the target architecture's
# runtime packages are extracted by apk into a staging rootfs, and the final
# stage starts from scratch. `podman build --platform linux/amd64,linux/arm64`
# and `docker buildx build --platform ...` both work unchanged; the only
# requirement is TARGETARCH/TARGETOS support in the builder (podman >= 5).

# ─── Stage 1: Build Vue.js Frontend ──────────────────────────────────────────
# The UI bundles to static files which are architecture-independent, so this
# always builds on the builder's platform.
FROM --platform=$BUILDPLATFORM docker.io/library/node:lts AS frontend-builder

WORKDIR /app/ui

COPY ui/package.json ui/package-lock.json* ./
RUN npm ci --prefer-offline --no-audit || npm install

COPY ui/ ./
RUN VITE_BUILD_TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ) npm run build

# ─── Stage 2: Build Go Core Backend ──────────────────────────────────────────
# Go cross-compiles natively (CGO_ENABLED=0), no emulation needed.
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.25-alpine AS backend-builder

WORKDIR /build

# Copy Go module files and download deps first (better layer caching)
COPY core/go.mod core/go.sum ./
RUN go mod download

# Copy source and build static binary for the target architecture
COPY core/ ./
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build \
    -ldflags="-w -s -X main.version=$(date -u +%Y%m%d)" \
    -o /yantr .

# ─── Stage 3: fetch the target arch's apk keyring ────────────────────────────
# Each architecture's base image signs its APKINDEX with its own keyring; COPY
# the files out — copying never executes foreign code.
FROM docker.io/library/alpine:latest AS target-keys

# ─── Stage 4: Runtime root filesystem for the target architecture ────────────
# apk extracts podman/podman-compose/wget for the target architecture into a
# staging rootfs on the builder platform; nothing foreign is ever run here.
FROM --platform=$BUILDPLATFORM docker.io/library/alpine:latest AS runtime-rootfs

ARG TARGETARCH
COPY --from=target-keys /etc/apk/keys/ /target-apk-keys/

# Map the Docker/Podman arch name (GOARCH style) to Alpine's arch name,
# passing through names that already match.
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) APK_ARCH="x86_64" ;; \
      arm64) APK_ARCH="aarch64" ;; \
      arm)   APK_ARCH="armv7" ;; \
      386)   APK_ARCH="x86" ;; \
      *)     APK_ARCH="${TARGETARCH}" ;; \
    esac; \
    mkdir -p /rootfs; \
    apk add \
      --root /rootfs \
      --arch "${APK_ARCH}" \
      --keys-dir /target-apk-keys \
      --repositories-file /etc/apk/repositories \
      --initdb \
      --no-cache \
      --no-progress \
      --no-scripts \
      alpine-base podman podman-compose wget; \
    # Podman inside the container is a pure remote client driving the socket
    # mounted from the host (same config the former RUN wrote, without exec).
    mkdir -p /rootfs/etc/containers; \
    printf '[engine]\nremote = true\nactive_service = "host"\n\n[engine.service_destinations.host]\nuri = "unix:///run/podman/podman.sock"\n' \
      > /rootfs/etc/containers/containers.conf; \
    mkdir -p /rootfs/data /rootfs/app; \
    rm -rf /rootfs/var/cache/apk/* /rootfs/var/log/* /rootfs/tmp/*

# ─── Stage 5: Final Image ────────────────────────────────────────────────────
# `scratch` has no platform of its own, so the image architecture comes purely
# from `--platform`; the pre-built rootfs is just merged in.
FROM scratch AS final

COPY --from=runtime-rootfs /rootfs/ /

COPY --from=frontend-builder /app/ui/dist /app/dist
COPY --from=backend-builder /yantr /app/yantr
COPY apps/ /app/apps/

WORKDIR /app

EXPOSE 5252

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD wget -qO- http://127.0.0.1:5252/api/health >/dev/null 2>&1 || exit 1

ENV YANTR_SERVE_UI=true
ENV NODE_ENV=production
ENV CONTAINER_HOST=unix:///run/podman/podman.sock

ENTRYPOINT ["/app/yantr"]
