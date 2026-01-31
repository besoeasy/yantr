👉 [Watch Video](https://media.primal.net/uploads2/4/88/71/4887131c91f8868696e1b899085481bd22b016f6a7f936fb3f76e746d63fc119.mp4)

# Yantra 🔧

> **Turn your Laptop into a Private Homelab.**
> Yantra is a sleek, self-hosted app store that runs _alongside_ your OS. Take back control of your data without dedicated hardware.

---

## 🚀 Quick Start (Copy & Paste)

Launch Yantra in seconds with a single Docker command.

### Run Yantra

**Docker (recommended):**

```bash
docker run -d \
  --name yantra \
  -p 80:5252 \
  -p 443:5252 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantra
```

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

## 🛠️ Perfect for Your Workflow

Yantra is designed for the modern "Work PC" homelab. Spin up tools, use them, spin them down.

| Goal                  | Solution                                                                     |
| :-------------------- | :--------------------------------------------------------------------------- |
| **📥 Download Media** | **[MeTube](apps/metube)** - YouTube/Facebook downloader.                     |
| **📄 Edit Docs**      | **[Stirling PDF](apps/stirling-pdf)** - Merge, split, and edit PDFs locally. |
| **🔄 Convert Files**  | **[ConvertX](apps/convertx)** - Unlimited file conversion (PNG, JPG, etc).   |
| **🕵️ Privacy**        | **[Tor Browser](apps/tor-browser)** & **[SearXNG](apps/searxng)**.           |
| **⚡ Share**          | **[SAMBA](apps/samba)** or **[Dufs](apps/dufs)** for instant P2P sharing.    |

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

## 🔒 Pro Tip: Go Remote with Tailscale

Want to access your apps from anywhere?

**We recommend [Tailscale](https://tailscale.com)**. It creates a secure private network for your devices.

- ✅ Access Yantra from any device, anywhere.
- ✅ No port forwarding required.
- ✅ End-to-end encrypted.

Simply install Tailscale on your server and devices to create your own secure private cloud.
