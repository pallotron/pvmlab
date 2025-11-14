#!/bin/bash
set -euo pipefail

export LIBGUESTFS_BACKEND=direct

IMAGE_PATH=$1
DISTRO_NAME=$2

if [ -z "${IMAGE_PATH}" ]; then
    echo "Error: Image path not provided." >&2
    exit 1
fi

if [ -z "${DISTRO_NAME}" ]; then
    echo "Error: Distro name not provided." >&2
    exit 1
fi

# --- Install Dependencies in the Container ---
echo "Updating container and installing dependencies..."
export DEBIAN_FRONTEND=noninteractive
apt-get update > /dev/null
apt-get install -y libguestfs-tools pv btrfs-progs > /dev/null

echo "Available RAM in the container:"
free -h


# --- Create the rootfs tarball ---
echo "Creating rootfs tarball for ${DISTRO_NAME}..."
ROOTFS_PATH=$(dirname "${IMAGE_PATH}")/rootfs.tar.gz

if [ "${DISTRO_NAME}" == "fedora" ]; then
    echo "Using Fedora BTRFS layout (/dev/sda4 for root, /dev/sda3 for boot)..."
    SIZE=$(guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda4:/:subvol=root -m /dev/sda3:/boot du / | awk '{print $1}')
    guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda4:/:subvol=root -m /dev/sda3:/boot tar-out / - | pv -s "${SIZE}" | gzip > "${ROOTFS_PATH}"
elif [ "${DISTRO_NAME}" == "ubuntu" ]; then
    echo "Using Ubuntu EXT4 layout..."
    if guestfish --ro -a "${IMAGE_PATH}" run : list-filesystems | grep -q /dev/sda16; then
        echo "Found /dev/sda16, assuming it is the boot partition and including it in the rootfs."
        SIZE=$(guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda1 -m /dev/sda16:/boot du / | awk '{print $1}')
        guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda1 -m /dev/sda16:/boot tar-out / - | pv -s "${SIZE}" | gzip > "${ROOTFS_PATH}"
    else
        echo "No /dev/sda16 found, creating rootfs from /dev/sda1 only."
        SIZE=$(guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda1 du / | awk '{print $1}')
        guestfish --ro -a "${IMAGE_PATH}" -m /dev/sda1 tar-out / - | pv -s "${SIZE}" | gzip > "${ROOTFS_PATH}"
    fi
else
    echo "Error: Unsupported distro '${DISTRO_NAME}'" >&2
    exit 1
fi

echo "Rootfs tarball created successfully."

