# =========================
# Builder stage (Vue build)
# =========================
FROM docker.io/oven/bun:debian AS builder

WORKDIR /app

# Copy dependency files
COPY package.json bun.lock ./

# Install ALL deps (including dev)
RUN bun install

# Copy source
COPY . .

# Build Vue app
RUN bun run build

# =========================
# Production stage
# =========================
FROM docker.io/oven/bun:1.3.7-debian

# Install Docker CLI (needed by your app)
RUN apt-get update \
	&& apt-get install -y --no-install-recommends docker.io docker-compose \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy dependency files
COPY package.json bun.lock ./

# Install only production deps
RUN bun install --production

# Copy built frontend
COPY --from=builder /app/dist ./dist

# Copy backend / app files
COPY daemon/ ./daemon/
COPY apps/ ./apps/
# Start server
EXPOSE 5252

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD wget -qO- http://127.0.0.1:5252/api/health >/dev/null 2>&1 || exit 1

# Environment
ENV PORT=5252
ENV NODE_ENV=production

CMD ["bun", "run", "server"]