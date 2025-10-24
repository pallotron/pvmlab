#!/bin/bash
set -euo pipefail

# Configuration
ARCH=${1:-arm64} # Default to arm64 if no arch is provided
BASE_IMAGE_URL="https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-${ARCH}.img"
BASE_IMAGE_NAME=$(basename "${BASE_IMAGE_URL}")
CUSTOM_IMAGE_NAME="provisioner-custom.${ARCH}.qcow2"
BUILD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
IMAGE_DIR="${BUILD_DIR}/output"

# Ensure output directory exists
mkdir -p "${IMAGE_DIR}"

# --- Download Base Image ---
if [ ! -f "${IMAGE_DIR}/${BASE_IMAGE_NAME}" ]; then
    echo "Downloading base image for ${ARCH}..."
    curl -L -o "${IMAGE_DIR}/${BASE_IMAGE_NAME}" "${BASE_IMAGE_URL}"
else
    echo "Base image for ${ARCH} already downloaded."
fi

# --- Create a copy to customize ---
echo "Creating a fresh copy of the image..."
cp "${IMAGE_DIR}/${BASE_IMAGE_NAME}" "${IMAGE_DIR}/${CUSTOM_IMAGE_NAME}"

# --- Run Customization in Docker ---
echo "Running customization script in Docker..."
docker run --rm -it \
    --privileged \
    -v "${IMAGE_DIR}:/images" \
    -v "${BUILD_DIR}/files:/files" \
    -v "${BUILD_DIR}/customize-image.sh:/customize-image.sh" \
    --entrypoint /bin/bash \
    debian:12 /customize-image.sh "/images/${CUSTOM_IMAGE_NAME}"

echo "Build complete!"
echo "Custom image is available at: ${IMAGE_DIR}/${CUSTOM_IMAGE_NAME}"
