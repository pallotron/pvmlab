#!/bin/bash
set -euo pipefail

# Generate a UUID for the vmnet network identifier.
# The identifier is stored in a file to persist across restarts of the service.
# This ensures that the same network identifier is used until the file is removed.
STATE_DIR="/var/run/pvmlab"
IDENTIFIER_FILE="${STATE_DIR}/socket_vmnet_network_identifier"

mkdir -p "${STATE_DIR}"

if [ ! -f "${IDENTIFIER_FILE}" ]; then
    uuidgen > "${IDENTIFIER_FILE}"
fi

NETWORK_IDENTIFIER=$(cat "${IDENTIFIER_FILE}")

exec /opt/homebrew/opt/socket_vmnet/bin/socket_vmnet \
    --vmnet-mode=host \
    --vmnet-network-identifier="${NETWORK_IDENTIFIER}" \
    /var/run/vmlab.socket_vmnet