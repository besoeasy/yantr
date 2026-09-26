#!/usr/bin/env bash
# ==============================================================================
#  Yantr Installer — Linux (Rootless Podman + Quadlet systemd)
#  https://yantr.org
#
#  Non-interactive by design: safe to run via `curl -fsSL ... | bash`.
#  See --help for options. Honors NO_COLOR, YANTR_PLAIN, YANTR_VERBOSE.
# ==============================================================================

set -euo pipefail

SCRIPT_START=$SECONDS

# --------------------------------------------------------------------------
# CLI options (parsed before anything else so --help/--no-color apply early)
# --------------------------------------------------------------------------
MODE="install"   # install | uninstall | help
VERBOSE="${YANTR_VERBOSE:-0}"
PURGE=0
NO_COLOR_REQ=0

for arg in "$@"; do
  case "$arg" in
    --help|-h|help)           MODE="help" ;;
    --uninstall|-u|uninstall) MODE="uninstall" ;;
    --purge|-p)               PURGE=1 ;;
    --verbose|-v)             VERBOSE=1 ;;
    --yes|-y)                 ;; # accepted for compatibility; installer never prompts
    --no-color)               NO_COLOR_REQ=1 ;;
    *) echo "Warning: unknown option '$arg' (ignored)" ;;
  esac
done

# --------------------------------------------------------------------------
# Terminal UX — colors, symbols, logging
# Policy (same as modern CLIs like brew/gh/bun):
#   * tput-based palette when available, ANSI fallback otherwise
#   * fully disabled via NO_COLOR, --no-color, dumb TERM, or non-TTY output
# --------------------------------------------------------------------------
USE_COLOR=1
if [ "$NO_COLOR_REQ" -eq 1 ] || [ -n "${NO_COLOR:-}" ] || [ ! -t 1 ] || [ "${TERM:-xterm}" = "dumb" ]; then
  USE_COLOR=0
fi

if [ "$USE_COLOR" -eq 1 ]; then
  if command -v tput >/dev/null 2>&1 && tput setaf 1 >/dev/null 2>&1; then
    BOLD="$(tput bold)"; DIM="$(tput dim)"
    GREEN="$(tput setaf 2)"; BLUE="$(tput setaf 4)"; CYAN="$(tput setaf 6)"
    YELLOW="$(tput setaf 3)"; RED="$(tput setaf 1)"; MAGENTA="$(tput setaf 5)"
    NC="$(tput sgr0)"
  else
    BOLD="\033[1m"; DIM="\033[2m"
    GREEN="\033[0;32m"; BLUE="\033[0;34m"; CYAN="\033[0;36m"
    YELLOW="\033[1;33m"; RED="\033[0;31m"; MAGENTA="\033[0;35m"
    NC="\033[0m"
  fi
else
  BOLD=""; DIM=""; GREEN=""; BLUE=""; CYAN=""
  YELLOW=""; RED=""; MAGENTA=""; NC=""
fi

# Symbols degrade to ASCII on non-UTF8 locales (or YANTR_PLAIN=1),
# so output stays readable over minimal SSH sessions.
if [ "${YANTR_PLAIN:-0}" = "1" ]; then
  ASCII=1
else
  _charmap="$(locale charmap 2>/dev/null || echo "UTF-8")"
  case "$_charmap" in
    *[Uu][Tt][Ff]*8*|*[Uu][Tt][Ff]*) ASCII=0 ;;
    *) ASCII=1 ;;
  esac
fi
if [ "$ASCII" -eq 1 ]; then
  TICK="[ok]"; CROSS="[fail]"; WARNM="[warn]"; INFOM="[info]"
  ARROW="==>"; BULLET="-"
else
  TICK="✔"; CROSS="✖"; WARNM="⚠"; INFOM="ℹ"
  ARROW="▸"; BULLET="•"
fi

STEP_CURRENT=0
STEP_TOTAL=5
STEP_START=$SECONDS

log_info()    { echo -e "${BLUE}${INFOM}${NC}  $*"; }
log_step()    { STEP_CURRENT=$((STEP_CURRENT + 1)); STEP_START=$SECONDS;
                echo -e "\n${BOLD}${GREEN}${ARROW}${NC} ${BOLD}[${STEP_CURRENT}/${STEP_TOTAL}] $*${NC}"; }
step_ok()     { local dt=$((SECONDS - STEP_START));
                echo -e "${GREEN}${TICK}${NC}  $* ${DIM}(${dt}s)${NC}"; }
