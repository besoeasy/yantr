#!/bin/sh
set -e

# ─── Self-update loop ─────────────────────────────────────────────────────────
# Runs Watchtower every YANTR_AUTOUPDATE_INTERVAL seconds (default 3h).
# Finds the container whose name contains "yantr" via Docker socket.

_interval="${YANTR_AUTOUPDATE_INTERVAL:-10800}"

_update_loop() {
    while true; do
        sleep "$_interval"
        _cname=$(docker ps --format '{{.Names}}' 2>/dev/null \
                 | grep -i yantr | head -1)
        _cname="${_cname:-yantr}"
        echo "[update:self] running watchtower for: $_cname"
        docker run --rm \
            -v /var/run/docker.sock:/var/run/docker.sock \
            -e DOCKER_API_VERSION=1.44 \
            containrrr/watchtower \
            --run-once --cleanup \
            "$_cname" 2>&1 || true
    done
}

_update_loop &

# ─── Start Yantr (becomes PID 1) ─────────────────────────────────────────────
exec /app/yantr
