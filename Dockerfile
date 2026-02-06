# =========================
# Builder stage (Vue build)
# =========================
FROM docker.io/library/node:slim AS builder

WORKDIR /app

# Copy dependency files
COPY package.json package-lock.json* ./

# Install ALL deps (including dev)
RUN npm ci --prefer-offline --no-audit || npm install

# Copy source
COPY . .

# Build Vue app
RUN npm run build

# Clean up build artifacts and cache
RUN rm -rf node_modules .npm

# =========================
# Production stage
# =========================
FROM docker.io/library/node:slim
# Install Docker CLI and rclone (needed by your app)
RUN apt-get update \
	&& apt-get install -y --no-install-recommends docker.io docker-compose rclone wget \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy dependency files
COPY package.json package-lock.json* ./

# Install only production deps
RUN npm ci --prefer-offline --no-audit --omit=dev || npm install --omit=dev

# Clean npm cache
RUN npm cache clean --force

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

CMD ["node", "daemon/main.js"]