log_success() { echo -e "${BOLD}${GREEN}${TICK} $*${NC}"; }
log_warn()    { echo -e "${YELLOW}${WARNM}${NC}  $*"; }
log_error()   { echo -e "${RED}${CROSS}${NC}  $*" >&2; }
log_debug()   { if [ "$VERBOSE" -eq 1 ]; then echo -e "${DIM}${MAGENTA}debug:${NC}${DIM} $*${NC}"; fi; }
die()         { log_error "$*"; exit 1; }
have()        { command -v "$1" >/dev/null 2>&1; }

# Show the failing command + line on any unguarded error (complements set -e).
on_error() { log_error "Installer failed at line $1: $2"; }
trap 'on_error "$LINENO" "$BASH_COMMAND"' ERR

# --------------------------------------------------------------------------
# Help
# --------------------------------------------------------------------------
if [ "$MODE" = "help" ]; then
  echo -e "${BOLD}${CYAN}Yantr Installer & Manager${NC}"
  echo -e "${DIM}Native Linux · Rootless Podman · Quadlet systemd${NC}"
  echo ""
  echo -e "${BOLD}Usage:${NC}"
  echo -e "  ${CYAN}curl -fsSL https://yantr.org/install.sh | bash${NC}                  # Install / update / repair Yantr"
  echo -e "  ${CYAN}curl -fsSL https://yantr.org/install.sh | bash -s -- --uninstall${NC} # Uninstall Yantr service"
  echo ""
  echo -e "${BOLD}Options:${NC}"
  echo "  --uninstall, -u   Uninstall Yantr systemd Quadlet service"
  echo "    --purge, -p     Also delete persistent volume 'yantr_data'"
  echo "  --verbose, -v     Show debug details (detections, paths, timings)"
  echo "  --yes, -y         Accepted for compatibility; installer never prompts"
  echo "  --no-color        Disable colored output (same as NO_COLOR=1)"
  echo "  --help, -h        Show this help message"
  echo ""
  echo -e "${BOLD}Environment:${NC}"
  echo "  NO_COLOR=1        Disable colors"
  echo "  YANTR_PLAIN=1     ASCII-only symbols (no unicode)"
  echo "  YANTR_VERBOSE=1   Same as --verbose"
  echo ""
  echo -e "${BOLD}Examples:${NC}"
  echo "  bash install.sh --verbose"
  echo "  bash install.sh --uninstall --purge"
  echo ""
  exit 0
fi

echo -e "${BOLD}${CYAN}"
echo "    __  __            __       "
echo "    \ \/ /___ _____  / /______ "
echo "     \  / __ \`/__ \/ __/ ___/ "
echo "     / / /_/ / / / / /_/ /     "
echo "    /_/\__,_/_/ /_/\__/_/      "
echo -e "${NC}"
echo -e "${BOLD}Yantr — Self-Hosted Homelab App Store${NC}"
echo -e "${DIM}Native Linux · Rootless Podman · Quadlet systemd${NC}\n"

# 1. Linux check
if [ "$(uname -s)" != "Linux" ]; then
  log_error "Yantr is designed for Linux using rootless Podman and Quadlet."
  echo "        For macOS or Windows, run Yantr inside a Linux VM or WSL2 instance."
  exit 1
fi

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  if have sudo; then
    SUDO="sudo"
  fi
fi

IS_ROOT=0
if [ "$(id -u)" -eq 0 ]; then
  IS_ROOT=1
fi

# Environment preflight — modern installers show what they detected up front.
DISTRO="$(grep '^PRETTY_NAME=' /etc/os-release 2>/dev/null | cut -d= -f2- | tr -d '"' || true)"
if [ -z "$DISTRO" ]; then
  DISTRO="Unknown Linux"
fi
ARCH="$(uname -m)"
if [ "$IS_ROOT" -eq 1 ]; then
  RUN_MODE="root"
else
  RUN_MODE="rootless (user: ${USER:-unknown})"
fi
if have podman; then
  PODMAN_INFO="$(podman --version 2>/dev/null || echo "podman: unknown version")"
else
  PODMAN_INFO="not installed"
fi
echo -e "${DIM}Environment:${NC} ${DISTRO} · ${ARCH} · ${RUN_MODE} · ${PODMAN_INFO}"
log_debug "sudo helper: '${SUDO:-<none>}'"
log_debug "verbose=$VERBOSE purge=$PURGE use_color=$USE_COLOR ascii=$ASCII"

