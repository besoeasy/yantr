#!/bin/sh

set -e

mkdir -p /config /downloads

# Configure crontab for daily auto-cleanup of downloads older than 30 days
cat >/etc/crontabs/root <<'EOF'
0 3 * * * find /downloads -type f -mtime +30 -delete 2>/dev/null
5 3 * * * find /downloads -type d -empty -delete 2>/dev/null
EOF

# Start cron daemon
crond -b -l 2

# Pre-seed core config if missing so default download location is /downloads
if [ ! -f /config/core.conf ]; then
  cat >/config/core.conf <<'EOF'
{
  "file": 1,
  "format": 1
}{
  "allow_remote": true,
  "download_location": "/downloads",
  "listen_ports": [
    58946,
    58946
  ],
  "random_port": false
}
EOF
fi

# Start Deluge daemon in background
deluged \
  --config /config \
  --loglevel=info &
DELUGED_PID=$!

# Give daemon time to initialize
sleep 3

# Start Deluge Web UI in background
deluge-web \
  --config /config &
WEB_PID=$!

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
trap 'kill -TERM $DELUGED_PID $WEB_PID $DUFS_PID 2>/dev/null' INT TERM

# Wait for background services
wait
