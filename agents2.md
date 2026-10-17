# Yantr App Authoring

How to add or change an app under `apps/`. This file covers the app recipe
format and the rules for writing one — nothing else.

---

## 1. Add an app — checklist

1. Create one folder: `apps/<app-name>/`.
2. Add one `compose.yml` (and an optional `logo.svg`).
3. Fill the `x-yantr` block — see [§3](#3-x-yantr-reference).
4. Write the compose services under the constraints in [§5](#5-compose-constraints).
5. Mark every user-facing port `show: true` — see [§4](#4-ports-and-visibility).
6. Verify:
   ```sh
   cd core && go build ./... && go vet ./... && go test ./...
   ```

---

## 2. File layout

```
apps/<app-name>/
├── compose.yml   # required — the whole app definition
└── logo.svg      # optional — square, ≥256×256, auto-detected
```

Not allowed: `info.json`, `Dockerfile`, `entrypoint.sh`, or any build context.

---

## 3. `x-yantr` reference

### Required

| Key | Type | Rule |
| --- | --- | --- |
| `name` | string | App name. |
| `tags` | list | 3–5 tags. |
| `short_description` | string | 50–100 chars. |
| `description` | string | 200–300 chars, written as a YAML `>` block. |
| `usecases` | list | At least 2. |
| `website` | string | Project or docs URL. |

### Optional

| Key | Type | Rule |
| --- | --- | --- |
| `notes` | list | Post-deploy hints shown to the user. |
| `developers` | list | Nostr public keys — see below. |
| `env_generators` | map | `VAR: {length, charset}`, charset ∈ `alnum` `hex` `numeric` `alpha` `base64url` `alnum_symbols`. |
| `ports` | list | `{port, protocol, show, label, service}`, protocol ∈ `HTTP` `HTTPS` `TCP` `UDP`. |

Flat arrays (`tags`, `usecases`, `notes`, `developers`) always use flow
sequences, never block sequences:

```yaml
tags: [tools, utility, self-hosted, homelab, podman]
```

### `env_generators`

Every `${VAR}` in the compose file that has no default must have a matching
`env_generators` entry — otherwise it deploys empty.

### `ports`

Port and display metadata lives **only** here — never in compose labels:

```yaml
x-yantr:
  ports:
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: my-app}
```

`service` must name a service that exists in the same file; a dangling one is
dropped at parse time.

### `developers` — crediting an app's author

`developers` names who the app is from. It is optional and changes nothing
about deployment; it drives the developer credit on the app page.

```yaml
x-yantr:
  developers: ["npub1hznmntyj254kqhr079a5gt2wvhyll6rz6q67pyjres4lfkql22kq5ml6zh"]
```

- An entry is an `npub1…` or an `nprofile1…` key.
- Never an `nsec1…` private key, a `note1…` event id, or a `nostr:` prefix.
- Anything that is not a valid key is dropped when the catalog loads.
- Entries are deduplicated on the **decoded** pubkey, not the spelling — one
  account written as an npub and again as an nprofile is one developer.
- Credit the **upstream author of the software**, not whoever wrote the recipe.
  Leave the field out when you cannot tie a key to a real account.

---

## 4. Ports and visibility

Every published port is one of two kinds. Decide which before you write it.

| | `show: true` — access port | omitted — work port |
| --- | --- | --- |
| What it is | A UI a browser can open | Protocol plumbing; nothing to click |
| In the UI | Own card with a working **Open** button | Hidden by default; listed under **All** |
| Rule | `protocol: HTTP` or `HTTPS` | Everything else |

`show` is **deny-by-default**: omitting it does not unpublish the port, it only
removes it from the user's default view. The rule to apply is: *if a browser can
show the user something, it is `show: true`; otherwise it is a work port.*

Omit `show` for:

- Any `TCP` / `UDP` port — torrent peers, WireGuard, Syncthing relays, database
  clients, git SSH. A browser cannot open them, so an "Open" button would lie.
- `HTTP` ports that are diagnostics rather than UI — metrics, pprof, health
  checks. Being HTTP is not enough; ask whether a human looks at it.

```yaml
ports:
  - {port: 45001, protocol: HTTP, show: true, label: "Torrent UI", service: torbox}
  - {port: 45000, protocol: HTTP, show: true, label: "File Server", service: torbox}
  - {port: 51413, protocol: TCP, label: "Peer Port", service: torbox}
  - {port: 51413, protocol: UDP, label: "Peer Port", service: torbox}
```

Two consequences:

- **Every port is still catalogued, shown or not.** A work port still
  identifies its service, so it still marks that service as the stack's primary
  one. Dropping the entry instead of omitting `show` would lose that.
- **Host ports are auto-assigned, so hiding a port does not make it
  unreachable** — the user just cannot discover the number. Say so in `notes`
  when a work port matters operationally.

An app may legitimately have zero access ports (a database, a file share). The
automated tests fail an app that publishes ports but marks none `show: true`;
such apps are allow-listed there deliberately.

---

## 5. Compose constraints

1. **Named Podman volumes only** — never bind mounts (rootless SELinux labeling
   requires it).
2. **`:latest` only** — never pin a version tag.
3. **Prebuilt images only** — never `build:`.
4. **Auto port assignment only** (`"8080"`) — never explicit mapping
   (`"8080:8080"`), even when the app normally wants a fixed host port.
5. **No Yantr labels** — do not add `yantr.app` / `yantr.service.N` /
   `yantr.port.N`. Identity comes from the app folder plus the native compose
   project/service labels. Runtime-only labels (`yantr.expireAt`,
   `yantr.temporary`, `yantr.system`) are injected by the backend.
6. **Container socket** — host side must be `${HOST_PODMAN_SOCKET}`, container
   side stays `/var/run/docker.sock`. Any other `*.sock` host source aborts the
   deploy. The placeholder is set by default on every deploy; never assume a
   fixed socket path exists.

---

## 6. Example

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

---

## 7. Validation

### Automated

`core/apps/catalog_test.go` reads the real `apps/` tree and fails on:

- an app that publishes ports but marks none `show: true`;
- a `show: true` port whose protocol is not `HTTP` / `HTTPS` (a dead "Open"
  button);
- a diagnostics port (`Metrics / pprof`, `Postgres client`, `gRPC service`)
  wrongly marked `show: true`.

Databases and file shares are allow-listed as legitimately having no access
port. A new one must be added there deliberately, not by accident.

### Manual — no linter exists

- Every `${VAR}` without a default has a matching `env_generators` entry.
- Flat arrays use flow sequences, not block sequences.
- `x-yantr.ports[].service` names a service that exists in the same file.