if [ "$MODE" = "uninstall" ]; then
  STEP_TOTAL=3

  if [ "$IS_ROOT" -eq 1 ]; then
    QUADLET_FILE="/etc/containers/systemd/yantr.container"
    SYSTEMCTL="systemctl"
  else
    export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
    export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=${XDG_RUNTIME_DIR}/bus}"
    QUADLET_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/containers/systemd/yantr.container"
    SYSTEMCTL="systemctl --user"
  fi
  log_debug "quadlet file: $QUADLET_FILE"

  # 1/3 Stop and disable service, remove container if still exists
  log_step "Stopping Yantr service..."
  $SYSTEMCTL stop yantr.service 2>/dev/null || true
  $SYSTEMCTL disable yantr.service 2>/dev/null || true
  if have podman; then
    if podman container exists yantr 2>/dev/null; then
      log_info "Removing container 'yantr'..."
      podman rm -f yantr >/dev/null 2>&1 || true
    fi
  fi
  step_ok "Service stopped and disabled."

  # 2/3 Remove Quadlet file, reload systemd
  log_step "Removing Quadlet specification..."
  if [ -f "$QUADLET_FILE" ]; then
    log_info "Removing: ${QUADLET_FILE}"
    if [ "$IS_ROOT" -eq 1 ]; then
      $SUDO rm -f "$QUADLET_FILE"
    else
      rm -f "$QUADLET_FILE"
    fi
  else
    log_info "No Quadlet file found (already removed)."
  fi
  log_info "Reloading systemd daemon..."
  $SYSTEMCTL daemon-reload
  step_ok "Quadlet specification removed."

  # 3/3 Purge data if requested
  log_step "Finalizing..."
  if [ "$PURGE" -eq 1 ]; then
    if have podman; then
      log_info "Purging persistent volume 'yantr_data'..."
      podman volume rm -f yantr_data >/dev/null 2>&1 || true
    fi
    step_ok "Purge complete."
    echo -e "\n${BOLD}${GREEN}${TICK} Yantr and all its data have been completely removed.${NC}\n"
  else
    step_ok "Uninstall complete."
    echo -e "\n${BOLD}${GREEN}${TICK} Yantr systemd service successfully uninstalled.${NC}"
    echo -e "\n${BOLD}Note on Persistent Data:${NC}"
    echo "  Your app data in Podman volume 'yantr_data' has been preserved."
    echo "  To completely remove all data, run:"
    echo -e "    ${BOLD}${CYAN}podman volume rm yantr_data${NC}\n"
  fi
  exit 0
fi

# 2. Check / Install Podman
log_step "Checking Podman..."
if have podman; then
  PODMAN_VER=$(podman --version | awk '{print $3}')
  log_info "Podman is already installed (version: ${PODMAN_VER})"
  if have apt-get && ! have newuidmap; then
    log_info "Installing missing uidmap package for rootless container support..."
    $SUDO apt-get update -y && $SUDO apt-get install -y uidmap 2>/dev/null || true
  fi
else
  log_info "Podman not detected. Installing Podman (this may take a minute)..."
  if have dnf; then
    $SUDO dnf install -y podman
  elif have apt-get; then
    $SUDO apt-get update -y
    $SUDO apt-get install -y podman uidmap
  elif have pacman; then
    $SUDO pacman -Sy --noconfirm podman
  elif have zypper; then
    $SUDO zypper in -y podman
  elif have apk; then
    $SUDO apk add podman
  else
    die "Could not detect a supported package manager (dnf, apt, pacman, zypper, apk). Please install Podman manually and run this script again."
  fi
  log_info "Podman installed successfully."
fi

# Builds triggered via the host Podman socket default to
# /etc/containers/seccomp.json on the server side, but
# Debian/Ubuntu only ship /usr/share/containers/seccomp.json — so builds fail with
# "opening seccomp profile failed". Link the system path when it's missing.
if [ ! -f /etc/containers/seccomp.json ] && [ -f /usr/share/containers/seccomp.json ]; then
  log_info "Linking missing /etc/containers/seccomp.json for remote builds..."
  if [ "$IS_ROOT" -eq 1 ]; then
    mkdir -p /etc/containers && ln -sf /usr/share/containers/seccomp.json /etc/containers/seccomp.json || true
  elif [ -n "$SUDO" ]; then
    $SUDO mkdir -p /etc/containers && $SUDO ln -sf /usr/share/containers/seccomp.json /etc/containers/seccomp.json || true
  else
    log_warn "Cannot create /etc/containers/seccomp.json (no sudo). Remote app builds may fail;"
    log_warn "run manually: sudo mkdir -p /etc/containers && sudo ln -sf /usr/share/containers/seccomp.json /etc/containers/seccomp.json"
  fi
