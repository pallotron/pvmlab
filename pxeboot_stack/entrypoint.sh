#!/bin/sh
set -e

echo $PROVISIONER_IP

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

# Ensure the hosts file exists so dnsmasq can start
mkdir -p /var/lib/pvmlab
touch /var/lib/pvmlab/dnsmasq.hosts

# Execute the command passed to this script (e.g., /usr/bin/supervisord)
exec "$@"
