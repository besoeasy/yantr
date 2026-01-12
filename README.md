# Yantra 🔧

A lightweight Docker App Store that runs alongside your system - not as a full operating system replacement.

## Screenshots

<img width="2996" height="2104" src="https://github.com/user-attachments/assets/f0eba908-7da0-4e12-8c03-ed7f3a088e9b" />
<img width="3072" height="2104" src="https://github.com/user-attachments/assets/8da85686-8018-4131-b063-69507111b531" />


## Why Yantra?

**Yantra is fundamentally different from UmbrelOS, CasaOS, NirvatiOS, and Start9.**

These platforms force you to run a complete operating system just to get an app store. When you install Umbrel on your Raspberry Pi, you're wasting its potential - your powerful computer becomes locked into their system, running outdated apps with no customization.

### Yantra's Philosophy: Your System, Your Rules

**Yantra is a Docker App Store, not an operating system.** It runs alongside your existing system without taking over.

**Key Advantages:**

- **🖥️ Your system stays yours** - Keep using your Raspberry Pi, server, or computer however you want
- **🔒 Complete isolation** - Apps never touch your system files, everything stays in Docker containers
- **⚡ No bloat** - ~70MB RAM vs 500MB+ for full OS alternatives
- **🎛️ Full customization** - Edit environment variables, ports, and volumes directly (unlike Umbrel)
- **🚀 Up-to-date apps** - No waiting for slow platform updates to get the latest versions
- **🔧 Simple & transparent** - Minimal codebase anyone can understand and modify
- **📍 Direct access** - No forced reverse proxies slowing down your services

### The Problem with Full OS Platforms

When you install UmbrelOS, CasaOS, or similar platforms:
- ❌ Your entire system becomes dedicated to their platform
- ❌ You're stuck with their app versions and update schedule
- ❌ Can't customize environment variables or configurations
- ❌ Apps are outdated by months
- ❌ Heavy resource usage even when idle
- ❌ Limited flexibility and control

### With Yantra

- ✅ Install apps in seconds while keeping your system free
- ✅ Run Yantra alongside your projects, dev environment, or daily tasks
- ✅ Get the app store benefits without sacrificing your computer
- ✅ Full Docker compatibility - use standard tools and commands
- ✅ Customize everything - all compose files are accessible and editable

## Perfect for Your Work PC

**We recommend installing Yantra on your work computer or laptop.** Install apps only when you need them, use them, then remove them - keeping your system clean and focused.

### Real-World Examples

**1. Download videos from YouTube or Facebook?**  
Install [MeTube](apps/metube), download your video, remove it.

**2. Need to edit PDF files?**  
Install [Stirling PDF](apps/stirling-pdf), edit your documents, remove it.

**3. Convert images (PNG to JPG, etc.)?**  
Install [ConvertX](apps/convertx), convert your files, remove it.

**4. Need to download torrents?**  
Install [Transmission](apps/transmission), download your files, remove it.

**5. Quickly share files with colleagues?**  
Install [SAMBA](apps/samba) or [Dufs](apps/dufs), share files, remove it.

**6. Browse privately with TOR?**  
Install [TOR Browser](apps/tor), use it, remove it when done.

**7. Need a tracking-free search engine?**  
Install [SearXNG](apps/searxng), search privately, remove it.

**Why this works:**  
Apps run in complete isolation - no leftovers, no system clutter, no permanent installations. Your work PC stays clean and you get powerful tools exactly when needed.

## Features

- 🚀 One-click app deployment
- 📦 Container management with volume cleanup
- 🎨 Clean Vue.js interface
- 🏪 Built-in app store with 6+ popular self-hosted apps
- � Docker-based (reliable, boot-persistent)
- 💾 Minimal resource usage (~70MB RAM)

## Run Yantra with Docker

```bash
docker run -d \
  --name yantra \
  -p 80:5252 \
  -p 443:5252 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantra
```
### 🔒 Recommended: Tailscale Integration

**We highly recommend installing [Tailscale](https://tailscale.com)** to turn Yantra into your own private cloud. Tailscale creates a secure, private network across all your devices, allowing you to:

- **Access Yantra securely** from anywhere without exposing ports to the internet
- **No port forwarding** or complex firewall rules needed
- **End-to-end encryption** for all connections
- **Zero-trust networking** with device authentication
- **Access all your self-hosted apps** as if they were on your local network

Simply install Tailscale on your server and devices, and you'll have secure remote access to your Yantra instance and all deployed apps without the security risks of exposing services publicly.
