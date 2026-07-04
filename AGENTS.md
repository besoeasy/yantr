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

**Apps Catalog** (`apps/`): 130+ apps, each with a single `compose.yml` that carries both the Docker Compose service definition and all app metadata under the `x-yantr` extension key. `info.json` is deprecated — do not create new ones.

## Critical Conventions

### x-yantr Metadata Block
All app metadata lives in the top-level `x-yantr` key of `compose.yml`. Docker Compose ignores `x-*` extension fields, so the file remains fully deployable.

**Required fields:**
- `name` — display name
- `tags` — 3–5 lowercase strings
- `short_description` — 50–100 chars
- `description` — 200–300 chars (use YAML `>` block scalar for multi-line)
- `usecases` — ≥2 strings
- `website` — https:// URL

**Optional fields:**
- `logo` — IPFS CID (upload at `https://originless.besoeasy.com/upload`); omit if no logo yet
- `notes` — list of strings explaining manual setup steps
- `customapp` — boolean; `true` for Yantr-built apps with Dockerfile (shows "Built by Yantr" badge, disables auto-update)
- `env_generators` — map of `VAR → {length, charset}` for auto-generated secrets. `charset` values: `alnum`, `hex`, `numeric`, `alpha`, `base64url`, `alnum_symbols`

**YAML style rule — always use flow sequences for flat arrays:**
```yaml
# ✅ compact flow sequence (preferred)
tags: [tools, utility, self-hosted, homelab, docker, open-source]
usecases: ["Use case one.", "Use case two."]
notes: ["Note one.", "Note two."]

# ❌ block sequence (only for structured objects like env_generators)
tags:
  - tools
  - utility
```
Use block sequences only when list items are objects with sub-fields (e.g. `env_generators`).

### Port Labels on Services
Port metadata is declared directly on each service via labels — collocated with the `ports:` declaration it describes.

```yaml
# Pattern: yantr.port.{PORT_NUMBER}: "PROTOCOL"
labels:
  yantr.service: "Web UI"      # human-readable name for this service endpoint
  yantr.port.8080: "HTTP"      # port 8080 speaks HTTP

# Multi-port example
labels:
  yantr.service: "App"
  yantr.port.80: "HTTP"
  yantr.port.443: "HTTPS"
```

Supported protocols: `HTTP`, `HTTPS`, `TCP`, `UDP`

### x-auth Block (Optional, Deploy-time Only)
Yantr reads `x-auth` at deploy time to configure a Caddy basic-auth proxy in front of the app. Docker Compose ignores `x-*` fields. The password is **never stored in container labels or visible via `docker inspect`** — Yantr bcrypts it, writes to Caddy config, then discards the plain text.

```yaml
x-auth:
  port: 3002       # Caddy listens here (auth-protected public port)
  username: admin  # basic auth username
  password: secret # plain text — bcrypted by Yantr at deploy time, then discarded
```

**How it works:**
- App runs on its native port (e.g. `3000`) — internal only
- Caddy listens on `x-auth.port` (e.g. `3002`) with basicauth → reverse proxies to `3000`
- Users access the app on port `3002` with username/password
- Omit `x-auth` entirely for apps that need no auth

```yaml
# Full example with auth
x-yantr:
  name: "My App"
  ...

x-auth:
  port: 3002
  username: admin
  password: secret

services:
  my-app:
    ports:
      - "3000"
    labels:
      yantr.service: "Web UI"
      yantr.port.3000: "HTTP"
```

### Minimal App Example

```yaml
# compose.yml — single file, no info.json needed
x-yantr:
  name: "My App"
  logo: "VALID IPFS CID or empty"
  tags: [productivity, self-hosted, webapp, tools, docker]
  short_description: "Self-hosted note-taking app."
  description: >
    A self-hosted note-taking service that lets you capture, organize,
    and share notes with your team. Runs entirely on your own hardware.
  usecases: ["Capture notes.", "Organize docs.", "Share with team."]
  website: "https://example.com/docs"
  env_generators:
    ADMIN_PASSWORD: {length: 20, charset: alnum_symbols}
  notes: ["Set admin email in the deploy form if the app requires one."]

services:
  my-app:
    image: ghcr.io/example/my-app:latest
    container_name: my-app
    labels:
      yantr.service: "Web UI"
      yantr.port.8080: "HTTP"
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
- `check.js` fails if `compose.yml` uses `${VAR}` without a default and `x-yantr.env_generators` is missing the matching entry
- Manual verification: `npm run docker` → open `http://localhost:5252`

## Key Files to Reference
- `check.js` — validation rules (run after app changes)
- `daemon/shared.js` — Docker socket path, logging, shared constants
- `daemon/auth.js` — daku token verification