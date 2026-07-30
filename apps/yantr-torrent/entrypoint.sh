#!/bin/sh

set -e

mkdir -p /downloads

cat >/etc/crontabs/root <<'EOF'
0 3 * * * find /downloads -type f -mtime +30 -delete 2>/dev/null
5 3 * * * find /downloads -type d -empty -delete 2>/dev/null
EOF

crond

exec dufs \
  /downloads \
  --bind 0.0.0.0 \
  --port 5000 \
  --allow-upload \
  --allow-delete \
  --allow-search \
  --allow-archive
