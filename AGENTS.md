# AGENTS.md — Yantr App Authoring

## Structure
Each app is a single `apps/<app-name>/compose.yml`. No `info.json`, no `Dockerfile`. Optional `logo.svg` in the same folder.

## `x-yantr` metadata block
**Required:** `name`, `tags` (3–5), `short_description` (50–100 chars), `description` (200–300 chars, YAML `>` block), `usecases` (≥2), `website`.
**Optional:** `notes` (list), `developers` (list of `platform:handle`), `env_generators` (`VAR: {length, charset}`; charset ∈ `alnum`, `hex`, `numeric`, `alpha`, `base64url`, `alnum_symbols`), `ports` (list of `{port, protocol, show, label, service}`; protocol ∈ `HTTP`, `HTTPS`, `TCP`, `UDP`).

Flat arrays (`tags`, `usecases`, `notes`, `developers`) always use flow sequences:
```yaml
tags: [tools, utility, self-hosted, homelab, podman]
```

## `developers` — crediting an app's author
`developers` names who the app is from, as `platform:handle` entries. It is optional and
changes nothing about deployment; it drives the developer credit on the app page, the
`#/developer/<handle>` profile route, and the `@mention` in a pre-filled issue body.

```yaml
x-yantr:
  developers: ["github:besoeasy", "gitlab:username2"]
```

The platform set is **closed**: `github`, `gitlab`, `codeberg`, `nostr`. Anything else is
dropped at parse time rather than rendered, because an entry becomes a profile link and,
on GitHub, an `@mention` in a filed issue — a wrong handle pings a stranger. An entry with
no colon, an empty handle, or a handle containing `/`, `?` or `#` is dropped the same way.
Handles are lowercased on the forge platforms (they are case-insensitive there, so
`Besoeasy` and `besoeasy` are one developer); Nostr keys keep their case.

Credit the **upstream author of the software** — the person whose project it is — not
whoever added the recipe. Everyone in the catalog ships a recipe written by somebody, so
the alternative reading makes the field meaningless. Only write a handle you can tie to a
real account: the image namespace (`ghcr.io/<owner>`) or the project's own website is
evidence; a guess is not. Leaving the field out is the correct choice when unsure.

A `nostr:` handle must be an `npub1…` (or `nprofile1…`) — a profile identifier, never an
`nsec1…` private key or a `note1…` event id. It buys more than a link: the developer page
shows the account's kind-0 name, picture and `nip05`, and a QR code of the NIP-21 URI
(`nostr:npub1…`) so a visitor can scan straight into a direct message. The QR is drawn
from the catalog's own key, so it is on screen even when nothing can be fetched.

That fetch is the one place Yantr reaches a third party for this field: the server queries
a single relay, `relay.primal.net`, over a websocket and caches the result for seven days
(`core/system/nostr.go`). One relay, not a merge of several — a developer page wants one
account's name, and extra relays would multiply the latency and the failure modes for a
result that is nearly always identical. Nothing else about a `nostr:` entry depends on the
network: the app list, the credit and the QR all come from the local catalog.

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

## Installer
`install.sh` lives at the repo root and is **not** published on the website. The documented one-liner fetches it from jsDelivr, which resolves `@latest` to the newest tagged release:

```
curl -fsSL https://cdn.jsdelivr.net/gh/besoeasy/yantr@latest/install.sh | bash
```

That is deliberate: users get a released installer, never an unreviewed commit off `main`. Keep the URL as the `INSTALL_URL` variable at the top of the script and reference it everywhere the script prints its own usage (help text, the uninstall hint) — do not hardcode a second copy. If you add a self-referencing URL, use a tag (`@v1.2.3`) rather than a branch (`@main`), or the installer stops being release-pinned.

The generated Quadlet also mounts `/:/host:ro` and `/sys:/sys:ro` for the host metrics dashboard. Those are Yantr's *own* binds and do not relax rule 1 — apps still use named volumes only. They are host-path binds, which normally need `:z` relabeling and would break on Fedora without it; they work only because the same Quadlet sets `SecurityLabelDisable=true`. Treat the two lines as a pair.

## Critical rules
Yantr is **rootless-only**, and that is a load-bearing constraint rather than a default. `install.sh` refuses to run as root, and no part of the stack supports rootful Podman. A root-mode install would attach Yantr to a rootful Podman service, so every app it deployed would hold real host root — exactly what the rootless architecture exists to prevent. Do not reintroduce a root code path, and do not assume `/run/podman/podman.sock` or `/var/run/podman/podman.sock` ever exists: the host socket is always the per-user `%t/podman/podman.sock`.

1. **Named Podman volumes only** — never bind mounts (rootless SELinux labeling requires it).
2. **`:latest` only** — never pin a version tag.
3. **Logo** — local `logo.svg`, square, ≥256×256, auto-detected. No URLs; omit if none.
4. **Auto port assignment only** (`"8080"`) — never explicit mapping (`"8080:8080"`), even if the app would normally want a fixed host port (e.g. VPN/peer protocol).
5. **Prebuilt images only** — never `build:`; no Dockerfile/entrypoint.sh in the app folder.
6. **Container socket** — host side must be `${HOST_PODMAN_SOCKET}`, container side stays `/var/run/docker.sock`. Any other `*.sock` host source aborts the deploy (`core/compose/compose.go:applyDockerSocketTransform`). The placeholder is set by default: `install.sh` writes the host socket path into the Quadlet as `HOST_PODMAN_SOCKET` (as does `test.sh` and the documented hand-rolled installs), so it resolves on every deploy without the app doing anything. There is no runtime fallback discovery — if it is ever unset, `podman.HostSocket()` logs one warning to stderr and returns the in-container path, which does not exist on a rootless host, so apps mounting `/var/run/docker.sock` fail to deploy.
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

The root `package.json` belongs to the website builder (`website-build.js`), not the app UI — `ui/` has its own. Rebuild yantr.org locally with `npm i && node website-build.js` from the repo root; output (`website/apps.json`, `website/apps/`, …) is gitignored and deployed by `.github/workflows/website.yml`.