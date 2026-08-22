# Yantr — App Authoring Instructions

## App Structure

Each app lives in `apps/<app-name>/compose.yml` — a single file with no `info.json` and no `Dockerfile`.

All metadata is in the top-level `x-yantr` key. Docker Compose ignores `x-*` fields, so the file remains fully deployable.

### x-yantr Metadata Block

**Required fields:**
- `name` — display name (lowercase, alphanumeric only, no spaces)
- `tags` — 3–5 lowercase strings
- `short_description` — 50–100 chars
- `description` — 200–300 chars (use YAML `>` block scalar for multi-line)
- `usecases` — ≥2 strings
- `website` — https:// URL

**Optional fields:**
- `logo` — omitted; place `logo.svg` in the app folder instead (auto-detected)
- `notes` — list of strings explaining manual setup steps
- `customapp` — boolean; `true` for Yantr-built apps. Define the image with `dockerfile_inline` in compose.yml (never a separate Dockerfile)
- `env_generators` — map of `VAR → {length, charset}` for auto-generated secrets. `charset` values: `alnum`, `hex`, `numeric`, `alpha`, `base64url`, `alnum_symbols`

**YAML style — always use flow sequences for flat arrays (`tags`, `usecases`, `notes`):**
```yaml
# ✅ correct
tags: [tools, utility, self-hosted, homelab, docker]
usecases: ["Use case one.", "Use case two."]
notes: ["Note one.", "Note two."]

# ❌ wrong — check.js WILL fail
tags:
  - tools
  - utility
```

### Port Labels on Services

`labels` MUST always be a map (key-value dict), never a sequence. `yantr.app` is strictly required on every service.

```yaml
labels:
  yantr.app: "my-app"
  yantr.service.8080: "Web UI"
  yantr.port.8080: "HTTP"
```

Supported protocols: `HTTP`, `HTTPS`, `TCP`, `UDP`

### x-auth Block (Optional)

```yaml
x-auth:
  port: 3002       # Caddy listens here (auth-protected public port)
  username: admin
  password: secret # bcrypted by Yantr at deploy time, then discarded
```

## Critical Rules

### 1. Always Use Docker Volumes
All persistent data MUST use named Docker volumes — never bind mounts. Declare every volume at the top-level `volumes:` key.

```yaml
# ✅ correct
volumes:
  - my_app_data:/data

volumes:
  my_app_data:

# ❌ wrong
volumes:
  - ./data:/data
```

### 2. Always Use Latest Images
Always use the `:latest` tag (or the upstream's equivalent rolling tag). Never pin to a specific version number.

```yaml
# ✅ correct
image: ghcr.io/example/my-app:latest

# ❌ wrong
image: ghcr.io/example/my-app:1.2.3
```

### 3. Logo — SVG
Place a `logo.svg` in the app folder alongside `compose.yml` (auto-detected, no field needed). SVGs must be square, minimum 256×256. Omit if no logo is available — do not use URLs.

```yaml
# ✅ correct — local SVG (auto-detected, no logo field needed)
# apps/my-app/logo.svg  (256x256 or larger, square)

# ❌ wrong
logo: "https://example.com/logo.png"
```

### 4. Prefer Auto Port Assignment — Avoid Mapped Ports
Use Docker's automatic port assignment `"8080"` instead of explicit host mappings `"8080:8080"`. Only use mapped ports when the app absolutely cannot function without a fixed host port (e.g. a VPN or peer protocol that must bind to a specific port).

```yaml
# ✅ correct — let Docker assign the host port
ports:
  - "8080"

# ❌ wrong — avoid unless the app cannot work without it
ports:
  - "8080:8080"
```

### 5. Custom Apps — Use dockerfile_inline
Yantr-built apps (`customapp: true`) MUST define the image with Compose `build.dockerfile_inline`. Do not add a `Dockerfile`, `entrypoint.sh`, or other build files in the app folder — keep it to `compose.yml` and optional `logo.svg`. If you need an entrypoint script, write it inside the inline Dockerfile.

```yaml
# ✅ correct
services:
  my-app:
    build:
      context: .
      dockerfile_inline: |
        FROM debian:stable
        RUN apt-get update && apt-get install -y --no-install-recommends curl
        CMD ["my-app"]

# ❌ wrong — never add apps/my-app/Dockerfile
services:
  my-app:
    build: .
```

## Minimal App Example

```yaml
# apps/my-app/compose.yml
x-yantr:
  name: "myapp"
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
    labels:
      yantr.app: "my-app"
      yantr.service.8080: "Web UI"
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

## Validation

Run `node check.js` after any app changes. It will fail if:
- `compose.yml` uses `${VAR}` without a default and `env_generators` is missing the matching entry
- Flat arrays use block sequences instead of flow sequences