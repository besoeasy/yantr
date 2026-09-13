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

# Parse CLI options (--help, --uninstall)
ACTION="${1:-install}"

if [ "$ACTION" = "--help" ] || [ "$ACTION" = "-h" ] || [ "$ACTION" = "help" ]; then
  echo -e "${BOLD}Yantr Installer & Manager${NC}"
  echo ""
  echo "Usage:"
  echo "  curl -fsSL https://yantr.org/install.sh | bash                  # Install / update Yantr"
  echo "  curl -fsSL https://yantr.org/install.sh | bash -s -- --uninstall # Uninstall Yantr service"
  echo ""
  echo "Options:"
  echo "  --uninstall, -u   Uninstall Yantr systemd Quadlet service"
  echo "    --purge, -p     (Optional) Also delete persistent volume 'yantr_data'"
  echo "  --help, -h        Show this help message"
  echo ""
  exit 0
fi

if [ "$ACTION" = "--uninstall" ] || [ "$ACTION" = "-u" ] || [ "$ACTION" = "uninstall" ]; then
  log_step "Uninstalling Yantr service..."

  IS_ROOT=0
  if [ "$(id -u)" -eq 0 ]; then
    IS_ROOT=1
  fi

  if [ "$IS_ROOT" -eq 1 ]; then
    QUADLET_FILE="/etc/containers/systemd/yantr.container"
    SYSTEMCTL="systemctl"
  else
    export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
    export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=${XDG_RUNTIME_DIR}/bus}"
    QUADLET_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/containers/systemd/yantr.container"
    SYSTEMCTL="systemctl --user"
  fi

  # Stop and disable service
  log_info "Stopping and disabling Yantr systemd service..."
  $SYSTEMCTL stop yantr.service 2>/dev/null || true
  $SYSTEMCTL disable yantr.service 2>/dev/null || true

  # Remove Quadlet file
  if [ -f "$QUADLET_FILE" ]; then
    log_info "Removing Quadlet specification: ${QUADLET_FILE}"
    if [ "$IS_ROOT" -eq 1 ]; then
      $SUDO rm -f "$QUADLET_FILE"
    else
      rm -f "$QUADLET_FILE"
    fi
  fi

  # Reload systemd
  log_info "Reloading systemd daemon..."
  $SYSTEMCTL daemon-reload

  # Remove container if still exists
  if command -v podman >/dev/null 2>&1; then
    if podman container exists yantr 2>/dev/null; then
      log_info "Removing container 'yantr'..."
      podman rm -f yantr >/dev/null 2>&1 || true
    fi
  fi

  PURGE=0
  for arg in "$@"; do
    if [ "$arg" = "--purge" ] || [ "$arg" = "-p" ]; then
      PURGE=1
    fi
  done

  if [ "$PURGE" -eq 1 ]; then
    if command -v podman >/dev/null 2>&1; then
      log_info "Purging persistent volume 'yantr_data'..."
      podman volume rm -f yantr_data >/dev/null 2>&1 || true
    fi
    echo -e "\n${BOLD}${GREEN}✔ Yantr and all its data have been completely removed.${NC}\n"
  else
    echo -e "\n${BOLD}${GREEN}✔ Yantr systemd service successfully uninstalled.${NC}"
    echo -e "\n${BOLD}Note on Persistent Data:${NC}"
    echo "  Your app data in Podman volume 'yantr_data' has been preserved."
    echo "  To completely remove all data, run:"
    echo -e "    ${BOLD}podman volume rm yantr_data${NC}\n"
  fi
  exit 0
fi

# 2. Check / Install Podman
log_step "Checking Podman..."
if command -v podman >/dev/null 2>&1; then
  PODMAN_VER=$(podman --version | awk '{print $3}')
  log_info "Podman is already installed (version: ${PODMAN_VER})"
  if command -v apt-get >/dev/null 2>&1 && ! command -v newuidmap >/dev/null 2>&1; then
    log_info "Installing missing uidmap package for rootless container support..."
    $SUDO apt-get update -y && $SUDO apt-get install -y uidmap 2>/dev/null || true
  fi
