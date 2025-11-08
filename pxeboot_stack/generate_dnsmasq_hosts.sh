#!/bin/sh
set -euo pipefail

# This script generates dnsmasq configuration files from VM JSON definitions.

DHCP_HOSTS_FILE="/var/lib/pvmlab/dnsmasq.hosts"
DNS_HOSTS_FILE="/var/lib/pvmlab/dns.hosts"
VMS_DIR="/mnt/host/vms"

# Ensure the target directory exists
mkdir -p "$(dirname "$DHCP_HOSTS_FILE")"

# Create temporary files to build the new lists
TMP_DHCP_HOSTS_FILE=$(mktemp)
trap 'rm -f "$TMP_DHCP_HOSTS_FILE"' EXIT
TMP_DNS_HOSTS_FILE=$(mktemp)
trap 'rm -f "$TMP_DNS_HOSTS_FILE"' EXIT


# Process each JSON file in the VMs directory
for vm_file in "$VMS_DIR"/*.json; do
  if [[ ! -f "$vm_file" ]]; then
    continue
  fi

  # Extract MAC, IP, and name for all roles
  mac=$(jq -r '.mac' "$vm_file")
  ip=$(jq -r '.ip' "$vm_file")
  ipv6=$(jq -r '.ipv6' "$vm_file")
  name=$(jq -r '.name' "$vm_file")

  # Add entries to the DHCP hosts file, combining IPv4 and IPv6 on a single line.
  if [[ -n "$mac" && -n "$name" ]]; then
    line="$mac"
    has_any_ip=false
    if [[ -n "$ip" ]]; then
      line="$line,$ip"
      has_any_ip=true
    fi
    if [[ -n "$ipv6" && "$ipv6" != "null" ]]; then
      line="$line,[$ipv6]"
      has_any_ip=true
    fi
    
    if [[ "$has_any_ip" = true ]]; then
      line="$line,$name"
      echo "$line" >> "$TMP_DHCP_HOSTS_FILE"
    fi
  fi

  # Add entries to the DNS hosts file in the format: ip hostname
  if [[ -n "$ip" && -n "$name" ]]; then
    echo "$ip $name" >> "$TMP_DNS_HOSTS_FILE"
  fi
  if [[ -n "$ipv6" && "$ipv6" != "null" && -n "$name" ]]; then
    echo "$ipv6 $name" >> "$TMP_DNS_HOSTS_FILE"
  fi
done

# Atomically replace the old hosts files with the new ones
mv "$TMP_DHCP_HOSTS_FILE" "$DHCP_HOSTS_FILE"
mv "$TMP_DNS_HOSTS_FILE" "$DNS_HOSTS_FILE"

echo "dnsmasq hosts files generated at $DHCP_HOSTS_FILE and $DNS_HOSTS_FILE"
