# Yantra 🔧

A lightweight, minimal, and open-source alternative to Umbrel for self-hosting enthusiasts.

## Screenshots

<img width="2102" height="1310" alt="image" src="https://github.com/user-attachments/assets/2e86639a-c48a-47ab-b31e-1b968c7ff494" />

## Why Yantra?

We created Yantra as a **faster, lighter, and more transparent** alternative to Umbrel. Umbrel's heavy apps, forced proxy architecture, and slow update cycles made self-hosting unnecessarily painful. Yantra takes a different approach:

- **Direct access** - No forced reverse proxies slowing down your apps
- **Minimal footprint** - ~70MB RAM vs 500MB+ for alternatives
- **Fast deployments** - Deploy apps in seconds, not minutes
- **Open & transparent** - Simple codebase anyone can understand and modify
- **Docker-native** - Works with standard Docker Engine for reliable container management

## Features

- 🚀 One-click app deployment
- 📦 Container management with volume cleanup
- 🎨 Clean, dark-mode Vue.js interface
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
  ghcr.io/besoeasy/yantra:main
```
