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

Credential and secret environment variables:

- Do not provide placeholder defaults for user-supplied credentials or secrets in `compose.yml` (for example, avoid `:-admin`, `:-password`, `:-changeme`)
- Require explicit values instead (for example, `ADMIN_PASSWORD: ${ADMIN_PASSWORD}`)
- If a variable is required from the user, add a matching `env_generators` entry in `info.json` so each field has its own generate button

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
- `notes` - optional array of operational caveats
- `customapp` - optional boolean, set to `true` for apps custom-built by the Yantr team (e.g. apps with a `Dockerfile` in the folder). Custom apps show a "Built by Yantr" badge in the UI and have the auto-update button disabled since they use a locally-built image that watchtower cannot update.
- `env_generators` - optional map of env var names to generation rules used by each env field in the deploy UI

Rules for required env vars:

- Any credential/secret variable that is required in `compose.yml` should have a corresponding key in `env_generators`
- Prefer secure defaults in generator rules (`alnum_symbols` and length 16+ for passwords, length 32+ for secrets/tokens)

`env_generators` example:

```json
"env_generators": {
  "BETTER_AUTH_SECRET": {
    "length": 48,
    "charset": "alnum_symbols",
    "regex": "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d).{32,}$"
  }
}
```

Supported rule fields:

- `length` (number): generated value length, clamped to 8..256
- `charset` (string): one of `alnum`, `hex`, `numeric`, `alpha`, `base64url`, `alnum_symbols`
- `characters` (string): optional custom character set (overrides `charset`)
- `regex` (string): optional JavaScript regex pattern that the generated value must satisfy

## Checklist

- [ ] `apps/<app-id>/compose.yml` + `info.json`
- [ ] `yantr.app` = folder name
- [ ] `yantr.service` = display label
- [ ] valid, deployable compose file
- [ ] required credential/secret env vars have no placeholder defaults in `compose.yml`
- [ ] required credential/secret env vars are covered by `env_generators` in `info.json`
- [ ] keep `info.json` self-contained; use `notes` to explain any manual app-to-app setup
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
  "notes": ["Change default admin password."]
}
```
