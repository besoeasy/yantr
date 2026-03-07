# Yantr App Format

This document describes the app format currently used in Yantr.

It is intentionally based on two things only:

- What the Yantr codebase actually reads and expects.
- What existing apps in `apps/` already do today.

It does not introduce extra policy beyond the current repo.

## What Yantr Actually Loads

Each app lives in its own folder under `apps/`.

Current structure:

- `apps/<app-id>/compose.yml`
- `apps/<app-id>/info.json`
- optional helper files referenced by the compose file, such as `entrypoint.sh`

Examples already in the repo:

- `pihole` includes a local `entrypoint.sh`
- `open-webui` uses an external network and a dependency on `ollama`
- `tailscale` uses host networking and extra capabilities

## App ID

The folder name acts as the app ID.

Observed convention in the repo:

- app IDs are lowercase
- app IDs use hyphens when needed
- the same ID is reused in labels and dependency declarations

Examples:

- `airsonic`
- `open-webui`
- `uptime-kuma`

## Compose File

Yantr reads `compose.yml` directly and deploys it with standard Docker Compose.

What matters in practice:

- the file must be valid YAML
- the app must be deployable with `docker compose`
- the first service is treated as the primary service by the website build when inferring the main image

### Labels Used by Yantr

Current apps consistently use these labels:

```yaml
labels:
  yantr.app: "my-app"
  yantr.service: "Web UI"
```

What they are used for:

- `yantr.app` identifies which app a container belongs to
- `yantr.service` provides a human-readable service name in the UI

Observed convention:

- `yantr.app` matches the folder name
- `yantr.service` is a short display label such as `Web UI`, `DNS Sinkhole`, `VPN`, or `Container UI`

## Environment Variables

Yantr parses environment variables from `compose.yml` to build deployment inputs.

The backend currently recognizes both of these patterns:

Map syntax:

```yaml
environment:
  WEBPASSWORD: ${WEBPASSWORD:-yantr}
  TZ: ${TZ:-UTC}
```

List syntax with `${...}` values:

```yaml
environment:
  - ADMIN_USER=${ADMIN_USER:-admin}
  - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme}
```

What current apps mostly do:

- most apps use map syntax
- user-editable values often use `${VAR:-default}`
- some apps also include fixed internal values such as `NODE_ENV: production`
- some apps require values with no default, for example `${TUNNEL_TOKEN}` or `${TAILSCALE_AUTH_KEY}`

## Ports

Yantr reads published ports from Compose and combines them with labels from `info.json`.

Current patterns in the repo:

- container-only published ports such as `- "4040"`
- fixed host mappings such as `- "53:53/udp"`
- mixed TCP and UDP exposure when the app needs both

Examples:

- `airsonic` publishes `4040`
- `pihole` publishes port `53` on both TCP and UDP and a web port
- `deluge` publishes both web and torrent ports

In practice, `info.json` is used to describe the ports that matter to the user.

## Volumes

Yantr apps commonly persist data with named volumes, but the exact style is not fully uniform across the repo.

Persistent Docker app data should be stored in Docker volumes, not in host-path bind mounts.

Use host bind mounts only for narrow special cases such as:

- local helper files that ship with the app folder
- Docker socket access
- device access such as `/dev/net/tun`

Do not use host filesystem paths for normal application state such as databases, uploads, media libraries, or config that should survive container replacement.

Current patterns already present:

- top-level named volumes without an explicit `name:` field
- top-level named volumes with an explicit `name:` field
- local helper-file bind mounts inside the app folder
- host bind mounts for special cases such as `/var/run/docker.sock` or `/dev/net/tun`

Examples:

- `airsonic` uses named volumes without explicit `name:` keys
- `deluge` uses named volumes with explicit `name:` keys
- `pihole` bind-mounts a local helper script from its app folder
- `portainer` mounts `/var/run/docker.sock`
- `tailscale` mounts `/dev/net/tun`

Because the repo contains both volume declaration styles today, this document does not treat one of them as the only valid format.

## Networks

Most apps rely on the default Compose network, but there are already exceptions.

Existing patterns:

- default per-project Compose networking
- external networks when an app must connect to another stack
- host networking for networking-focused tools

Examples:

- `open-webui` joins the external `ollama_network`
- `tailscale` uses `network_mode: host`

When an app depends on another Yantr app through networking, the current catalog also reflects that in `info.json` dependencies.

## Elevated Host Access

Some current apps require extra host access. This is already part of the repo and should be documented as such rather than treated as impossible.

Existing examples:

