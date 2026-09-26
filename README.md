<img width="2072" height="1505" alt="yantr" src="https://github.com/user-attachments/assets/c8a4f811-294e-4db5-9eb8-ce38dcbf0a33" />


<div align="center">

# ⚡ Yantr

**A self-hosted app store that runs alongside your OS — not instead of it.**

[![Podman](https://img.shields.io/badge/Podman-ghcr.io-892ca0?style=flat-square&logo=podman&logoColor=white)](https://github.com/besoeasy/yantr/pkgs/container/yantr)
[![Rootless](https://img.shields.io/badge/Security-100%25%20Rootless-emerald?style=flat-square)](https://yantr.org)
[![Daemonless](https://img.shields.io/badge/Architecture-Daemonless-blueviolet?style=flat-square)](https://yantr.org)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-PolyForm%20Noncommercial-blue?style=flat-square)](LICENSE)
[![Vue 3](https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vue.js&logoColor=white)](https://vuejs.org)
[![Website](https://img.shields.io/badge/Website-yantr.org-5c6bc0?style=flat-square)](https://yantr.org)

[Why Podman?](#️-rootless-by-design-why-podman-over-docker) · [Quick Start](#-quick-start) · [App Catalog](#-app-catalog) · [yantr.org](https://yantr.org) · [Report a Bug](https://github.com/besoeasy/yantr/issues)

</div>

---

## The Problem with Umbrel & Friends

Platforms like Umbrel want to **be your OS**. Dedicate a machine, surrender the environment, live by their rules.

**Yantr is an app, not an OS.** It runs as a single rootless container on the machine you already have — laptop, Raspberry Pi, old PC, home server. Your host stays untouched. Your files stay put. Yantr manages rootless Podman stacks alongside everything else.

```
┌──────────────────────────────────────────┐
│            Your Operating System         │
│                                          │
│  ┌──────────────┐   ┌─────────────────┐  │
│  │    Yantr     │   │   Your existing │  │
│  │  (container) │   │   apps & files  │  │
│  └──────┬───────┘   └─────────────────┘  │
│         │ Podman user socket             │
│  ┌──────▼────────────────────────────┐   │
│  │  Jellyfin · n8n · Nextcloud · …   │   │
│  │      (isolated Podman stacks)     │   │
│  └───────────────────────────────────┘   │
└──────────────────────────────────────────┘
```

---

## 🛡️ Rootless by Design: Why Podman Over Docker

Most homelab platforms and container managers rely on Docker running as a root-level daemon. While popular, Docker's monolithic architecture introduces critical security vulnerabilities, host filesystem modifications, and single-point-of-failure risks.

**Yantr was built from day one to run 100% rootless on Podman.** Here is how they compare:

| Feature / Metric | Traditional Docker | Yantr + Rootless Podman |
|---|---|---|
| **Daemon Privilege** | Runs as `root` (`dockerd` background daemon) | **Rootless** — runs in user namespaces with an unprivileged UID |
| **Security Blast Radius** | `docker` group = passwordless root on host (`docker run -v /:/host`) | **True isolation** — container breakout cannot compromise host root |
| **Engine Architecture** | Monolithic background daemon (single point of failure) | **Daemonless** — fork/exec model (`conmon`); no daemon crash kills apps |
| **System Service Manager** | Proprietary daemon supervision | **Native systemd Quadlet** (`systemctl --user`, auto-starts on boot) |
| **Logging & Monitoring** | Custom Docker logging drivers | **Native `journalctl --user`** and Linux cgroups v2 resource accounting |
| **Host Firewall Impact** | Modifies root `iptables`/`nftables` directly (silently bypasses UFW) | **Zero host firewall tampering** — isolated user network (pasta/slirp4netns) |
| **Host Filesystem Footprint** | Pollutes root `/var/lib/docker` (requires `sudo` for maintenance) | **100% user space** (`~/.local/share/containers/`) with SELinux `:z` isolation |
| **Idle Memory Overhead** | Always-on background daemon consuming RAM & CPU 24/7 | **Zero daemon overhead** — only your running apps consume resources |
| **Docker Socket Safety** | Exposing `/var/run/docker.sock` gives complete host root takeover | **Safe user socket translation** — apps get Docker API without host root |

### Key Architectural Advantages

#### 1. True Rootless Isolation (Zero Host Root Privileges)
Under standard Docker, the container engine runs as `root`. Granting a user access to the Docker socket or `docker` group is practically identical to granting passwordless `sudo`. If any web app or container has a zero-day exploit or escape vulnerability, the attacker gains root control of the physical host.

Under rootless Podman, Yantr runs entirely inside Linux **user namespaces** (`subuid` and `subgid`). Even if an app runs as "root" (UID 0) inside its container, the Linux kernel maps that UID to an unprivileged user ID on the host. If an attacker breaches the container sandbox, they have zero host root permissions: they cannot read `/etc/shadow`, cannot overwrite host binaries, and cannot install system-wide rootkits.

#### 2. Daemonless Architecture (No Single Point of Failure)
Docker requires `dockerd` running continuously in the background. If the Docker daemon crashes, hangs, or restarts during a package update, all running containers can be killed or lose network connectivity. Furthermore, the idle daemon constantly consumes memory.

Podman is **daemonless**. It uses a direct fork/exec model managed by lightweight `conmon` (container monitor) processes. There is no central daemon to crash. If Yantr itself is stopped, restarted, or updated, all of your background applications (Nextcloud, Jellyfin, Vaultwarden, AI models) continue running uninterrupted.

#### 3. Native Linux systemd Integration (Quadlet)
Docker fights your operating system with its own restart policies and proprietary supervisors.

Yantr embraces the native Linux init system using **systemd Quadlet** (`.container` declarations in `~/.config/containers/systemd/`):
- **Auto-starts on boot** via standard Linux user lingering (`loginctl enable-linger`).
- **Native administration**: Control Yantr using standard `systemctl --user {status,restart,stop} yantr`.
- **Unified system logging**: Stream live container logs directly with `journalctl --user -u yantr -f`.
- **Native cgroups v2**: Clean per-service resource limits and process tracking managed directly by the kernel.

#### 4. Untouched Host Networking & Firewall Safety
Docker notoriously rewrites host `iptables` and `nftables` rules directly. On Linux distros like Ubuntu or Debian, Docker often punches holes right through UFW (Uncomplicated Firewall) without warning, accidentally exposing internal database ports or admin panels to the public internet.

Rootless Podman executes network routing strictly within an isolated user network namespace (via `pasta` or `slirp4netns`). It **never modifies host firewall rules**. Your host firewall configuration stays 100% intact and trustworthy.

#### 5. Transparent Docker Socket Emulation Without Root Risk
Many popular homelab tools (such as Glances, Beszel, Portainer, or Arcane) expect `/var/run/docker.sock`. On Docker, mounting this socket hands the container keys to the kingdom.

Yantr automatically detects and routes Docker socket requests to Podman's unprivileged user socket (`/run/user/$UID/podman/podman.sock`). The apps function with complete Docker compatibility, but without giving them root privileges on your host machine.

---

## 🚀 Quick Start

One-line installation on Linux (automatically configures rootless Podman and systemd Quadlet):

```bash
curl -fsSL https://yantr.org/install.sh | bash
```

Or run manually with rootless Podman:

```bash
podman run -d \
  --name yantr \
  --network host \
  --security-opt label=disable \
  -v ${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock:/run/podman/podman.sock \
  -v yantr_data:/data:z \
  ghcr.io/besoeasy/yantr:latest
```

Open **http://localhost:5252** and create your operator account on first visit.

---

## 📦 App Catalog

350+ curated apps, one-click deploy.

| Category | Apps |
|---|---|
| 🤖 AI & Agents | Open WebUI, Perplexica, Agent Zero, Kokoro |
| 🎬 Media | Jellyfin, Emby, Stremio, Swingmusic, Metube |
| 📥 Downloads | qBittorrent, Radarr, Sonarr, Jackett, Prowlarr |
| ☁️ Cloud & Files | Nextcloud, Seafile, Syncthing, MinIO |
| 🔐 Security | Vaultwarden, AdGuard Home, Pi-hole, WireGuard |
| 🛠️ Dev Tools | Supabase, PostgreSQL, MongoDB, n8n, Gitea |
| 📡 Networking | Cloudflare Tunnel, Glances |
| 📝 Productivity | Memos, Wiki.js, Linkwarden, FreshRSS |
| 💰 Finance | Actual Budget, Ghostfolio, Rotki |
| ⛓️ Bitcoin | Bitcoin Knots, Monerod, Cashu.me |

> Browse all in [`apps/`](apps/) · Add your own via the [authoring guide](AGENTS.md)

---

## ✨ What Makes It Different

**Zero OS footprint** — Yantr never writes to your host. Every app is a Podman Compose stack with named volumes. Uninstall and nothing is left behind.

**Automatic Docker socket translation** — Apps requesting `/var/run/docker.sock` (such as Glances, Beszel, Arcane, or Portainer) are automatically routed to your rootless Podman socket with zero configuration.

**Full environment control** — Edit env vars per deployment. No fixed presets, no platform lock-in. Deploy the same app twice for testing, family, or clients.

**No dependency hell** — Python, Node, CUDA, whatever — each app brings its own runtime inside its container. Nothing conflicts.

**Built-in networking** — Port conflict detection and Cloudflare Tunnel for public exposure.

**Volume browser** — Inspect and manage your app data directly from the browser.

**REST API** — Every action (install, update, remove, logs) is a JSON endpoint. Scriptable and CLI-friendly.

---

## Star History

<a href="https://www.star-history.com/?repos=besoeasy%2Fyantr&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=besoeasy/yantr&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=besoeasy/yantr&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=besoeasy/yantr&type=date&legend=top-left" />
 </picture>
</a>

---

## License

Yantr is source-available under the [PolyForm Noncommercial License 1.0.0](LICENSE).

- Personal, hobby, and homelab use: yes
- Study, change, and share the source under the same terms: yes
- Schools, charities, and other noncommercial orgs: yes
- Commercial use (sell it, host it as a paid product, run a business on it): no

The source stays public. The short version is at the top of [`LICENSE`](LICENSE).

---

<div align="center">
  <sub>Vue 3 · Go · Podman · Custom CSS</sub><br/><br/>
  <a href="https://yantr.org">yantr.org</a> · <a href="https://github.com/besoeasy/yantr/issues">Issues</a> · <a href="AGENTS.md">App Format Guide</a>
</div>
