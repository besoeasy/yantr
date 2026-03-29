<div align="center">

# Yantr

**Turn the machine you already own into a powerful, isolated homelab.**

[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)](https://github.com/besoeasy/yantr/pkgs/container/yantr)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?style=for-the-badge&logo=vue.js&logoColor=white)](https://vuejs.org)
[![License](https://img.shields.io/badge/License-ISC-blue.svg?style=for-the-badge)](LICENSE)

</div>

---

## Why Yantr?

Most homelab platforms (like Umbrel, CasaOS, or Proxmox) demand total hardware control. They want to be the Operating System, requiring you to dedicate and wipe a machine just to run a few network apps.

**Yantr flips the model.**

It is a lightweight, self-hosted App Store that runs _alongside_ your existing OS. Whether you are using a daily-driver MacBook, a gaming PC, or an existing Linux server, Yantr lets you spin up powerful software instantly without taking over your machine.

- **Zero OS Footprint:** Yantr never touches your host filesystem. Everything runs exclusively in isolated Docker containers and virtual volumes.
- **No Dependency Hell:** Run complex AI models, media servers, and databases simultaneously. Because everything is isolated, you will never face a Python, Node, or CUDA version conflict.
- **Install, Use, Destroy:** Spin up an app, use it, and delete it. No leftover registry keys, background services, or hidden configuration files scattered across your system.

<br/>

## Quick Start

Launch Yantr in seconds with a single command. Requires Docker.

```bash
docker run -d \
  --name yantr \
  --network host \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/docker/volumes:/var/lib/docker/volumes \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantr
```

Navigate to `http://localhost:5252` in your browser. You are ready to go.

<br/>

## Core Features

| Area | Feature | What it does |
| --- | --- | --- |
| Platform | Zero OS Footprint | Runs alongside your existing OS in Docker without taking over the machine. |
| Platform | No Lock-In | Keeps everything in standard Docker Compose and Docker volumes so you retain full control outside Yantr. |
| Deployment | One-Command Install | Starts Yantr itself with a single `docker run` command. |
| Deployment | App Catalog | Ships curated app templates across AI, media, finance, networking, databases, and developer tools. |
| Deployment | One-Click App Deploys | Launches catalog apps instantly without manually wiring compose files and dependencies. |
| Deployment | Latest Upstream Versions | Tracks current upstream app versions through maintained catalog templates. |
| Interface | Web UI | Provides a fast Vue-based UI with real-time status, dense operations, and dark mode support. |
| Interface | REST API | Exposes installs, removals, logs, backups, and status checks through JSON endpoints. |
| Interface | CLI / Scriptable Automation | Works with `curl`, shell scripts, cron, and headless automation through the API. |
| Networking | Docker-Native Port Management | Supports Docker-style mappings like `8080:80` or `80`, including opening extra ports after deployment. |
| Networking | Port Conflict Detection | Tracks allocated host ports to prevent collisions between deployed apps. |
| Networking | Internal Service Routing | Manages Docker network routing and service-to-service communication between stack services. |
| Remote Access | Tailscale Integration | Makes private operator access possible without traditional port forwarding. |
| Remote Access | Cloudflare Tunnel Integration | Publishes services publicly without exposing inbound ports on your router. |
| Remote Access | Caddy Auth Proxy | Adds an authenticated front door to internal apps with a one-click reverse proxy pattern. |
| Storage | Volume Manager | Keeps app data isolated in Docker volumes and avoids orphaned state on install or removal. |
| Storage | Direct Volume Access | Lets you browse and manage persistent data from the browser via built-in volume tooling. |
| Backup | Restic Backups | Supports encrypted scheduled backups and point-in-time restore to local, S3-compatible, or B2 storage. |
| Lifecycle | Temporary Installs | Allows ephemeral apps with expiration timers and automatic cleanup. |
| Lifecycle | Clean Removal | Removes apps without leaving hidden background services, host dependencies, or stray data behind. |
| Linux Environments | Debian and Alpine Boxes | Deploys ready-to-use Linux environments with SSH access as disposable or long-lived utility boxes. |
| Linux Environments | Post-Deploy Port Publishing | Lets Debian/Alpine boxes expose more services later using the same Docker syntax users already know. |

<br/>

---

<div align="center">
  <br/>
  <a href="https://github.com/besoeasy/yantr/issues">Report an Issue</a> • <a href="apps/apps.md">App Format Guide</a>
  <br/>
</div>