else
  log_info "Podman not detected. Installing Podman..."
  if command -v dnf >/dev/null 2>&1; then
    $SUDO dnf install -y podman
  elif command -v apt-get >/dev/null 2>&1; then
    $SUDO apt-get update -y
    $SUDO apt-get install -y podman uidmap
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

  # Ensure user runtime dir is set for systemd user bus (especially over SSH)
  export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
  export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=${XDG_RUNTIME_DIR}/bus}"

  # Check /etc/subuid and /etc/subgid allocations
  if [ -f /etc/subuid ] && ! grep -q "^${USER}:" /etc/subuid 2>/dev/null; then
    USERMOD="usermod"
    if ! command -v usermod >/dev/null 2>&1 && [ -x /usr/sbin/usermod ]; then
      USERMOD="/usr/sbin/usermod"
    fi
    if [ -n "$SUDO" ]; then
      log_info "Configuring subuid and subgid ranges for ${USER}..."
      $SUDO "$USERMOD" --add-subuids 100000-165535 --add-subgids 100000-165535 "$USER" 2>/dev/null || true
    else
      log_warn "User '${USER}' is missing subuid/subgid allocations in /etc/subuid."
      log_warn "If rootless Podman fails, run: sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 ${USER}"
    fi
  fi

  # Enable user lingering so services persist after logout and auto-start on boot
  if command -v loginctl >/dev/null 2>&1; then
    IS_LINGERING="no"
    if [ -f "/var/lib/systemd/linger/$USER" ] || [ "$(loginctl show-user "$USER" --property=Linger 2>/dev/null)" = "Linger=yes" ]; then
      IS_LINGERING="yes"
    fi

    if [ "$IS_LINGERING" = "yes" ]; then
      log_info "User lingering is already enabled."
    else
      log_info "Enabling user lingering with loginctl..."
      # 1. Try non-interactive unprivileged loginctl without polkit tty agent
      if loginctl --no-ask-password enable-linger "$USER" >/dev/null 2>&1; then
        :
      # 2. Try passwordless sudo if available
      elif [ -n "$SUDO" ] && sudo -n true 2>/dev/null; then
        $SUDO loginctl --no-ask-password enable-linger "$USER" >/dev/null 2>&1 || true
      # 3. If running interactively with sudo available, prompt cleanly with context
      elif [ -n "$SUDO" ] && [ -c /dev/tty ]; then
        log_info "Enabling user lingering requires sudo privileges (keeps Yantr running after logout):"
        $SUDO loginctl --no-ask-password enable-linger "$USER" </dev/tty >/dev/null 2>&1 || true
      fi

      if [ -f "/var/lib/systemd/linger/$USER" ] || [ "$(loginctl show-user "$USER" --property=Linger 2>/dev/null)" = "Linger=yes" ]; then
        log_info "User lingering enabled."
      else
        log_warn "Could not enable user lingering automatically."
        log_warn "Containers might stop when you log out. To keep Yantr running 24/7, run: sudo loginctl enable-linger $USER"
      fi
    fi
  fi

  # Enable and start rootless podman.socket
  log_info "Enabling user Podman socket..."
  systemctl --user --no-ask-password enable --now podman.socket

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

# Remove pre-existing standalone container to avoid name collision with Quadlet
if command -v podman >/dev/null 2>&1; then
  if podman container exists yantr 2>/dev/null; then
    log_info "Removing existing standalone 'yantr' container to avoid name collision..."
    podman rm -f yantr >/dev/null 2>&1 || true
  fi
fi

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

HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
if [ -n "$HOST_IP" ] && [ "$HOST_IP" != "127.0.0.1" ]; then
  echo -e "Access the Web UI:  ${BOLD}${GREEN}http://${HOST_IP}:5252${NC}  (or http://localhost:5252)"
else
  echo -e "Access the Web UI:  ${BOLD}${GREEN}http://localhost:5252${NC}"
fi
echo -e "\n${BOLD}Useful Management Commands:${NC}"
echo -e "  • Status:   ${SYSTEMCTL} status yantr"
echo -e "  • Logs:     journalctl $([ "$IS_ROOT" -eq 0 ] && echo "--user ") -u yantr -f"
echo -e "  • Restart:   ${SYSTEMCTL} restart yantr"
echo -e "  • Stop:      ${SYSTEMCTL} stop yantr"
echo -e "  • Update:    podman auto-update"
echo -e "  • Uninstall: curl -fsSL https://yantr.org/install.sh | bash -s -- --uninstall"
echo ""
