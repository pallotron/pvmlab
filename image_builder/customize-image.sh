#!/bin/bash
set -euo pipefail

export LIBGUESTFS_BACKEND=direct

IMAGE_PATH=$1
if [ -z "${IMAGE_PATH}" ]; then
    echo "Error: Image path not provided." >&2
    exit 1
fi

# --- Install Dependencies in the Container ---
echo "Updating container and installing dependencies..."
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y libguestfs-tools qemu-utils cloud-guest-utils curl

# --- Customize the Image ---
echo "Starting image customization..."

# --- Resize the disk image ---
echo "Resizing the disk image..."
qemu-img resize "${IMAGE_PATH}" +10G

# --- Expand the partition and filesystem ---
echo "Expanding the root partition..."
virt-customize -a "${IMAGE_PATH}" \
    --run-command 'growpart /dev/sda 1' \
    --run-command 'resize2fs /dev/sda1'

# --- Install packages ---
virt-customize -a "${IMAGE_PATH}" \
    --run-command 'install -m 0755 -d /etc/apt/keyrings' \
    --run-command 'curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg' \
    --run-command 'chmod a+r /etc/apt/keyrings/docker.gpg' \
    --run-command 'echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null' \
    --run-command 'apt-get update' \
    --install docker-ce,docker-ce-cli,containerd.io,acpid,iptables-persistent,ca-certificates,curl,gnupg,radvd \
    --run-command 'rm -f /etc/update-motd.d/50-landscape-sysinfo' \
    --run-command 'rm -f /etc/update-motd.d/10-help-text' \
    --run-command 'rm -f /etc/update-motd.d/50-motd-news' \
    --run-command 'rm -f /etc/update-motd.d/90-updates-available'

# --- Sparsify the image to reduce its size ---
echo "Sparsifying the image..."
virt-sparsify --in-place "${IMAGE_PATH}"

# --- Compress the image ---
echo "Compressing the image..."
mv "${IMAGE_PATH}" "${IMAGE_PATH}.tmp"
qemu-img convert -c -O qcow2 "${IMAGE_PATH}.tmp" "${IMAGE_PATH}"
rm "${IMAGE_PATH}.tmp"

echo "Image customization complete."
