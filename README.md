<img width="2072" height="1505" alt="yantr" src="https://github.com/user-attachments/assets/c8a4f811-294e-4db5-9eb8-ce38dcbf0a33" />


<div align="center">

# ⚡ Yantr

**A self-hosted app store that runs alongside your OS — not instead of it.**

[![Docker](https://img.shields.io/badge/Docker-ghcr.io-0db7ed?style=flat-square&logo=docker&logoColor=white)](https://github.com/besoeasy/yantr/pkgs/container/yantr)
[![License](https://img.shields.io/badge/License-PolyForm%20Noncommercial-blue?style=flat-square)](LICENSE)
[![Vue 3](https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vue.js&logoColor=white)](https://vuejs.org)
[![Website](https://img.shields.io/badge/Website-yantr.org-5c6bc0?style=flat-square)](https://yantr.org)

[Quick Start](#-quick-start) · [App Catalog](#-app-catalog) · [yantr.org](https://yantr.org) · [Report a Bug](https://github.com/besoeasy/yantr/issues)

</div>

---

## The Problem with Umbrel & Friends

Platforms like Umbrel want to **be your OS**. Dedicate a machine, surrender the environment, live by their rules.

**Yantr is an app, not an OS.** It runs as a single container on the machine you already have — laptop, Raspberry Pi, old PC, home server. Your host stays untouched. Your files stay put. Yantr manages Docker stacks alongside everything else.

```
┌──────────────────────────────────────────┐
│            Your Operating System         │
│                                          │
│  ┌──────────────┐   ┌─────────────────┐  │
│  │    Yantr     │   │   Your existing │  │
│  │  (container) │   │   apps & files  │  │
│  └──────┬───────┘   └─────────────────┘  │
│         │ Docker socket                  │
│  ┌──────▼────────────────────────────┐   │
│  │  Jellyfin · n8n · Nextcloud · …   │   │
│  │      (isolated Docker stacks)     │   │
│  └───────────────────────────────────┘   │
└──────────────────────────────────────────┘
```

---

## 🚀 Quick Start

> **Requires Docker** — [install guide](https://yantr.org/install.html)

```bash
docker run -d \
  --name yantr \
  --network host \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/docker/volumes:/var/lib/docker/volumes \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantr
```

Open **http://localhost:5252** and create your operator account on first visit.

> **macOS / Windows?** Host networking is Linux-only. See the [platform install guide](https://yantr.org/install.html).

---

## 📦 App Catalog

130+ curated apps, one-click deploy.

| Category | Apps |
|---|---|
| 🤖 AI & Agents | Open WebUI, Perplexica, Agent Zero, Kokoro |
| 🎬 Media | Jellyfin, Emby, Stremio, Swingmusic, Metube |
| 📥 Downloads | qBittorrent, Radarr, Sonarr, Jackett, Prowlarr |
| ☁️ Cloud & Files | Nextcloud, Seafile, Syncthing, MinIO |
| 🔐 Security | Vaultwarden, AdGuard Home, Pi-hole, WireGuard |
| 🛠️ Dev Tools | Supabase, PostgreSQL, MongoDB, n8n, Gitea |
| 📡 Networking | Tailscale, Cloudflare Tunnel, Glances |
| 📝 Productivity | Memos, Wiki.js, Linkwarden, FreshRSS |
| 💰 Finance | Actual Budget, Ghostfolio, Rotki |
| ⛓️ Bitcoin | Bitcoin Knots, Monerod, Cashu.me |

> Browse all in [`apps/`](apps/) · Add your own via [`apps/apps.md`](apps/apps.md)

---

## ✨ What Makes It Different

**Zero OS footprint** — Yantr never writes to your host. Every app is a Docker Compose stack with named volumes. Uninstall and nothing is left behind.

**Full environment control** — Edit env vars per deployment. No fixed presets, no platform lock-in. Deploy the same app twice for testing, family, or clients.

**No dependency hell** — Python, Node, CUDA, whatever — each app brings its own runtime inside its container. Nothing conflicts.

**Built-in networking** — Port conflict detection, Tailscale for private access, Cloudflare Tunnel for public exposure, Caddy reverse proxy with HTTPS.

**Volume browser** — Inspect and manage your app data directly from the browser.

**REST API** — Every action (install, update, remove, logs) is a JSON endpoint. Scriptable and CLI-friendly.

---

## 🏗️ How It Works

One container. No database. No external dependencies.

1. **Serves a Vue 3 UI** on port `5252`
2. **Exposes a Fastify API** that talks to Docker via socket
3. **Reads compose templates** from its built-in catalog
4. **Deploys isolated stacks** — each app is an independent Compose project

---

## 🤝 Contributing

To add an app, follow the format in [`apps/apps.md`](apps/apps.md), then validate:

```bash
node check.js
```

PRs welcome.

---

<div align="center">
  <sub>Vue 3 · Fastify · Docker · Tailwind CSS</sub><br/><br/>
  <a href="https://yantr.org">yantr.org</a> · <a href="https://github.com/besoeasy/yantr/issues">Issues</a> · <a href="apps/apps.md">App Format Guide</a>
</div>
