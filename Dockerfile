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
RUN apk add --no-cache docker-cli docker-cli-compose wget dufs caddy

WORKDIR /app

RUN mkdir -p /data

# Copy compiled assets
COPY --from=frontend-builder /app/dist ./dist
COPY --from=backend-builder /yantr ./yantr
COPY apps/ ./apps/
COPY entrypoint.sh /entrypoint.sh

EXPOSE 5252

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD wget -qO- http://127.0.0.1:5252/api/health >/dev/null 2>&1 || exit 1

ENV YANTR_SERVE_UI=true
ENV NODE_ENV=production

# OFFICIAL=true is set by GitHub CI when building ghcr.io/besoeasy/yantr.
# Local builds leave it false — the self-update loop in entrypoint.sh is skipped.
ARG OFFICIAL=false
ENV YANTR_OFFICIAL=${OFFICIAL}

ENTRYPOINT ["/entrypoint.sh"]
