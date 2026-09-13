#!/usr/bin/env bash
# ==============================================================================
#  Yantr Installer — Linux (Rootless Podman + Quadlet systemd)
#  https://yantr.org
# ==============================================================================

set -euo pipefail

BOLD="\033[1m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m"

log_info()  { echo -e "${BLUE}ℹ${NC}  $*"; }
log_step()  { echo -e "\n${BOLD}${GREEN}==>${NC} ${BOLD}$*${NC}"; }
log_warn()  { echo -e "${YELLOW}⚠${NC}  $*"; }
log_error() { echo -e "${RED}✖${NC}  $*"; }

echo -e "${BOLD}"
echo "    __  __            __       "
echo "    \ \/ /___ _____  / /______ "
echo "     \  / __ \`/ __ \/ __/ ___/ "
echo "     / / /_/ / / / / /_/ /     "
echo "    /_/\__,_/_/ /_/\__/_/      "
echo -e "${NC}"
echo -e "${BOLD}Yantr — Self-Hosted Homelab App Store${NC}"
echo -e "Native Linux · Rootless Podman · Quadlet systemd\n"

# 1. Linux check
if [ "$(uname -s)" != "Linux" ]; then
  log_error "Yantr is designed for Linux using rootless Podman and Quadlet."
  echo "        For macOS or Windows, run Yantr inside a Linux VM or WSL2 instance."
  exit 1
fi

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  fi
fi

# 2. Check / Install Podman
log_step "Checking Podman..."
if command -v podman >/dev/null 2>&1; then
  PODMAN_VER=$(podman --version | awk '{print $3}')
  log_info "Podman is already installed (version: ${PODMAN_VER})"
else
  log_info "Podman not detected. Installing Podman..."
  if command -v dnf >/dev/null 2>&1; then
    $SUDO dnf install -y podman
  elif command -v apt-get >/dev/null 2>&1; then
    $SUDO apt-get update -y
    $SUDO apt-get install -y podman
  elif command -v pacman >/dev/null 2>&1; then
    $SUDO pacman -Sy --noconfirm podman
  elif command -v zypper >/dev/null 2>&1; then
    $SUDO zypper in -y podman
  elif command -v apk >/dev/null 2>&1; then
    $SUDO apk add podman
  else
    log_error "Could not detect a supported package manager (dnf, apt, pacman, zypper, apk)."
    log_error "Please install Podman manually and run this script again."
    exit 1
  fi
  log_info "Podman installed successfully."
fi

# 3. Configure systemd & socket
IS_ROOT=0
if [ "$(id -u)" -eq 0 ]; then
  IS_ROOT=1
fi

if [ "$IS_ROOT" -eq 1 ]; then
  log_step "Configuring Podman system service (root mode)..."
  systemctl enable --now podman.socket
  QUADLET_DIR="/etc/containers/systemd"
  $SUDO mkdir -p "$QUADLET_DIR"
  QUADLET_FILE="$QUADLET_DIR/yantr.container"
  SYSTEMCTL="systemctl"
  SOCKET_BIND="/run/podman/podman.sock:/run/podman/podman.sock"
else
  log_step "Configuring Rootless Podman & systemd user service..."

  # Enable user lingering so services persist after logout and auto-start on boot
  if command -v loginctl >/dev/null 2>&1; then
    log_info "Enabling user lingering with loginctl..."
    loginctl enable-linger "$USER" 2>/dev/null || $SUDO loginctl enable-linger "$USER" 2>/dev/null || true
  fi

  # Enable and start rootless podman.socket
  log_info "Enabling user Podman socket..."
  systemctl --user enable --now podman.socket

  QUADLET_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/containers/systemd"
  mkdir -p "$QUADLET_DIR"
  QUADLET_FILE="$QUADLET_DIR/yantr.container"
  SYSTEMCTL="systemctl --user"
  SOCKET_BIND="%t/podman/podman.sock:/run/podman/podman.sock"
fi

# 4. Write Quadlet container specification
log_step "Writing Quadlet specification..."
log_info "Target: ${QUADLET_FILE}"

QUADLET_CONTENT="[Unit]
Description=Yantr
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/besoeasy/yantr:latest
ContainerName=yantr
Network=host
SecurityLabelDisable=true
Environment=CONTAINER_HOST=unix:///run/podman/podman.sock
Volume=yantr_data:/data:z
Volume=${SOCKET_BIND}
AutoUpdate=registry

[Service]
Restart=always

[Install]
WantedBy=default.target"

if [ "$IS_ROOT" -eq 1 ]; then
  echo "$QUADLET_CONTENT" | $SUDO tee "$QUADLET_FILE" > /dev/null
else
  echo "$QUADLET_CONTENT" > "$QUADLET_FILE"
fi

# 5. Reload systemd and start Yantr service
log_step "Activating Yantr systemd service..."
$SYSTEMCTL daemon-reload
$SYSTEMCTL restart yantr.service

# Also enable podman auto-update timer if available
$SYSTEMCTL enable --now podman-auto-update.timer 2>/dev/null || true

# 6. Verify health
log_step "Verifying Yantr status..."
sleep 2

if $SYSTEMCTL is-active --quiet yantr.service; then
  echo -e "\n${BOLD}${GREEN}✔ Yantr is up and running!${NC}\n"
else
  log_warn "Service was started, but may still be pulling image or initializing."
  echo "Check status with: ${SYSTEMCTL} status yantr"
fi

echo -e "Access the Web UI:  ${BOLD}${GREEN}http://localhost:5252${NC}"
echo -e "\n${BOLD}Useful Management Commands:${NC}"
echo -e "  • Status:   ${SYSTEMCTL} status yantr"
echo -e "  • Logs:     journalctl $([ "$IS_ROOT" -eq 0 ] && echo "--user ") -u yantr -f"
echo -e "  • Restart:  ${SYSTEMCTL} restart yantr"
echo -e "  • Stop:     ${SYSTEMCTL} stop yantr"
echo -e "  • Update:   podman auto-update"
echo ""
