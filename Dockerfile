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
COPY apps/ ./apps/

RUN chmod +x ./daemon/entrypoint.sh ./daemon/updater.sh

# Expose port
EXPOSE 5252

# Environment
ENV PORT=5252
ENV NODE_ENV=production

# Start server + optional auto-updater
CMD ["./daemon/entrypoint.sh"]