<div align="center">

<h1>⚡ Yantr</h1>

<p><strong>Your self-hosted app store — runs alongside your existing OS, no hardware takeover required.</strong></p>

<p>
  <a href="https://github.com/besoeasy/yantr/pkgs/container/yantr"><img src="https://img.shields.io/badge/Docker-ghcr.io-0db7ed?style=flat-square&logo=docker&logoColor=white" alt="Docker"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-ISC-blue?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vue.js&logoColor=white" alt="Vue 3">
  <img src="https://img.shields.io/badge/Fastify-5-000000?style=flat-square&logo=fastify&logoColor=white" alt="Fastify">
  <a href="https://yantr.org"><img src="https://img.shields.io/badge/Website-yantr.org-5c6bc0?style=flat-square" alt="Website"></a>
</p>

<p>
  <a href="#-quick-start">Quick Start</a> ·
  <a href="#-app-catalog">App Catalog</a> ·
  <a href="#-features">Features</a> ·
  <a href="https://yantr.org">Website</a> ·
  <a href="https://github.com/besoeasy/yantr/issues">Report a Bug</a>
</p>

</div>

---

## Why Yantr Over Umbrel?

Umbrel and similar homelab platforms want to **be your operating system**. You have to dedicate a machine, hand over the environment, and work around their platform rules when you want something custom.

**Yantr takes a different approach.**

Yantr is **an app, not an OS**. It runs as a lightweight container on the machine you already have: daily-driver laptop, Raspberry Pi, old PC, new workstation, home server, or anything else with Docker. Your host keeps running normally, your existing files and services stay in place, and Yantr manages apps alongside them instead of taking over the whole machine.

| What you need | Yantr approach |
|---|---|
| Latest app versions | Pull current upstream images and update apps when you choose |
| No OS takeover | Install Yantr as a container on your existing OS |
| Custom configuration | Edit environment variables per deployment instead of accepting fixed presets |
| Multiple instances | Deploy the same app more than once for testing, family, clients, or separate workspaces |
| Update control | Self-update Yantr, check for app updates, and update apps on demand |
| System isolation | Keep using your server, laptop, or old PC normally while apps run in Docker stacks |
| Lightweight runtime | No dedicated appliance layer, no full platform OS, just a small app manager |
| Runs almost anywhere | Works wherever you already have an OS and Docker: laptop, Raspberry Pi, old PC, new PC, or server |

Install it in 30 seconds, deploy 130+ apps from the catalog, customize the environment when needed, and remove it just as cleanly when you're done. Your OS stays exactly as you left it.

```
┌─────────────────────────────────────────┐
│           Your Operating System         │
│                                         │
│  ┌─────────────┐   ┌─────────────────┐  │
│  │    Yantr    │   │  Your existing  │  │
│  │  container  │   │   apps & files  │  │
│  └──────┬──────┘   └─────────────────┘  │
│         │ Docker API                    │
│  ┌──────▼──────────────────────────┐    │
│  │  Jellyfin · n8n · Nextcloud ...  │   │
│  │     (isolated Docker stacks)     │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

---

## 🚀 Quick Start

> **Requires:** Docker — [install guide](https://yantr.org/install-docker.html)

```bash
docker run -d \
  --name yantr \
  -p 5252:5252 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantr
```

Open **http://localhost:5252** and create the local operator account on first visit.

Authentication is stateless by default: if you do not set a public key env var, Yantr keeps the configured operator key only in memory and clears it on restart or self-update. To pin one operator identity across restarts, pass a daku public key such as `-e dakupublickey=<66-char-hex-public-key>`.

---

## 📦 App Catalog

130+ curated apps, all ready to deploy in one click.

| Category | Apps |
|---|---|
| 🤖 AI & Agents | Open WebUI, Perplexica, Agent Zero, Hermes Agent, Kokoro |
| 🎬 Media | Jellyfin, Emby, Airsonic, Fireshare, Metube, Stremio, Swingmusic |
| 📥 Downloads | qBittorrent, Transmission, Deluge, Radarr, Sonarr, Jackett, Prowlarr |
| 💰 Finance | Actual Budget, Maybe, Ghostfolio, Rotki, Cryptofolio, Monetr |
| ☁️ Cloud & Files | Nextcloud, Seafile, Syncthing, MinIO, Droppy, Samba |
| 🔐 Privacy & Security | Vaultwarden, AdGuard Home, Pi-hole, WireGuard, Tor Browser |
| 🛠️ Dev Tools | Supabase, PostgreSQL, MariaDB, MongoDB, n8n, Dockge, Gitea Mirror |
| 📡 Networking | Tailscale, Cloudflared, Glances, Smokeping, Snort, Librespeed |
| 📝 Productivity | Memos, Jotty, Wiki.js, Linkwarden, Miniflux, FreshRSS |
| 💬 Comms | Element (Matrix), Synapse, Nostr relay (strfry), Gotify |
| ⛓️ Bitcoin | Bitcoin Knots, Monerod, Specter Desktop, Cashu.me |
| 🐧 Linux Boxes | Debian + SSH, Alpine + SSH (disposable or long-lived) |

> Browse the full list in [apps/](apps/) · Contribute a new app via [apps/apps.md](apps/apps.md)

---

## ✨ Features

### Zero OS Footprint
Yantr never touches your host filesystem. Every app lives in its own Docker stack with isolated volumes. Remove an app and nothing is left behind — no registry keys, no background services, no scattered config files.

### One-Click Deploys
Pick an app from the catalog, click deploy. Yantr handles compose file wiring, port allocation, and volume setup automatically.

### No Dependency Hell
AI models, media servers, and databases run side-by-side without version conflicts. Python, Node, CUDA — each app brings its own runtime inside its container.

### Full Lifecycle Control

| Action | What happens |
|---|---|
| **Install** | Pulls images, starts the stack, registers ports |
| **Update** | Pulls latest images, recreates containers |
| **Remove** | Stops stack, removes containers, cleans volumes |

### Networking & Remote Access

- **Port conflict detection** — Yantr tracks every allocated host port to prevent collisions
- **Tailscale** — private operator access without port forwarding
- **Cloudflare Tunnel** — publish services publicly without opening your router
- **Caddy reverse proxy** — add auth and HTTPS in front of any internal app

### Storage
- Browse and manage volume data directly from the browser

### REST API + CLI-Friendly

Every operation — install, remove, logs, status — is available as a JSON endpoint. After first-run setup, the API requires a signed bearer token from the configured operator identity.

---

## 🏗️ How It Works

Yantr is a single Docker container that:

1. **Serves a Vue 3 web UI** on port `5252`
2. **Exposes a Fastify REST API** that manages Docker via the socket
3. **Reads app templates** from its built-in catalog (Docker Compose + metadata)
4. **Deploys isolated stacks** — each app is an independent Docker Compose project

No databases, no external dependencies, no agents on the host.

---

## 🤝 Contributing

PRs are welcome. To add an app to the catalog, follow the format in [apps/apps.md](apps/apps.md). After editing any app in `apps/`, run:

```bash
node check.js
```

Fix any format issues it reports before opening a PR.

---

<div align="center">
  <sub>Built with Vue 3 · Fastify · Docker · Tailwind CSS</sub>
  <br/>
  <br/>
  <a href="https://yantr.org">yantr.org</a> · <a href="https://github.com/besoeasy/yantr/issues">Issues</a> · <a href="apps/apps.md">App Format Guide</a>
</div>
