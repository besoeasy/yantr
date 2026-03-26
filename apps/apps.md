# Yantr App Format

Each app lives in `apps/<app-id>/` with:

- `compose.yml` - Docker Compose file
- `info.json` - catalog metadata

## Hard Rules

- **Never touch the host filesystem** - all persistent data must use Docker volumes
- **Always use Docker volumes** for: databases, config, uploads, media, logs, caches
- **Never use host bind mounts** except for: `/var/run/docker.sock`, `/dev/net/tun`, local helper files in the app folder
- **Always run `check.js`** after adding, editing, or fixing an app to ensure it matches repo standards

## compose.yml

Required:

- valid YAML, deployable with `docker compose`
- labels on service containers:
  ```yaml
  labels:
    yantr.app: "my-app"
    yantr.service: "Web UI"
  ```

Environment variables (both supported):

```yaml
environment:
  TZ: ${TZ:-UTC}
  # or list syntax:
  - ADMIN_USER=${ADMIN_USER:-admin}
```

Ports: publish as needed (`"8080"` or `"53:53/udp"`)

Images: prefer `:latest` tags over pinned version tags when defining app images

Volumes: always use named Docker volumes for persistent data. Host bind mounts only for:

- local helper files in the app folder
- `/var/run/docker.sock`
- `/dev/net/tun` and similar devices

Networks: default, external (`network: external`), or host (`network_mode: host`)

## info.json

Required fields:

- `name` - human-readable product name
- `logo` - IPFS CID (upload to `https://originless.besoeasy.com/upload`)
- `tags` - at least 6 lowercase tags, e.g. `["media", "docker", "ai", "self-hosted", "webui", "tools"]`
- `ports` - user-facing ports:
  ```json
  "ports": [{ "port": 8080, "protocol": "HTTP", "label": "Web UI" }]
  ```
- `short_description` - 50-100 characters
- `description` - 200-300 characters
- `usecases` - at least 2 entries
- `website` - `https://` docs/homepage/repo URL
- `dependencies` - array of app IDs (can be empty)
- `notes` - optional array of operational caveats
- `customapp` - optional boolean, set to `true` for apps custom-built by the Yantr team (e.g. apps with a `Dockerfile` in the folder). Custom apps show a "Built by Yantr" badge in the UI and have the auto-update button disabled since they use a locally-built image that watchtower cannot update.

## Checklist

- [ ] `apps/<app-id>/compose.yml` + `info.json`
- [ ] `yantr.app` = folder name
- [ ] `yantr.service` = display label
- [ ] valid, deployable compose file
- [ ] declare dependencies in `info.json`
- [ ] document unusual host access in `notes`
- [ ] describe user-facing ports in `info.json`

## Minimal Example

```yaml
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
  "logo": "IPFS CID",
  "tags": ["productivity", "self-hosted", "webapp"],
  "ports": [{ "port": 8080, "protocol": "HTTP", "label": "Web UI" }],
  "short_description": "Self-hosted note-taking app.",
  "description": "A self-hosted note-taking service.",
  "usecases": ["Capture notes.", "Organize docs.", "Share with team."],
  "website": "https://example.com/docs",
  "dependencies": [],
  "notes": ["Change default admin password."]
}
```
