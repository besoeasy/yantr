#!/bin/sh
set -e

# ─── Self-update via Watchtower ───────────────────────────────────────────────
# Only active when YANTR_OFFICIAL=true, which is baked in at build time by
# GitHub CI for the ghcr.io/besoeasy/yantr image.
# Local builds and custom images have YANTR_OFFICIAL=false — loop is skipped.

if [ "${YANTR_OFFICIAL:-false}" = "true" ]; then
    _interval="${YANTR_AUTOUPDATE_INTERVAL:-10800}" # seconds, default = 3 h

    # Find the name of the running Yantr container via Docker socket.
    # Prefers YANTR_CONTAINER_NAME env var; falls back to image-name scan.
    _find_container() {
        if [ -n "${YANTR_CONTAINER_NAME:-}" ]; then
            printf '%s' "$YANTR_CONTAINER_NAME"
            return
        fi
        _name=$(docker ps --format '{{.Names}}\t{{.Image}}' 2>/dev/null \
                | awk -F'\t' 'tolower($2) ~ /yantr/ { print $1; exit }')
        printf '%s' "${_name:-yantr}"
    }

    _update_loop() {
        while true; do
            sleep "$_interval"
            _cname=$(_find_container)
            echo "[update:self] running watchtower for: $_cname"
            docker run --rm \
                -v /var/run/docker.sock:/var/run/docker.sock \
                -e DOCKER_API_VERSION=1.44 \
                containrrr/watchtower \
                --run-once --cleanup \
                "$_cname" 2>&1 || true
        done
    }

    echo "[update:self] auto-update enabled (interval=${_interval}s)"
    _update_loop &
else
    echo "[update:self] auto-update disabled (not an official build)"
fi

# ─── Start Yantr (becomes PID 1) ─────────────────────────────────────────────
exec /app/yantr
