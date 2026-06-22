import { runSelfInstallIfNeeded } from "./selfinstall.js";

// ─── Self-install bootstrap ───────────────────────────────────────────────────
// When the user runs the minimal install command (docker run -v /var/run/docker.sock:...)
// we detect the missing configuration, re-launch yantr with the full production
// flags (host network, volumes mount, restart policy, name), then exit this
// ephemeral bootstrap container.  If already fully configured, this is a no-op.
const bootstrapped = await runSelfInstallIfNeeded();
if (bootstrapped) process.exit(0);

import Fastify from "fastify";
import fastifyCors from "@fastify/cors";
import fastifyStatic from "@fastify/static";
import path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";

import { resolveComposeCommand } from "./compose.js";
import { errorHandler } from "./utils.js";
import { startCleanupScheduler } from "./cleanup.js";
import { initAutoUpdate } from "./autoupdate.js";
import { socketPath, log } from "./shared.js";
import { stopAll as stopAllBrowsers } from "./dufs.js";
import { extractBearerToken, verifyYantrAuthToken } from "./auth.js";

import authRoutes from "./routes/auth.js";
import systemRoutes from "./routes/system.js";
import containersRoutes from "./routes/containers.js";
import stacksRoutes from "./routes/stacks.js";
import appsRoutes from "./routes/apps.js";
import imagesRoutes from "./routes/images.js";
import volumesRoutes from "./routes/volumes.js";
import proxyRoutes from "./routes/proxy.js";
import { startCaddy, stopCaddy } from "./caddy.js";
import { startPresenceScheduler } from "./telemetry.js";
import { getBrowserPort } from "./dufs.js";
import http from "node:http";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const PUBLIC_API_PATHS = new Set([
  "/api/health",
  "/api/version",
  "/api/setup/status",
  "/api/setup/admin",
  "/api/auth/login",
]);

const fastify = Fastify({ logger: false });

fastify.decorateRequest("yantrUser", null);

// ─── CORS ─────────────────────────────────────────────────────────────────────
await fastify.register(fastifyCors, { origin: "*" });

fastify.addHook("onRequest", async (request, reply) => {
  const pathname = String(request.raw.url || "").split("?")[0] || "/";
  const isProtectedPath = pathname.startsWith("/browse/") || pathname.startsWith("/api/");

  if (!isProtectedPath || PUBLIC_API_PATHS.has(pathname)) return;

  const result = await verifyYantrAuthToken(extractBearerToken(request));
  if (!result.config) {
    return reply.code(503).send({ success: false, error: "Setup required", code: "SETUP_REQUIRED" });
  }
  if (!result.ok) {
    return reply.code(401).send({ success: false, error: "Unauthorized", code: result.reason || "UNAUTHORIZED" });
  }

  request.yantrUser = {
    username: result.config.username,
    publicKey: result.publicKey,
  };
});

// ─── Dufs proxy (root scope — must be before static + route registration) ────
fastify.addHook('onRequest', (req, reply, done) => {
  if (!req.raw.url.startsWith('/browse/')) return done()

  reply.hijack()
  const parts = req.raw.url.split('/')   // ['', 'browse', 'volumeName', ...]
  const volumeName = decodeURIComponent(parts[2] || '')
  const port = getBrowserPort(volumeName)

  if (!port) {
    reply.raw.writeHead(503, { 'content-type': 'text/plain; charset=utf-8' })
    reply.raw.end(`Volume browser for "${volumeName}" is not running. Start it from the Volumes page.`)
    return done()
  }

  const proxy = http.request(
    { hostname: 'localhost', port, path: req.raw.url, method: req.raw.method, headers: { ...req.raw.headers, host: `localhost:${port}` } },
    (res) => {
      const skip = new Set(['transfer-encoding', 'connection'])
      const headers = {}
      for (const [k, v] of Object.entries(res.headers)) {
        if (!skip.has(k.toLowerCase())) headers[k] = v
      }
      reply.raw.writeHead(res.statusCode, headers)
      res.pipe(reply.raw)
    }
  )
  proxy.on('error', () => {
    if (!reply.raw.headersSent) reply.raw.writeHead(502)
    reply.raw.end()
  })
  req.raw.pipe(proxy)
  done()
})

// ─── Static UI (production only) ─────────────────────────────────────────────
function normalizeUiBasePath(value) {
  if (!value || value === "/") return "/";
  const trimmed = String(value).trim();
  if (!trimmed) return "/";
  const withLeadingSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  return withLeadingSlash.replace(/\/+$/, "");
}

if (process.env.NODE_ENV === "production") {
  const uiDistPath = path.join(__dirname, "..", "dist");
  const uiBasePath = normalizeUiBasePath(process.env.UI_BASE_PATH || process.env.VITE_BASE_PATH || "/");

  await fastify.register(fastifyStatic, {
    root: uiDistPath,
    prefix: uiBasePath,
    wildcard: false,
    decorateReply: true,
  });

  log("info", `📦 Serving Vue.js app from: ${uiDistPath}`);
  log("info", `🧭 UI virtual root: ${uiBasePath}`);
}

// ─── API Routes ───────────────────────────────────────────────────────────────
await fastify.register(authRoutes);
await fastify.register(systemRoutes);
await fastify.register(containersRoutes);
await fastify.register(stacksRoutes);
await fastify.register(appsRoutes);
await fastify.register(imagesRoutes);
await fastify.register(volumesRoutes);
await fastify.register(proxyRoutes);

// ─── Error handler ────────────────────────────────────────────────────────────
fastify.setErrorHandler(errorHandler);

// ─── SPA fallback (production only, after API routes) ────────────────────────
if (process.env.NODE_ENV === "production") {
  fastify.setNotFoundHandler((_request, reply) => {
    reply.sendFile("index.html");
  });
}

// ─── Start server ─────────────────────────────────────────────────────────────
const PORT = 5252;
try {
  await fastify.listen({ port: PORT, host: "0.0.0.0" });

  log("info", "\n" + "=".repeat(50));
  log("info", "🚀 Yantr API Server Started");
  log("info", "=".repeat(50));
  log("info", `📡 Port: ${PORT}`);
  log("info", `🔌 Socket: ${socketPath}`);
  log("info", `📂 Apps directory: ${path.join(__dirname, "..", "apps")}`);
  log("info", `🌐 Access: http://localhost:${PORT}`);
  log("info", "=".repeat(50) + "\n");

  resolveComposeCommand({ socketPath, log }).catch((err) => {
    log("warn", `⚠️  [COMPOSE] ${err.message}`);
  });

  log("info", "🧹 Starting automatic cleanup scheduler");
  startCleanupScheduler(11);

  log("info", "🔄 Starting auto-update (self-update scheduler)");
  initAutoUpdate(log);

  log("info", "🔒 Starting embedded Caddy proxy");
  startCaddy().catch((err) => {
    log("warn", `⚠️  [CADDY] ${err.message}`);
  });

  startPresenceScheduler();
} catch (err) {
  console.error("Failed to start server:", err);
  process.exit(1);
}

for (const signal of ["SIGTERM", "SIGINT"]) {
  process.on(signal, () => {
    log("info", `Received ${signal}, shutting down...`);
    stopAllBrowsers();
    stopCaddy();
    process.exit(0);
  });
}
