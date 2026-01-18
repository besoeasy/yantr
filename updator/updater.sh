#!/bin/sh

# Yantra Auto-Updater (lightweight)
# Runs every hour and, if the image changed, atomically recreates the Yantra container.
#
# Important: `docker restart` is NOT enough to apply a new image.

UPDATE_INTERVAL_SECONDS="${YANTRA_AUTO_UPDATE_INTERVAL_SECONDS:-3600}"

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

log "Yantra Auto-Updater loop started (interval: ${UPDATE_INTERVAL_SECONDS}s)"

while true; do
  sleep "$UPDATE_INTERVAL_SECONDS"
  log "Checking for updates..."
  bun updator/self_update.js
done
