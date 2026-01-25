👉 [Watch Video](https://media.primal.net/uploads2/4/88/71/4887131c91f8868696e1b899085481bd22b016f6a7f936fb3f76e746d63fc119.mp4)


# Yantra 🔧

> **Turn your Laptop into a Private Homelab.**
> Yantra is a sleek, self-hosted app store that runs _alongside_ your OS. Take back control of your data without dedicated hardware.

---

## 🚀 Quick Start (Copy & Paste)

Get up and running in seconds. No complicated setup.

### 1) Install Docker (minimal)

> These commands install Docker Engine + Docker Compose plugin.

**Raspberry Pi OS / Debian / Ubuntu:**

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker
```

If you prefer the official step-by-step instructions instead of the convenience script:
- https://docs.docker.com/engine/install/

### 2) Run Yantra

**Using Docker:**

```bash
docker run -d \
  --name yantra \
  -p 80:5252 \
  -p 443:5252 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantra
```



_[New to Docker? Learn how to install it here.](https://chatgpt.com/?prompt=how%20to%20install%20docker%20on%20my%20system%20%3F%0A%0Ai%20want%20to%20install%20yantra%20https%3A%2F%2Fgithub.com%2Fbesoeasy%2Fyantra%0A%0Ahere%20is%20the%20docker%20command%2C%20in%20cli%20mode%0Adocker%20run%20-d%20%5C%0A%20%20--name%20yantra%20%5C%0A%20%20-p%2080%3A5252%20%5C%0A%20%20-p%20443%3A5252%20%5C%0A%20%20-v%20%2Fvar%2Frun%2Fdocker.sock%3A%2Fvar%2Frun%2Fdocker.sock%20%5C%0A%20%20--restart%20unless-stopped%20%5C%0A%20%20ghcr.io%2Fbesoeasy%2Fyantra%0A%0Aas%20ai%20agent%2C%20your%20aim%20is%20to%20help%20me%20install%20this%20software
)_

---

## ✨ Why Yantra?

Most homelab platforms (Umbrel, CasaOS) demand your **entire computer**. Yantra is different.

| Feature            | 🔧 Yantra                     | 🚫 Others (Umbrel/CasaOS)              |
| :----------------- | :---------------------------- | :------------------------------------- |
| **Philosophy**     | App Store (Runs _on_ your OS) | Operating System (Taking over your OS) |
| **Hardware**       | Your existing Laptop/PC       | Dedicated Pi/Server                    |
| **Installation**   | Single Docker command         | Full OS installation required          |
| **Flexibility**    | Run alongside daily work      | Exclusive computer use                 |
| **Control**        | Full System Access            | Locked Ecosystem                       |
| **Resource Usage** | On-demand (start what you use)| Always-on services                     |
| **Isolation**      | Clean Docker Containers       | System-wide dependencies               |
| **Portability**    | Easy backup/migration         | Tied to specific hardware              |
| **Learning Curve** | Familiar Docker workflow      | New OS paradigm to learn               |


**The Result?** You get a powerful server-grade environment without sacrificing your daily driver machine.

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
