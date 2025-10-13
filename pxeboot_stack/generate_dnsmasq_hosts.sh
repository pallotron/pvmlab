#!/bin/sh
set -euo pipefail

# This script generates a dnsmasq hosts file from VM JSON definitions.

HOSTS_FILE="/var/lib/pvmlab/dnsmasq.hosts"
VMS_DIR="/mnt/host/vms"

# Ensure the target directory exists
mkdir -p "$(dirname "$HOSTS_FILE")"

# Create a temporary file to build the new hosts list
TMP_HOSTS_FILE=$(mktemp)
trap 'rm -f "$TMP_HOSTS_FILE"' EXIT

# Process each JSON file in the VMs directory
for vm_file in "$VMS_DIR"/*.json; do
  if [[ ! -f "$vm_file" ]]; then
    continue
  fi

  # Skip the provisioner, as it does not get its IP from dnsmasq
  if jq -e '.role == "provisioner"' "$vm_file" > /dev/null; then
    continue
  fi

  # Extract MAC and IP for all other roles (e.g., "target")
  mac=$(jq -r '.mac' "$vm_file")
  ip=$(jq -r '.ip' "$vm_file")

  # Add the entry to our temporary hosts file
  if [[ -n "$mac" && -n "$ip" ]]; then
    echo "$mac,$ip" >> "$TMP_HOSTS_FILE"
  fi
done

# Atomically replace the old hosts file with the new one
mv "$TMP_HOSTS_FILE" "$HOSTS_FILE"

echo "dnsmasq hosts file generated at $HOSTS_FILE"
