# Yantr — Agent Instructions

## Project Overview
Self-hosted app store running as a Docker container alongside existing OS. Vue 3 frontend + **Go** API backend managing Docker Compose stacks via Docker socket.

## Commands

| Command | Purpose |
|---------|----------|
| `npm run dev` | Start Vite dev server (proxies `/api` to `localhost:5252`) |
| `npm run build` | Build production Vue app to `dist/` |
| `npm run preview` | Preview production build |
| `npm run docker` | Build & run Docker image with host network + Docker socket |
| `npm run website` | Build static website to `website/` |
| `node check.js` | Validate all apps in `apps/` (run after any app changes) |

## Architecture

**Frontend** (`src/`): Vue 3 + Vue Router + Pinia (implicit), Tailwind CSS v4

**Backend** (`core/`): Go HTTP server on port 5252 (replaces `daemon/`)
- Entry: `core/main.go` (HTTP server → routes → Caddy subprocess)
- Packages: `auth/` (HMAC-SHA256 token auth), `apps/` (catalog reader), `compose/` (YAML ops), `caddy/` (reverse proxy), `docker/` (SDK client), `selfinstall/` (bootstrap), `shared/` (logging), `system/` (IP identity)
- Auth: stateless HMAC-SHA256 JWT (compatible with browser `SubtleCrypto`), configured via `POST /api/setup/admin` with `{username, secretHex}` (64-char hex = 32-byte key)
- Public paths: `/api/health`, `/api/version`, `/api/setup/status`, `/api/setup/admin`, `/api/auth/login`
- Volume browser: `browser.go` manages dufs subprocess instances per volume

**Apps Catalog** (`apps/`): 130+ apps, each with:
- `compose.yml` — Docker Compose with `yantr.app` + `yantr.service` labels
- `info.json` — metadata (name, logo CID, tags ≥6, ports[], short_description 50-100, description 200-300, usecases ≥2, website https, notes[], env_generators)

## Critical Conventions

### App Development
- **Always run `node check.js` after editing `apps/`** — validates info.json/compose.yml format, port conflicts, required fields
- **Hard rules** (from `apps/apps.md`):
  - Never touch host filesystem — all persistent data uses Docker volumes
  - Never use host bind mounts except `/var/run/docker.sock`, `/dev/net/tun`, local helper files
  - Always use named Docker volumes for databases, config, uploads, media, logs, caches
- `compose.yml` requirements:
  - Valid YAML, deployable with `docker compose`
  - Required labels: `yantr.app` (folder name), `yantr.service` (display label)
  - Environment: key-value format preferred (not list)
  - User-provided values: use `${VAR}` or `${VAR:-default}` — never hardcode secrets/passwords as literals
  - Credentials/secrets: no placeholder defaults (avoid `:-admin`, `:-password`, `:-changeme`); use `ADMIN_PASSWORD: ${ADMIN_PASSWORD}`
  - **`check.js` enforces `env_generators`**: any compose value written as `${VAR}` (no `:-` fallback) must have a matching `env_generators` entry in `info.json`, except system vars (`TZ`, `PUID`, `PGID`, `TUNNEL_TOKEN`, `TAILSCALE_AUTH_KEY`, `TELEGRAM_BOT_TOKEN`, `NOSTR_NSEC`, `AUTHCODE`). `${VAR:-default}` is optional in the deploy form and does not require `env_generators`.
  - Ports: container-only format (`"8096"` not `"8096:8096"`)
  - Prefer `:latest` image tags
  - Volumes: named Docker volumes only
- `info.json` required fields:
  - `name`, `logo` (IPFS CID — upload at `https://originless.besoeasy.com/upload`), `tags` (≥6 lowercase), `ports[]` (port/protocol/label), `short_description` (50-100 chars), `description` (200-300 chars), `usecases` (≥2), `website` (https://)
  - Optional: `notes[]`, `customapp` (boolean — true for Yantr-built apps with Dockerfile; shows "Built by Yantr" badge, disables auto-update), `env_generators`
  - `env_generators`: map env var → generation rules (`length`, `charset`: `alnum`/`hex`/`numeric`/`alpha`/`base64url`/`alnum_symbols`, optional `characters`, `regex`). Required for every `${VAR}` reference in `compose.yml` that has no default (validated by `check.js`).
- Checklist: `yantr.app` = folder name, valid compose, no placeholder defaults for secrets, every `${VAR}` (no default) has `env_generators`, `notes` explains manual setup, ports documented

### Minimal App Example
```yaml
# compose.yml
services:
  my-app:
    image: ghcr.io/example/my-app:latest
    container_name: my-app
    labels:
      yantr.app: "my-app"
      yantr.service: "Web UI"
    environment:
      TZ: ${TZ:-UTC}
      ADMIN_USER: ${ADMIN_USER:-admin}
      ADMIN_PASSWORD: ${ADMIN_PASSWORD}
    ports:
      - "8080"
    volumes:
      - my_app_data:/data
    restart: unless-stopped

volumes:
  my_app_data:
```

```json
// info.json
{
  "name": "My App",
  "logo": "VALID IPFS CID or empty",
  "tags": ["productivity", "self-hosted", "webapp"],
  "ports": [{ "port": 8080, "protocol": "HTTP", "label": "Web UI" }],
  "short_description": "Self-hosted note-taking app.",
  "description": "A self-hosted note-taking service.",
  "usecases": ["Capture notes.", "Organize docs.", "Share with team."],
  "website": "https://example.com/docs",
  "env_generators": {
    "ADMIN_PASSWORD": { "length": 20, "charset": "alnum_symbols" }
  },
  "notes": ["Set admin email in the deploy form if the app requires one."]
}
```

### Frontend (from `.github/copilot-instructions.md`)
- CSS tokens: `--bg-body`, `--text-primary`, `--text-secondary`, `--surface`, `--surface-muted`
- Flat surfaces only — no gradients, glass morphism, backdrop-filter
- No borders/rings unless required; use `smooth-shadow`/`smooth-shadow-lg` for elevation
- Mobile-first, no horizontal scroll, 44x44 tap targets
- Lucide icons only; every interactive element needs hover state
- Buttons: leading Lucide icon unless space constrained
- Animations: subtle, use existing motion tokens
- Compact card/list layouts over dense tables
- Dashboard cards: minimal identity-card style (quiet header, one strong focal line, ≤2 detail rows, clear bottom action); no nested backgrounds, pills, borders, rings, or dividers; emphasis via spacing/typography/icons/hover motion

### Backend
- Auth: daku public key → stateless in-memory by default; set `dakupublickey` env to persist
- API: all `/api/*` and `/browse/*` require bearer token after setup
- Public paths: `/api/health`, `/api/version`, `/api/setup/status`, `/api/setup/admin`, `/api/auth/login`
- Docker socket: `/var/run/docker.sock` (mounted in container)

## Docker Build
- Multi-stage: Node LTS builder → Node Alpine final
- Final image installs: `docker-cli`, `docker-cli-compose`, `wget`, `dufs`, `caddy`
- Copies: `dist/`, `daemon/`, `apps/`
- Healthcheck: `wget http://127.0.0.1:5252/api/health`
- Runs: `node daemon/index.js`

## Testing / Validation
- No formal test suite — `node check.js` is the primary validation
- `check.js` fails if `compose.yml` uses `${VAR}` without a default and `info.json` is missing the matching `env_generators` entry
- Manual verification: `npm run docker` → open `http://localhost:5252`

## Key Files to Reference
- `check.js` — validation rules (run after app changes)
- `daemon/shared.js` — Docker socket path, logging, shared constants
- `daemon/auth.js` — daku token verification