fi
step_ok "Podman ready."

# 3. Configure systemd & socket
if [ "$IS_ROOT" -eq 1 ]; then
  log_step "Configuring Podman system service (root mode)..."
  systemctl enable --now podman.socket
  QUADLET_DIR="/etc/containers/systemd"
  $SUDO mkdir -p "$QUADLET_DIR"
  QUADLET_FILE="$QUADLET_DIR/yantr.container"
  SYSTEMCTL="systemctl"
  SOCKET_BIND="/run/podman/podman.sock:/run/podman/podman.sock"
  WANTED_BY="multi-user.target"
else
  log_step "Configuring Rootless Podman & systemd user service..."

  # Ensure user runtime dir is set for systemd user bus (especially over SSH)
  export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
  export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=${XDG_RUNTIME_DIR}/bus}"

  # Check /etc/subuid and /etc/subgid allocations
  if [ -f /etc/subuid ] && ! grep -q "^${USER}:" /etc/subuid 2>/dev/null; then
    USERMOD="usermod"
    if ! have usermod && [ -x /usr/sbin/usermod ]; then
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
  if have loginctl; then
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
  WANTED_BY="default.target"
fi
log_debug "quadlet file: $QUADLET_FILE"
log_debug "socket bind: $SOCKET_BIND"
step_ok "System service configured."

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
WantedBy=${WANTED_BY}"

if [ "$IS_ROOT" -eq 1 ]; then
  echo "$QUADLET_CONTENT" | $SUDO tee "$QUADLET_FILE" > /dev/null
else
  echo "$QUADLET_CONTENT" > "$QUADLET_FILE"
fi
step_ok "Quadlet specification written."

# 5. Reload systemd and start Yantr service
log_step "Activating Yantr systemd service..."
log_info "First start pulls the image — this may take a minute..."

# Remove pre-existing standalone container to avoid name collision with Quadlet
if have podman; then
  if podman container exists yantr 2>/dev/null; then
    log_info "Removing existing standalone 'yantr' container to avoid name collision..."
    podman rm -f yantr >/dev/null 2>&1 || true
  fi
fi

$SYSTEMCTL daemon-reload
$SYSTEMCTL restart yantr.service

# Also enable podman auto-update timer if available
$SYSTEMCTL enable --now podman-auto-update.timer 2>/dev/null || true
step_ok "Service activated."

# 6. Verify health
log_step "Verifying Yantr status..."
sleep 2

ELAPSED=$((SECONDS - SCRIPT_START))

if $SYSTEMCTL is-active --quiet yantr.service; then
  echo -e "\n${BOLD}${GREEN}${TICK} Yantr is up and running!${NC} ${DIM}(${ELAPSED}s)${NC}\n"
else
  log_warn "Service was started, but may still be pulling image or initializing."
  echo "Check status with: ${SYSTEMCTL} status yantr"
fi

HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
if [ -n "$HOST_IP" ] && [ "$HOST_IP" != "127.0.0.1" ]; then
  echo -e "  ${BOLD}Web UI:${NC}  ${BOLD}${GREEN}http://${HOST_IP}:5252${NC}  ${DIM}(or http://localhost:5252)${NC}"
else
  echo -e "  ${BOLD}Web UI:${NC}  ${BOLD}${GREEN}http://localhost:5252${NC}"
fi
echo -e "\n  ${BOLD}Next steps:${NC}"
echo -e "  ${BULLET} Status:    ${CYAN}${SYSTEMCTL} status yantr${NC}"
echo -e "  ${BULLET} Logs:      ${CYAN}journalctl $([ "$IS_ROOT" -eq 0 ] && echo "--user ") -u yantr -f${NC}"
echo -e "  ${BULLET} Restart:   ${CYAN}${SYSTEMCTL} restart yantr${NC}"
echo -e "  ${BULLET} Update:    ${CYAN}podman auto-update${NC}"
echo -e "  ${BULLET} Uninstall: ${CYAN}curl -fsSL https://yantr.org/install.sh | bash -s -- --uninstall${NC}"
echo -e "\n  ${DIM}Tip: re-run this installer anytime to update or repair the setup.${NC}"
echo ""
