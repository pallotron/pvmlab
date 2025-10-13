#!/bin/sh
set -euo pipefail

# This script watches for changes in the VM directory and triggers the
# dnsmasq hosts file generation and a reload of the dnsmasq service.
# It uses polling instead of inotify because the underlying virtfs filesystem
# may not propagate file change events correctly.

VMS_DIR="/mnt/host/vms"
GENERATOR_SCRIPT="/usr/local/bin/generate_dnsmasq_hosts.sh"
POLL_INTERVAL=5 # seconds

# Function to get the state of the directory.
# We create a checksum of the contents of all json files.
get_dir_state() {
  # Check if directory exists and contains json files to avoid errors
  if [ -d "$VMS_DIR" ] && [ -n "$(find "$VMS_DIR" -maxdepth 1 -name '*.json' -print -quit)" ]; then
    cat "$VMS_DIR"/*.json | md5sum
  else
    echo "no-vms"
  fi
}

# Run the generator once on startup to ensure the hosts file is current
"$GENERATOR_SCRIPT"
# Signal dnsmasq to reload its configuration after the initial generation
pkill -HUP dnsmasq || echo "dnsmasq not running yet."

echo "Watching for changes in $VMS_DIR by polling every $POLL_INTERVAL seconds..."

last_state=$(get_dir_state)

# note: we don't use inotify here because the underlying virtfs filesystem may not propagate file change events correctly
while true; do
  sleep "$POLL_INTERVAL"
  current_state=$(get_dir_state)
  if [ "$current_state" != "$last_state" ]; then
    echo "Detected changes in $VMS_DIR. Regenerating dnsmasq hosts file..."
    "$GENERATOR_SCRIPT"
    # Signal dnsmasq to reload its configuration
    pkill -HUP dnsmasq || echo "Failed to reload dnsmasq. Is it running?"
    last_state="$current_state"
  fi
done
