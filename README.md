<div align="center">

# ⚙️ Yantra

**The self-hosted app store for your PC**

[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)](https://github.com/besoeasy/yantra/pkgs/container/yantra)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?style=for-the-badge&logo=vue.js&logoColor=white)](https://vuejs.org)
[![License](https://img.shields.io/badge/License-ISC-blue.svg?style=for-the-badge)](LICENSE)

Run 100+ powerful applications on-demand, without buying extra hardware or replacing your OS. Yantra is to self-hosted apps what Homebrew is to software packages—simple, elegant, and built for your daily driver.

[🚀 Quick Start](#-quick-start) • [✨ Features](#-features) • [🛠️ Apps](#%EF%B8%8F-app-catalog) • [🎥 Demo Video](https://media.primal.net/uploads2/4/88/71/4887131c91f8868696e1b899085481bd22b016f6a7f936fb3f76e746d63fc119.mp4) • [💬 Issues](https://github.com/besoeasy/yantra/issues)

</div>

---

## 🚀 Quick Start

Launch Yantra in seconds with a single Docker command and access it at http://localhost.

```bash
docker run -d \
  --name yantra \
  -p 80:5252 \
  -p 443:5252 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantra
```

**✅ That’s it!** Open your browser and navigate to http://localhost to start deploying apps.

### System Requirements

- **Docker** installed and running
- **Linux, macOS, or Windows** (with Docker Desktop)
- **Disk space** for app data (minimal baseline, grows with apps)
- **Network access** to Docker daemon

---

## ✨ At a Glance

| | |
|---|---|
| **Philosophy** | 📦 App Store (not an OS replacement) |
| **Hardware** | 💻 Use your existing PC/Laptop |
| **Installation** | 🚀 Single Docker command |
| **Commitment** | 🎯 Zero—install, use, delete |
| **Ecosystem** | 🔓 100% Docker Compose (no lock-in) |
| **Resource Usage** | ⚡ On-demand (nothing runs unless you start it) |

---

## ✨ Features

<table>
<tr>
<td width="33%" valign="top">

### ⚡ One-Click Deploys
Instant access to popular self-hosted apps with clean defaults and smart presets.

</td>
<td width="33%" valign="top">

### ⏱️ Temporary Installations
Set expiration times and auto-delete apps when you’re done.

</td>
<td width="33%" valign="top">

### 🔄 Multiple Instances
Run the same app multiple times with different configurations.

</td>
</tr>
<tr>
<td width="33%" valign="top">

### 📂 Direct Volume Access
Browse and manage app data files directly from the UI.

</td>
<td width="33%" valign="top">

### 🧹 Auto-Cleanup
Removes old unused Docker images (10+ days) to free disk space.

</td>
<td width="33%" valign="top">

### 🎨 Premium UI
Modern Vue.js interface that feels fast and polished.

</td>
</tr>
</table>

---

## 🧭 Why Yantra

Most homelab platforms treat your hardware like an **OS replacement**. Yantra is different: it’s a lightweight app store that runs **alongside** your existing system. Your laptop stays yours—Yantra simply gives you one-click access to powerful self-hosted apps when you need them.

### Feature Comparison

| Feature            | 🔧 Yantra                      | 🚫 Others (Umbrel/CasaOS)              |
| :----------------- | :----------------------------- | :------------------------------------- |
| **Philosophy**     | App Store (Runs _on_ your OS)  | Operating System (Taking over your OS) |
| **Hardware**       | Your existing Laptop/PC        | Dedicated Pi/Server                    |
| **Installation**   | Single Docker command          | Full OS installation required          |
| **Flexibility**    | Run alongside daily work       | Exclusive computer use                 |
| **Control**        | Full System Access             | Locked Ecosystem                       |
| **Resource Usage** | On-demand (start what you use) | Always-on services                     |
| **Isolation**      | Clean Docker Containers        | System-wide dependencies               |
| **Portability**    | Easy backup/migration          | Tied to specific hardware              |
| **Learning Curve** | Familiar Docker workflow       | New OS paradigm to learn               |

---

## 🛠️ App Catalog

Yantra ships with 100+ pre-configured apps. Here are popular picks by use case:

### 📥 Media & Downloads

- **[MeTube](apps/metube)** — YouTube/TikTok/Instagram downloader
- **[Transmission](apps/transmission)** — Lightweight torrent client
- **[qBittorrent](apps/qbittorrent)** — Feature-rich torrent client

### 📄 Document Tools

- **[Stirling PDF](apps/stirling-pdf)** — PDF merge, split, edit, and convert
- **[LibreOffice](apps/libreoffice)** — Office document editing _(coming soon)_

### 🔄 File Conversion

- **[ConvertX](apps/convertx)** — Unlimited file conversion (PNG, JPG, WebP, etc.)
- **[BenToPDF](apps/bentopdf)** — Convert various formats to PDF

### 🔐 Privacy & Security

- **[Tor Browser](apps/tor-browser)** — Browse the web anonymously
- **[SearXNG](apps/searxng)** — Privacy-respecting search engine

### 💾 File Sharing & Sync

- **[Syncthing](apps/syncthing)** — Continuous file synchronization
- **[SAMBA](apps/samba)** — Network file sharing
- **[Dufs](apps/dufs)** — Lightweight file server

### 📊 Monitoring & Status

- **[Glances](apps/glances)** — Real-time system resource monitoring
- **[Uptime Kuma](apps/uptime-kuma)** — Service uptime monitoring

### 🎵 Media & Entertainment

- **[Jellyfin](apps/jellyfin)** — Open-source media server
- **[Emby](apps/emby)** — Personal media library
- **[Airsonic](apps/airsonic)** — Music streaming server

### 🌐 Networking & Utilities

- **[Pi-hole](apps/pihole)** — Network-wide ad blocker
- **[Wireguard](apps/wireguard)** — VPN server
- **[Networking Toolbox](apps/networking-toolbox)** — Network testing utilities

**[👉 Browse all 100+ apps](apps/)**

---

## 💾 Volume Management & WebDAV

Yantra gives you **direct access** to your data with built-in WebDAV support.

1. **Browse Volumes**: Go to the **Volumes** tab and click **Browse** on any volume.
2. **WebDAV Access**: The browser uses `dufs`, which inherently supports WebDAV.
3. **Sync Data**: Use WebDAV to sync files between volumes or machines.

**Example: Syncing two volumes with rclone**

```bash
# Sync from Volume A (port 5001) to Volume B (port 5002)
rclone sync :webdav:http://localhost:5001 :webdav:http://localhost:5002 --webdav-vendor other
```

---

## 🔒 Remote Access with Tailscale

Want to access your Yantra apps from anywhere securely? **We recommend [Tailscale](https://tailscale.com)** — it creates a private network between your devices.

- ✅ Access Yantra from any device, anywhere
- ✅ No port forwarding or firewall configuration needed
- ✅ End-to-end encrypted
- ✅ Free tier available

---

## 🤝 Contributing

Have ideas for new apps or improvements? Contributions are welcome!

- **Add a new app**: Follow the [Apps Template Standards](apps/apps.md)
- **Report issues**: [GitHub Issues](https://github.com/besoeasy/yantra/issues)
- **Share feedback**: Help shape the future of Yantra

---

## 📝 License

Yantra is open-source and available under the [LICENSE](LICENSE) file.

---

<div align="center">

**Built with ❤️ by [besoeasy](https://github.com/besoeasy)**

</div>
