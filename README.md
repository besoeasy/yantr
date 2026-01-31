👉 [Watch Video](https://media.primal.net/uploads2/4/88/71/4887131c91f8868696e1b899085481bd22b016f6a7f936fb3f76e746d63fc119.mp4) • [GitHub](https://github.com/besoeasy/yantra)

# ⚙️ Yantra

## Turn Your Laptop into a Private Homelab

**The self-hosted app store for your PC.** Run 100+ powerful applications on-demand, without buying extra hardware or replacing your OS.

Yantra is to self-hosted apps what Homebrew is to software packages—simple, elegant, and built for your daily driver.

---

### ✨ At a Glance

| | |
|---|---|
| **Philosophy** | 📦 App Store (not an OS replacement) |
| **Hardware** | 💻 Use your existing PC/Laptop |
| **Installation** | 🚀 Single Docker command |
| **Commitment** | 🎯 Zero—install, use, delete |
| **Ecosystem** | 🔓 100% Docker Compose (no lock-in) |
| **Resource Usage** | ⚡ On-demand (nothing runs unless you start it) |

---

## 🚀 Quick Start

Launch Yantra in seconds with a single Docker command and access it at `http://localhost`.

```bash
docker run -d \
  --name yantra \
  -p 80:5252 \
  -p 443:5252 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantra
```

**✅ That's it!** Yantra is now running. Open your browser and navigate to `http://localhost` to start deploying apps.

---

### System Requirements

- **Docker** installed and running
- **Linux, macOS, or Windows** (with Docker Desktop)
- **Disk space** for app data (minimal baseline, grows with apps)
- **Network access** to Docker daemon

---

## ✨ Why Yantra?

### The Problem with Traditional Homelabs

Most homelab platforms (Umbrel, CasaOS, etc.) treat your hardware like an **operating system replacement**. They demand your **entire computer**, require dedicated hardware, and lock you into their ecosystem. This approach made sense for Raspberry Pis gathering dust in drawers, but it's overkill for modern users who want self-hosted tools without buying extra hardware.

### Yantra's Philosophy: An App Store, Not an OS

Yantra takes a fundamentally different approach. It's a **lightweight app store** that runs _alongside_ your existing operating system, not instead of it. Think of it like Homebrew or apt, but for self-hosted applications with beautiful UI.

Your laptop remains yours. Your OS stays untouched. Yantra just gives you one-click access to powerful self-hosted apps whenever you need them.

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

### Why This Matters

**🎯 Zero Commitment**: Install apps temporarily, test them out, delete them when done. No permanent system changes.

**💻 Use Your Daily Driver**: No need to keep a separate machine running 24/7. Spin up services on your work laptop when needed, shut them down when you're done.

**🔓 No Lock-In**: Built on standard Docker Compose. Every app configuration is transparent and portable. Don't like Yantra? Your apps will work anywhere Docker runs.

**⚡ Instant Utility**: Need to convert a file? Download a video? Edit a PDF? Launch the app, do your task, close it. Homelab tools become everyday utilities instead of weekend projects.

**The Result?** You get a powerful server-grade environment without sacrificing your daily driver machine, buying extra hardware, or committing to a new operating system.

---

## 🛠️ Popular Apps & Use Cases

Yantra comes with 100+ pre-configured applications. Here are some popular ones organized by use case:

### 📥 Media & Downloads

- **[MeTube](apps/metube)** - YouTube/TikTok/Instagram downloader
- **[Transmission](apps/transmission)** - Lightweight torrent client
- **[qBittorrent](apps/qbittorrent)** - Feature-rich torrent client

### 📄 Document Tools

- **[Stirling PDF](apps/stirling-pdf)** - PDF merge, split, edit, and convert
- **[LibreOffice](apps/libreoffice)** - Office document editing _(coming soon)_

### 🔄 File Conversion

- **[ConvertX](apps/convertx)** - Unlimited file conversion (PNG, JPG, WebP, etc)
- **[BenToPDF](apps/bentopdf)** - Convert various formats to PDF

### 🔐 Privacy & Security

- **[Tor Browser](apps/tor-browser)** - Browse the web anonymously
- **[SearXNG](apps/searxng)** - Privacy-respecting search engine

### 💾 File Sharing & Sync

- **[Syncthing](apps/syncthing)** - Continuous file synchronization
- **[SAMBA](apps/samba)** - Network file sharing
- **[Dufs](apps/dufs)** - Lightweight file server

### 📊 Monitoring & Status

- **[Glances](apps/glances)** - Real-time system resource monitoring
- **[Uptime Kuma](apps/uptime-kuma)** - Service uptime monitoring

### 🎵 Media & Entertainment

- **[Jellyfin](apps/jellyfin)** - Open-source media server
- **[Emby](apps/emby)** - Personal media library
- **[Airsonic](apps/airsonic)** - Music streaming server

### 🌐 Networking & Utilities

- **[Pi-hole](apps/pihole)** - Network-wide ad blocker
- **[Wireguard](apps/wireguard)** - VPN server
- **[Networking Toolbox](apps/networking-toolbox)** - Network testing utilities

**[👉 Browse all 100+ apps](apps/)**

---

## 🌟 Key Features

- **⚡ One-Click Deploys**: Instant access to popular self-hosted apps.
- **⏱️ Temporary Installations**: Set expiration time, apps auto-delete when done.
- **🔄 Multiple Instances**: Run the same app multiple times with different configurations.
- **📂 Direct Volume Access**: Browse and manage app data files directly.
- **🧹 Auto-Cleanup**: Removes old unused Docker images (10+ days) to free disk space.
- **🎨 Clean Interface**: Modern Vue.js UI that feels premium.

---

## 💾 Volume Management & WebDAV

Yantra gives you **direct access** to your data with built-in WebDAV support.

1. **Browse Volumes**: Go to the **Volumes** tab and click **Browse** on any volume.
2. **WebDAV Access**: The browser uses `dufs`, which inherently supports WebDAV.
3. **Sync Data**: Use WebDAV to sync files between volumes or machines.

**Example: Syncing two volumes with `rclone`**

```bash
# Sync from Volume A (port 5001) to Volume B (port 5002)
rclone sync :webdav:http://localhost:5001 :webdav:http://localhost:5002 --webdav-vendor other
```

---

## 🔒 Remote Access with Tailscale

Want to access your Yantra apps from anywhere securely?

**We recommend [Tailscale](https://tailscale.com)** - it creates a secure private network between your devices.

- ✅ Access Yantra from any device, anywhere
- ✅ No port forwarding or firewall configuration needed
- ✅ End-to-end encrypted
- ✅ Free tier available

Simply install Tailscale on your server and devices to create your own secure private cloud.

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

## 🙋 Support & Community

- **Questions?** Check existing [GitHub Issues](https://github.com/besoeasy/yantra/issues)
- **Found a bug?** [Report it here](https://github.com/besoeasy/yantra/issues/new)
- **Want to contribute?** Fork, modify, and submit a pull request