- Docker socket mounts in `portainer`, `dockge`, `watchtower`, `glances`, `moltis`, and optionally `homarr`
- host networking in `tailscale`
- extra capabilities in `tailscale`
- device bind mounts such as `/dev/net/tun`

The important point for contributors is not that these patterns are banned. It is that they should be intentional and clearly explained in `info.json` notes when the app depends on them.

## Dependencies

Yantr reads the `dependencies` array from `info.json` and uses it during deployment checks.

Current pattern:

- dependencies are app IDs
- most apps use an empty array
- some apps declare a real dependency when needed

Examples:

- `nextcloud` depends on `mariadb`
- `open-webui` depends on `ollama`

## info.json

`info.json` provides catalog metadata for both the app UI and the website build.

Fields that are commonly present across current apps:

- `name`
- `logo`
- `tags`
- `ports`
- `short_description`
- `description`
- `usecases`
- `website`
- `dependencies`
- `notes`

The codebase has fallbacks for some missing fields, but the current catalog format is built around these keys being present.

### `name`

Current apps use the human-readable product name.

Examples:

- `Airsonic`
- `Open WebUI`
- `Pi-hole`

### `logo`

Current apps overwhelmingly use IPFS CIDs for logos.

That is the format already established across the repo. Yantr then turns those CIDs into gateway URLs when building the catalog and website.

If you need to upload a logo and get a CID, the current workflow is:

```bash
curl -X POST -F "file=@your-logo-file.png" https://originless.besoeasy.com/upload
```

The response is a JSON object with a `cid` field. That `cid` value is what goes into `info.json` as the `logo` field.

### `tags`

Current apps use lowercase tags.

Common patterns:

- category-like tags such as `media`, `database`, `security`, `ai`, `network`
- product-shape tags such as `streaming`, `chatbot`, `docker`, `utility`

### `ports`

This array describes user-facing ports.

Current shape:

```json
"ports": [
  { "port": 8080, "protocol": "HTTP", "label": "Web UI" },
  { "port": 5432, "protocol": "TCP", "label": "Database" }
]
```

This data is used by the UI to label otherwise raw port mappings.

### `short_description`

Current apps use a short one-sentence summary suitable for cards and lists.

### `description`

Current apps use a longer paragraph describing what the app does and where it fits.

### `usecases`

Current apps usually include 3 or 4 concrete example use cases.

### `website`

Current apps point to one of these:

- official documentation
- project homepage
- upstream source repository

### `dependencies`

This is always an array in current apps, even when empty.

### `notes`

Current apps use notes for operational caveats.

Examples already present in the repo:

- default credential warnings
- required Docker socket access
- router or DNS setup reminders
- internal network or dependency notes

## Practical Submission Checklist

This checklist is based on the current repo and runtime behavior.

- add the app under `apps/<app-id>/`
- include both `compose.yml` and `info.json`
- make sure `compose.yml` is valid and deployable with `docker compose`
- set `yantr.app` and `yantr.service` labels on the service containers
- keep `yantr.app` aligned with the folder name
- include the catalog metadata fields already used by current apps
- if the app depends on another Yantr app, declare it in `dependencies`
- if the app needs unusual host access, explain it in `notes`
- if the app exposes ports users care about, describe them in `info.json`

## Minimal Example Based on Current Apps

```yaml
services:
  my-app:
    image: ghcr.io/example/my-app:1.0.0
    container_name: my-app
    labels:
      yantr.app: "my-app"
      yantr.service: "Web UI"
    environment:
      TZ: ${TZ:-UTC}
      ADMIN_USER: ${ADMIN_USER:-admin}
      ADMIN_PASSWORD: ${ADMIN_PASSWORD:-changeme}
    ports:
      - "8080"
    volumes:
      - my_app_data:/data
    restart: unless-stopped

volumes:
  my_app_data:
```

```json
{
  "name": "My App",
  "logo": "QmExampleCid",
  "tags": ["productivity", "notes", "self-hosted", "webapp"],
  "ports": [
    { "port": 8080, "protocol": "HTTP", "label": "Web UI" }
  ],
  "short_description": "Self-hosted note-taking app with a clean web interface and fast search.",
  "description": "My App is a self-hosted note-taking service for capturing, organizing, and searching personal or team knowledge. It offers a browser-based editor, tagging, and simple sharing features in a lightweight deployment.",
  "usecases": [
    "Capture personal notes and knowledge in a private workspace.",
    "Organize project documentation with tags and full-text search.",
    "Share lightweight internal notes with a small team."
  ],
  "website": "https://example.com/docs",
  "dependencies": [],
  "notes": [
    "Change the default admin password before exposing the app externally."
  ]
}
```
