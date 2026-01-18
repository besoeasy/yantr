#!/bin/sh
set -eu

# Starts Yantra server and (optionally) the hourly auto-updater loop.
#
# Auto-update is enabled by default.
# Disable by setting: YANTRA_AUTO_UPDATE=false

cleanup() {
  if [ -n "${UPDATER_PID:-}" ] && kill -0 "$UPDATER_PID" 2>/dev/null; then
    kill "$UPDATER_PID" 2>/dev/null || true
  fi
}
trap cleanup INT TERM

# Start server
bun run server &
SERVER_PID=$!

# Start updater loop only if enabled
if [ "${YANTRA_AUTO_UPDATE:-true}" != "" ] && echo "${YANTRA_AUTO_UPDATE:-true}" | grep -Eiq '^(0|false|no|off)$'; then
  : "auto-update disabled"
else
  ./updator/updater.sh &
  UPDATER_PID=$!
fi

wait "$SERVER_PID"
