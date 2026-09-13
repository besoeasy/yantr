#!/bin/sh
set -e

# ─── Self-update loop ─────────────────────────────────────────────────────────
# Runs podman auto-update every YANTR_AUTOUPDATE_INTERVAL seconds (default 3h).

_interval="${YANTR_AUTOUPDATE_INTERVAL:-10800}"

_update_loop() {
    while true; do
        sleep "$_interval"
        echo "[update:self] running podman auto-update"
        podman auto-update 2>&1 || true
    done
}

_update_loop &

# ─── Start Yantr (becomes PID 1) ─────────────────────────────────────────────
exec /app/yantr
