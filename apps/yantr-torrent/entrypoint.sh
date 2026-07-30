#!/bin/sh

set -e

mkdir -p /config /downloads /config/flood

# Configure crontab for daily auto-cleanup of downloads older than 30 days
cat >/etc/crontabs/root <<'EOF'
0 3 * * * find /downloads -type f -mtime +30 -delete 2>/dev/null
5 3 * * * find /downloads -type d -empty -delete 2>/dev/null
EOF

# Start cron daemon
crond -b -l 2

# Pre-seed Transmission settings if missing
if [ ! -f /config/settings.json ]; then
  cat >/config/settings.json <<'EOF'
{
  "download-dir": "/downloads",
  "incomplete-dir": "/downloads/incomplete",
  "incomplete-dir-enabled": false,
  "rpc-authentication-required": false,
  "rpc-bind-address": "0.0.0.0",
  "rpc-enabled": true,
  "rpc-port": 9091,
  "rpc-whitelist-enabled": false,
  "peer-port": 51413
}
EOF
fi

# Start Transmission daemon in background
transmission-daemon -g /config -f &
TR_PID=$!

# Give daemon time to initialize
sleep 2

# Start Flood Web UI in background pointing to Transmission RPC
flood \
  --host 0.0.0.0 \
  --port 3000 \
  --rundir /config/flood \
  --auth none \
  --trurl http://127.0.0.1:9091/transmission/rpc \
  --truser "" \
  --trpass "" &
FLOOD_PID=$!

# Start Dufs WebDAV file browser in background
dufs \
  /downloads \
  --bind 0.0.0.0 \
  --port 5000 \
  --allow-upload \
  --allow-delete \
  --allow-search \
  --allow-archive &
DUFS_PID=$!

# Trap signals for graceful shutdown
trap 'kill -TERM $TR_PID $FLOOD_PID $DUFS_PID 2>/dev/null' INT TERM

# Wait for background services
wait
