# AGENTS.md — Yantr App Authoring

## Structure
Each app is a single `apps/<app-name>/compose.yml`. No `info.json`, no `Dockerfile`. Optional `logo.svg` in the same folder.

## `x-yantr` metadata block
**Required:** `name`, `tags` (3–5), `short_description` (50–100 chars), `description` (200–300 chars, YAML `>` block), `usecases` (≥2), `website`.
**Optional:** `notes` (list), `env_generators` (`VAR: {length, charset}`; charset ∈ `alnum`, `hex`, `numeric`, `alpha`, `base64url`, `alnum_symbols`), `ports` (list of `{port, protocol, show, label, service}`; protocol ∈ `HTTP`, `HTTPS`, `TCP`, `UDP`).

Flat arrays (`tags`, `usecases`, `notes`) always use flow sequences:
```yaml
tags: [tools, utility, self-hosted, homelab, podman]
```

Port/display metadata lives only in `x-yantr.ports` — never in labels:
```yaml
x-yantr:
  ports:
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: my-app}
```

## Port visibility — `show`
Every port is one of two kinds. Decide which before you write the entry.

| | `show: true` — **access port** | omitted — **work port** |
|---|---|---|
| What it is | A UI the user opens in a browser | Protocol plumbing; nothing to click |
| In the UI | Own card, with a working **Open** button | Hidden by default; listed under **All** |
| Rule | `protocol: HTTP` or `HTTPS` | Everything else |

**`show` is deny-by-default.** Omitting it does not hide the port from the container — it is still published and still works. It only removes it from the user's default view. A work port renders as a muted "Work port" card under the **All** toggle, so nothing becomes undebuggable.

The whole point of the flag is that a four-port app should not present four dead cards. Write the rule as: *if a browser can show the user something, it is `show: true`; otherwise it is a work port.*

Work ports — omit `show`:
- Any `TCP` / `UDP` port. This covers torrent peer ports, WireGuard tunnels, Syncthing relays, database clients, and git SSH. A browser cannot open any of them, so an "Open" button would be a lie.
- `HTTP` ports that are diagnostics rather than UI — metrics, pprof, health checks. Being HTTP is not enough; ask whether a human looks at it.

```yaml
ports:
  - {port: 45001, protocol: HTTP, show: true, label: "Torrent UI", service: torbox}
  - {port: 45000, protocol: HTTP, show: true, label: "File Server", service: torbox}
  - {port: 51413, protocol: TCP, label: "Peer Port", service: torbox}
  - {port: 51413, protocol: UDP, label: "Peer Port", service: torbox}
```

Two consequences worth knowing:
- **Every port is still catalogued, shown or not.** A work port still identifies its service, so it still marks that service as the stack's primary one (`core/handlers_stacks.go:primaryServices`). Dropping the entry instead of omitting `show` would lose that.
- **Host ports are auto-assigned, so hiding a port does not make it unreachable** — the user just cannot discover the number. Say so in `notes` when a work port matters operationally (e.g. "port 51413 is the peer port; forward the assigned host port on your router").

An app may legitimately have zero access ports — a database or a file share has no web UI. The tests in Validation fail an app that publishes ports but marks none `show: true`; such apps are allow-listed there deliberately.

## Service labels
Do not add `yantr.app` / `yantr.service.N` / `yantr.port.N` labels. Identity comes from the app folder + native compose project/service labels (`core/compose/compose.go:ComposeProjectLabel`). Runtime-only labels (`yantr.expireAt`, `yantr.temporary`, `yantr.system`) are injected by the backend.

## Critical rules
1. **Named Podman volumes only** — never bind mounts (rootless SELinux labeling requires it).
2. **`:latest` only** — never pin a version tag.
3. **Logo** — local `logo.svg`, square, ≥256×256, auto-detected. No URLs; omit if none.
4. **Auto port assignment only** (`"8080"`) — never explicit mapping (`"8080:8080"`), even if the app would normally want a fixed host port (e.g. VPN/peer protocol).
5. **Prebuilt images only** — never `build:`; no Dockerfile/entrypoint.sh in the app folder.
6. **Container socket** — host side must be `${HOST_PODMAN_SOCKET}`, container side stays `/var/run/docker.sock`. Any other `*.sock` host source aborts the deploy (`core/compose/compose.go:applyDockerSocketTransform`).
7. **Mark every user-facing port `show: true`** — the key is optional and defaults to off, so a forgotten port is a work port nobody can click. See Port visibility.

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
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: my-app}

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

## Validation
Automated — `core/apps/catalog_test.go` reads the real `apps/` tree and fails on:
- an app that publishes ports but marks none `show: true` (its stack view would render no reachable port);
- a `show: true` port whose protocol is not `HTTP`/`HTTPS` (a dead "Open" button — the exact noise `show` exists to remove);
- a diagnostics port (`Metrics / pprof`, `Postgres client`, `gRPC service`) wrongly marked `show: true`.

Databases and file shares are allow-listed as legitimately having no access port. A new one must be added there deliberately, not by accident.

Manual — no linter exists:
- Every `${VAR}` without a default has a matching `env_generators` entry — otherwise it deploys empty (`core/apps/catalog.go:parseEnvVars`).
- Flat arrays use flow sequences, not block sequences.
- `x-yantr.ports[].service` names a service that exists in the same file (a dangling one is dropped at parse time).

Go core changes:
```sh
cd core && go build ./... && go vet ./... && go test ./...
```
`gofmt -l .` shows pre-existing drift in `apps/catalog.go`, `handlers_apps.go`, `handlers_images.go`, `main.go` — don't mass-reformat those.