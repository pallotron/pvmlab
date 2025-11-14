#!/bin/sh
# This script is called by mdev when a network interface is added.
# The interface name is passed as the first argument ($1).

INTERFACE=$1

# Basic sanity check
if [ -z "$INTERFACE" ]; then
    exit 1
fi

echo "==> mdev: Bringing up network interface ${INTERFACE}"
ip link set dev "${INTERFACE}" up

echo "==> mdev: Starting udhcpc on ${INTERFACE}"
# -b: run in background
# -q: exit after obtaining lease
udhcpc -i "${INTERFACE}" -b -q
