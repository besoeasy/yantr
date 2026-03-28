#!/bin/sh
set -e

echo "Waiting for web container to be ready..."
ATTEMPTS=0
max_attempts=30

while [ $ATTEMPTS -lt $max_attempts ]; do
    if wget -q --spider http://web:80 2>/dev/null; then
        echo "Web container is ready!"
        break
    fi
    ATTEMPTS=$((ATTEMPTS + 1))
    echo "Waiting for web... ($ATTEMPTS/$max_attempts)"
    sleep 1
done

WEB_IP=$(getent hosts web | awk '{print $1}')
if [ -z "$WEB_IP" ]; then
    echo "Warning: Could not resolve web container, starting Tor without onion service"
    exec tor
fi

echo "Web container IP: $WEB_IP"

echo "Creating Tor config with onion service..."
cat > /tmp/torrc << EOF
SOCKSPort 0.0.0.0:9050
AutomapHostsOnResolve 1
VirtualAddrNetwork 10.192.0.0/10
HiddenServiceDir /var/lib/tor/onion_service
HiddenServicePort 80 ${WEB_IP}:80
EOF

cat /tmp/torrc

echo "Starting Tor..."
exec tor -f /tmp/torrc
