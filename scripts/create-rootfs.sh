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
apt-get update > /dev/null
apt-get install -y libguestfs-tools pv > /dev/null

# --- Create the rootfs tarball ---
echo "Creating rootfs tarball..."
ROOTFS_PATH=$(dirname "${IMAGE_PATH}")/rootfs.tar.gz

# Get the uncompressed size of the filesystem for pv
SIZE=$(guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda1 du / | awk '{print $1}')

# Create the tarball with a progress bar
guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda1 tar-out / - | pv -s "${SIZE}" | gzip > "${ROOTFS_PATH}"

echo "Rootfs tarball created successfully."

