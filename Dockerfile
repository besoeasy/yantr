# ─── Stage 1: Build Vue.js Frontend ──────────────────────────────────────────
FROM docker.io/library/node:lts AS frontend-builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --prefer-offline --no-audit || npm install

COPY . .
RUN VITE_BUILD_TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ) npm run build
RUN rm -rf node_modules .npm

# ─── Stage 2: Build Go Core Backend ──────────────────────────────────────────
FROM docker.io/library/golang:1.25-alpine AS backend-builder

WORKDIR /build

# Copy Go module files and download deps first (better layer caching)
COPY core/go.mod core/go.sum ./
RUN go mod download

# Copy source and build static binary
COPY core/ ./
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(date -u +%Y%m%d)" \
    -o /yantr .

# ─── Stage 3: Final Image ────────────────────────────────────────────────────
FROM docker.io/library/alpine:latest

# Install runtime dependencies (no Node.js runtime needed!)
RUN apk add --no-cache podman podman-compose wget

# Builds triggered from inside this container run on the host via the
# mounted Podman socket, but the server defaults to /etc/containers/seccomp.json
# which is missing on Debian/Ubuntu hosts (they only ship
# /usr/share/containers/seccomp.json). Direct host builds fall back correctly,
# but remote API builds fail with "opening seccomp profile failed". Wrap the
# podman CLI so every `podman build` from inside this container pins an explicit
# profile that exists on both client and host.
RUN mv /usr/bin/podman /usr/bin/podman.real && cat > /usr/bin/podman <<'WRAPPER_EOF'
#!/usr/bin/env python3
"""Pin explicit seccomp profile on `podman build` for remote builds."""
import os
import sys

REAL = "/usr/bin/podman.real"
PROFILE = "/usr/share/containers/seccomp.json"

args = sys.argv[1:]
has_build = "build" in args
has_seccomp = any("seccomp" in a for a in args)
if has_build and not has_seccomp and os.path.isfile(PROFILE):
    out = []
    inserted = False
    for a in args:
        out.append(a)
        if not inserted and a == "build":
            out.append("--security-opt")
            out.append("seccomp=" + PROFILE)
            inserted = True
    args = out
os.execv(REAL, [REAL] + args)
WRAPPER_EOF
RUN chmod +x /usr/bin/podman

# Configure Podman inside the container as a pure remote client using the mounted host socket
RUN mkdir -p /etc/containers && printf '[engine]\nremote = true\nactive_service = "host"\n\n[engine.service_destinations.host]\nuri = "unix:///run/podman/podman.sock"\n' > /etc/containers/containers.conf

WORKDIR /app

RUN mkdir -p /data

# Copy compiled assets
COPY --from=frontend-builder /app/dist ./dist
COPY --from=backend-builder /yantr ./yantr
COPY apps/ ./apps/

EXPOSE 5252

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD wget -qO- http://127.0.0.1:5252/api/health >/dev/null 2>&1 || exit 1

ENV YANTR_SERVE_UI=true
ENV NODE_ENV=production
ENV CONTAINER_HOST=unix:///run/podman/podman.sock

ENTRYPOINT ["/app/yantr"]
