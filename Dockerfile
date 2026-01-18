# =========================
# Builder stage (Vue build)
# =========================
FROM docker.io/oven/bun:latest AS builder

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
FROM docker.io/oven/bun:alpine

# Install Docker CLI (needed by your app)
RUN apk add --no-cache docker-cli docker-cli-compose

WORKDIR /app

# Copy dependency files
COPY package.json bun.lock ./

# Install only production deps
RUN bun install --production

# Copy built frontend
COPY --from=builder /app/dist ./dist

# Copy backend / app files
COPY daemon/ ./daemon/
COPY updator/ ./updator/
COPY apps/ ./apps/

RUN chmod +x ./updator/entrypoint.sh ./updator/updater.sh

# Expose port
EXPOSE 5252

# Healthcheck used by the self-updater to validate the new container started correctly
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD wget -qO- http://127.0.0.1:5252/api/health >/dev/null 2>&1 || exit 1

# Environment
ENV PORT=5252
ENV NODE_ENV=production

# Start server + optional auto-updater
CMD ["./updator/entrypoint.sh"]