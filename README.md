<img width="2072" height="1505" alt="yantr" src="https://github.com/user-attachments/assets/c8a4f811-294e-4db5-9eb8-ce38dcbf0a33" />

<div align="center">

# ⚡ Yantr

**A self-hosted app store that never touches your host.**

[![Podman](https://img.shields.io/badge/Podman-ghcr.io-892ca0?style=flat-square&logo=podman&logoColor=white)](https://github.com/besoeasy/yantr/pkgs/container/yantr)
[![Rootless](https://img.shields.io/badge/Security-100%25%20Rootless-emerald?style=flat-square)](https://yantr.org)
[![Daemonless](https://img.shields.io/badge/Architecture-Daemonless-blueviolet?style=flat-square)](https://yantr.org)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-PolyForm%20Noncommercial-blue?style=flat-square)](LICENSE)
[![Vue 3](https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vue.js&logoColor=white)](https://vuejs.org)
[![Website](https://img.shields.io/badge/Website-yantr.org-5c6bc0?style=flat-square)](https://yantr.org)

[![Star History Rank](https://api.star-history.com/badge?repo=besoeasy/yantr&type=rank)](https://www.star-history.com/besoeasy/yantr)

[Why Podman?](#️-rootless-by-design-why-podman-over-docker) · [Quick Start](#-quick-start) · [App Catalog](#-app-catalog) · [Community](#-community) · [yantr.org](https://yantr.org) · [Report a Bug](https://github.com/besoeasy/yantr/issues)

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
| **Engine Architecture** | Monolithic background daemon (single point of failure) | **Daemonless** — fork/exec model (`conmon`); no daemon to crash and kill apps |
| **System Service Manager** | Proprietary daemon supervision | **Native systemd Quadlet** (`systemctl --user`, auto-starts on boot) |
| **Logging & Monitoring** | Custom Docker logging drivers | **Native `journalctl --user`** and Linux cgroups v2 resource accounting |
| **Host Firewall Impact** | Modifies root `iptables`/`nftables` directly (silently bypasses UFW) | **Zero host firewall tampering** — isolated user network (pasta/slirp4netns) |
| **Host Filesystem Footprint** | Pollutes root `/var/lib/docker` (requires `sudo` for maintenance) | **100% user space** (`~/.local/share/containers/`) with SELinux `:z` isolation |
| **Idle Memory Overhead** | Always-on background daemon consuming RAM & CPU 24/7 | **Zero daemon overhead** — only your running apps consume resources |
| **Docker Socket Safety** | Exposing `/var/run/docker.sock` gives complete host root takeover | **Safe user socket translation** — apps get Docker API without host root |

---

## 🚀 Quick Start

One-line installation on Linux (automatically configures rootless Podman and systemd Quadlet):

```bash
curl -fsSL https://cdn.jsdelivr.net/gh/besoeasy/yantr@latest/install.sh | bash
```

Resolves `@latest` to the newest tagged release via jsDelivr (never `main`). The script is also in the repo root as [`install.sh`](install.sh).

<div align="center">

[![Fedora](https://img.shields.io/badge/Fedora-dnf-51A2DA?style=flat-square&logo=fedora&logoColor=white)](https://fedoraproject.org)
[![Debian](https://img.shields.io/badge/Debian-apt-A81D33?style=flat-square&logo=debian&logoColor=white)](https://www.debian.org)
[![Ubuntu](https://img.shields.io/badge/Ubuntu-apt-E95420?style=flat-square&logo=ubuntu&logoColor=white)](https://ubuntu.com)
[![Arch](https://img.shields.io/badge/Arch-pacman-1793D1?style=flat-square&logo=archlinux&logoColor=white)](https://archlinux.org)
[![openSUSE](https://img.shields.io/badge/openSUSE-zypper-73BA25?style=flat-square&logo=opensuse&logoColor=white)](https://www.opensuse.org)

</div>

Detects your package manager (`dnf`, `apt`, `pacman`, `zypper`), sets up rootless Podman + systemd Quadlet. Any Linux with Podman + systemd works via the manual command below (Alpine excluded — no systemd Quadlet).

Image on GHCR ([besoeasy/yantr](https://github.com/besoeasy/yantr/pkgs/container/yantr)):

```bash
podman pull ghcr.io/besoeasy/yantr:latest
```

Multi-arch (`amd64`/`arm64`), cosign-signed. `:dev` tracks `main`.

> **Run as your normal user — never `sudo`.** Rootless-only by design; the installer refuses root.

Or run manually with rootless Podman:

```bash
podman run -d \
  --name yantr \
  --network host \
  --security-opt label=disable \
  -e HOST_PODMAN_SOCKET=${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock \
  -v ${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock:/run/podman/podman.sock \
  -v yantr_data:/data:z \
  -v /:/host:ro \
  -v /sys:/sys:ro \
  ghcr.io/besoeasy/yantr:latest
```

- `HOST_PODMAN_SOCKET` — host path to the Podman socket handed to apps mounting `/var/run/docker.sock` (Portainer, Homarr, Glances). Without it Yantr queries the API at runtime and socket-needing apps may fail to deploy.
- `/host` and `/sys` (read-only) — back the host dashboard's filesystem-usage and temperature tiles. CPU, memory, load, uptime, disk I/O and network come from `/proc` and need no mount. Skip the binds and those two tiles just report unavailable.

Open **http://localhost:5252** and create your operator account on first visit.

---


## 💬 Community

Issues and bug reports belong on [GitHub](https://github.com/besoeasy/yantr/issues) so they stay searchable and trackable — that is where they get fixed.

- **App requests** — vote and discuss what lands in the catalog next.
- **Troubleshooting** — paste logs, and open a report that already says which version ran and on what host.
- **Show and tell** — what you're self-hosting.

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
