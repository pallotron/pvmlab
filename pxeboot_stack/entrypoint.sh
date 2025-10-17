#!/bin/sh
set -e

# Ensure required environment variables are set
if [ -z "$PROVISIONER_IP" ]; then
  echo "Error: PROVISIONER_IP environment variable is not set."
  exit 1
fi

# Derive the network prefix from the provisioner's IP address
# e.g., "192.168.100.1" -> "192.168.100"
export NETWORK=$(echo "$PROVISIONER_IP" | cut -d. -f1-3)

# Substitute variables in the template to create the final config file
envsubst < /etc/dnsmasq.conf.template > /etc/dnsmasq.conf

# Conditionally add IPv6 DHCP range if the variables are set
if [ -n "${DHCP_RANGE_V6_START:-}" ] && [ -n "${DHCP_RANGE_V6_END:-}" ]; then
  echo "enable-ra" >> /etc/dnsmasq.conf
  echo "dhcp-range=${DHCP_RANGE_V6_START},${DHCP_RANGE_V6_END},constructor:enp0s2,12h" >> /etc/dnsmasq.conf
fi

# Ensure the hosts file exists so dnsmasq can start
mkdir -p /var/lib/pvmlab
touch /var/lib/pvmlab/dnsmasq.hosts

# Execute the command passed to this script (e.g., /usr/bin/supervisord)
exec "$@"
