#!/bin/bash
set -e

if ! id "$USERNAME" >/dev/null 2>&1; then
    useradd -m -s /bin/bash "$USERNAME"

    echo "${USERNAME}:${PASSWORD}" | chpasswd

    usermod -aG sudo "$USERNAME"

    echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" \
        > /etc/sudoers.d/${USERNAME}

    chmod 440 /etc/sudoers.d/${USERNAME}
fi

exec "$@"
