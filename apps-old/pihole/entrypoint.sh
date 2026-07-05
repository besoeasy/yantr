#!/bin/bash
set -e

# If WEBPASSWORD is set, use pihole setpassword to set it
if [ -n "$WEBPASSWORD" ]; then
    # Wait for pihole to start before setting password
    sleep 10
    pihole setpassword "$WEBPASSWORD"
fi

# Continue with normal pihole startup
exec /s6-init
