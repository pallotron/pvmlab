#!/bin/bash
set -e

# The image to pull is passed in from cloud-init.
IMAGE_TO_PULL="{{ ds.meta_data.pxeboot_stack_image }}"

# --- Argument Parsing ---
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <tar_file_name | ''> <container_name> [docker_run_flags...]" >&2
    exit 1
fi

TAR_FILE_ARG=$1
CONTAINER_NAME=$2
shift 2
DOCKER_RUN_FLAGS="$@"

# --- Main Logic ---
echo "Stopping and removing old container..."
docker stop "${CONTAINER_NAME}" 2>/dev/null || true
docker rm "${CONTAINER_NAME}" 2>/dev/null || true

FINAL_IMAGE_NAME="${CONTAINER_NAME}:latest"
TAR_FILE_PATH="/mnt/host/docker_images/${TAR_FILE_ARG}"

# Decide whether to load from tarball or pull from registry
if [ -n "$TAR_FILE_ARG" ] && [ -f "$TAR_FILE_PATH" ]; then
    # --- Load from Local Tarball ---
    echo "Loading new image from ${TAR_FILE_PATH}..."
    LOAD_OUTPUT=$(docker load -i "${TAR_FILE_PATH}")
    echo "Docker load output: $LOAD_OUTPUT"

    # Extract the image name from the "Loaded image: ..." line for single-platform tarballs
    # or the manifest name for multi-platform tarballs (which docker load handles differently)
    LOADED_IMAGE_NAME=$(echo "$LOAD_OUTPUT" | grep -oP "(?<=Loaded image: ).*" || echo "$LOAD_OUTPUT" | grep -oP "(?<=Loaded image list for: ).*")

    if [ -z "$LOADED_IMAGE_NAME" ]; then
        echo "Failed to get image name from docker load output." >&2
        exit 1
    fi

    echo "Loaded image: $LOADED_IMAGE_NAME"
    
    # If it's a manifest list, we need to tag the specific architecture image
    if echo "$LOAD_OUTPUT" | grep -q "Loaded image list for:"; then
        ARCH=$(uname -m)
        if [ "$ARCH" = "x86_64" ]; then
            TARGET_ARCH="amd64"
        elif [ "$ARCH" = "aarch64" ]; then
            TARGET_ARCH="arm64"
        else
            echo "Unsupported architecture: $ARCH" >&2
            exit 1
        fi
        IMAGE_DIGEST=$(docker manifest inspect "$LOADED_IMAGE_NAME" | jq -r ".manifests[] | select(.platform.architecture == \"$TARGET_ARCH\") | .digest")
        if [ -z "$IMAGE_DIGEST" ]; then
            echo "Failed to find a matching image for architecture $TARGET_ARCH in the manifest." >&2
            exit 1
        fi
        docker tag "$IMAGE_DIGEST" "$FINAL_IMAGE_NAME"
        echo "Tagged $IMAGE_DIGEST as $FINAL_IMAGE_NAME"
    else
        # Single platform image, just tag it
        docker tag "$LOADED_IMAGE_NAME" "$FINAL_IMAGE_NAME"
        echo "Tagged $LOADED_IMAGE_NAME as $FINAL_IMAGE_NAME"
    fi

else
    # --- Pull from Registry ---
    if [ -z "$IMAGE_TO_PULL" ]; then
        echo "Error: No local tarball was found and no image was specified to pull from the registry." >&2
        exit 1
    fi
    
    echo "No local tarball provided or found. Pulling image from registry: ${IMAGE_TO_PULL}"
    
    docker pull "${IMAGE_TO_PULL}"
    
    # Docker pull automatically selects the correct architecture.
    # We just need to re-tag it for consistency with the rest of the script.
    docker tag "${IMAGE_TO_PULL}" "${FINAL_IMAGE_NAME}"
    echo "Tagged ${IMAGE_TO_PULL} as ${FINAL_IMAGE_NAME}"
fi

# --- Environment and Container Start ---
export PROVISIONER_IP={{ ds.meta_data.provisioner_ip }}
export DHCP_RANGE_START={{ ds.meta_data.dhcp_range_start }}
export DHCP_RANGE_END={{ ds.meta_data.dhcp_range_end }}
{% if ds.meta_data.dhcp_range_v6_start %}
export DHCP_RANGE_V6_START={{ ds.meta_data.dhcp_range_v6_start }}
export DHCP_RANGE_V6_END={{ ds.meta_data.dhcp_range_v6_end }}
{% endif %}

echo "Starting new container..."
docker run --mount type=bind,source=/mnt/host/vms,target=/mnt/host/vms \
  --mount type=bind,source=/mnt/host/images,target=/www/images -d \
  -e PROVISIONER_IP=$PROVISIONER_IP \
  -e DHCP_RANGE_START=$DHCP_RANGE_START \
  -e DHCP_RANGE_END=$DHCP_RANGE_END \
  {% if ds.meta_data.dhcp_range_v6_start %} \
  -e DHCP_RANGE_V6_START=$DHCP_RANGE_V6_START \
  -e DHCP_RANGE_V6_END=$DHCP_RANGE_V6_END \
  {% endif %} \
  --name "${CONTAINER_NAME}" ${DOCKER_RUN_FLAGS} "${FINAL_IMAGE_NAME}"

echo "Done."
