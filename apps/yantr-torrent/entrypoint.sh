#!/bin/sh

set -e

mkdir -p /config
mkdir -p /downloads

cat >/etc/crontabs/root <<'EOF'
0 3 * * * find /downloads -type f -mtime +30 -delete 2>/dev/null
5 3 * * * find /downloads -type d -empty -delete 2>/dev/null
EOF

crond

# Start Deluge daemon
deluged \
  --config /config \
  --loglevel=info

# Wait for daemon to initialize
sleep 5

# Start Deluge Web UI
deluge-web \
  --config /config &

# Start Dufs
dufs \
  /downloads \
  --bind 0.0.0.0 \
  --port 5000 \
  --allow-upload \
  --allow-delete \
  --allow-search \
  --allow-archive &

# Keep container alive by following Deluge
wait
