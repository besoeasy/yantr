# Yantra 🔧

A lightweight, minimal, and open-source alternative to Umbrel for self-hosting enthusiasts.

<img width="1908" height="1207" alt="Screenshot From 2026-01-09 07-48-16" src="https://github.com/user-attachments/assets/4da76a09-f0b1-4fa6-8432-57fbed4fcb7e" />
<img width="1908" height="1207" alt="Screenshot From 2026-01-09 07-48-07" src="https://github.com/user-attachments/assets/ec9b382f-3c98-450e-ab9d-036d3d859908" />


## Why Yantra?

We created Yantra as a **faster, lighter, and more transparent** alternative to Umbrel. Umbrel's heavy apps, forced proxy architecture, and slow update cycles made self-hosting unnecessarily painful. Yantra takes a different approach:

- **Direct access** - No forced reverse proxies slowing down your apps
- **Minimal footprint** - ~100MB RAM vs 500MB+ for alternatives
- **Fast deployments** - Deploy apps in seconds, not minutes
- **Open & transparent** - Simple codebase anyone can understand and modify
- **Docker-native** - Works with standard Docker Engine for reliable container management

## Supported Operating Systems

Yantra runs on any Debian-based Linux distribution:

- **Debian** 11, 12+
- **Ubuntu** 20.04, 22.04, 24.04+
- **Raspberry Pi OS** (Bullseye, Bookworm)
- **Linux Mint** 20, 21+
- **Pop!_OS** 20.04+
- **Elementary OS** 6+
- **MX Linux** 21+
- Any other Debian derivative

## Features

- 🚀 One-click app deployment
- 📦 Container management with volume cleanup
- 🎨 Clean, dark-mode Vue.js interface
- 🏪 Built-in app store with 6+ popular self-hosted apps
- � Docker-based (reliable, boot-persistent)
- 💾 Minimal resource usage (~100MB RAM)

## Installation

### Prerequisites

**Install Docker** (if not already installed)

```bash
sudo apt update
sudo apt install -y docker.io

# Add your user to the docker group (optional, for non-root usage)
sudo usermod -aG docker $USER

# Enable Docker to start on boot
sudo systemctl enable docker
sudo systemctl start docker

# Verify Docker is running
sudo systemctl status docker
```


### Run Yantra with Docker

```bash
# Pull and run the pre-built image
docker run -d \
  --name yantra \
  -p 3000:3000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart unless-stopped \
  ghcr.io/besoeasy/yantra:main

# Note: The container includes Docker CLI for managing containers.
# On some systems, you may need to run: docker pull ghcr.io/besoeasy/yantra:main --platform linux/arm64
```
