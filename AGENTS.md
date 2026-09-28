# AGENTS.md — Yantr App Authoring

## Structure
Each app is a single `apps/<app-name>/compose.yml`. No `info.json`, no `Dockerfile`. Optional `logo.svg` in the same folder.

## `x-yantr` metadata block
**Required:** `name`, `tags` (3–5), `short_description` (50–100 chars), `description` (200–300 chars, YAML `>` block), `usecases` (≥2), `website`.
**Optional:** `notes` (list), `env_generators` (`VAR: {length, charset}`; charset ∈ `alnum`, `hex`, `numeric`, `alpha`, `base64url`, `alnum_symbols`), `ports` (list of `{port, protocol, label, service}`; protocol ∈ `HTTP`, `HTTPS`, `TCP`, `UDP`).

Flat arrays (`tags`, `usecases`, `notes`) always use flow sequences:
```yaml
tags: [tools, utility, self-hosted, homelab, podman]
```

Port/display metadata lives only in `x-yantr.ports` — never in labels:
```yaml
x-yantr:
  ports:
    - {port: 8080, protocol: HTTP, label: "Web UI", service: my-app}
```

## Service labels
Do not add `yantr.app` / `yantr.service.N` / `yantr.port.N` labels. Identity comes from the app folder + native compose project/service labels (`core/compose/compose.go:ComposeProjectLabel`). Runtime-only labels (`yantr.expireAt`, `yantr.temporary`, `yantr.system`) are injected by the backend.

## Critical rules
1. **Named Podman volumes only** — never bind mounts (rootless SELinux labeling requires it).
2. **`:latest` only** — never pin a version tag.
3. **Logo** — local `logo.svg`, square, ≥256×256, auto-detected. No URLs; omit if none.
4. **Auto port assignment only** (`"8080"`) — never explicit mapping (`"8080:8080"`), even if the app would normally want a fixed host port (e.g. VPN/peer protocol).
5. **Prebuilt images only** — never `build:`; no Dockerfile/entrypoint.sh in the app folder.
6. **Container socket** — host side must be `${HOST_PODMAN_SOCKET}`, container side stays `/var/run/docker.sock`. Any other `*.sock` host source aborts the deploy (`core/compose/compose.go:applyDockerSocketTransform`).

## Minimal example
```yaml
# apps/my-app/compose.yml
x-yantr:
  name: "myapp"
  tags: [productivity, self-hosted, webapp, tools, podman]
  short_description: "Self-hosted note-taking app."
  description: >
    A self-hosted note-taking service that lets you capture, organize,
    and share notes with your team. Runs entirely on your own hardware.
  usecases: ["Capture notes.", "Organize docs.", "Share with team."]
  website: "https://example.com/docs"
  env_generators:
    ADMIN_PASSWORD: {length: 20, charset: alnum_symbols}
  notes: ["Set admin email in the deploy form if the app requires one."]
  ports:
    - {port: 8080, protocol: HTTP, label: "Web UI", service: my-app}

services:
  my-app:
    image: ghcr.io/example/my-app:latest
    environment:
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

## Validation (manual — no linter exists)
- Every `${VAR}` without a default has a matching `env_generators` entry — otherwise it deploys empty (`core/apps/catalog.go:parseEnvVars`).
- Flat arrays use flow sequences, not block sequences.
- `labels` is a map; every service has `yantr.app`.

Go core changes:
```sh
cd core && go build ./... && go vet ./... && go test ./...
```
`gofmt -l .` shows pre-existing drift in `apps/catalog.go`, `handlers_apps.go`, `handlers_images.go`, `main.go` — don't mass-reformat those.