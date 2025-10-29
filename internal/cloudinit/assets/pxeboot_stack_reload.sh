#!/bin/bash
set -e
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <tar_file> <container_name> [docker_run_flags...]" >&2
    exit 1
fi
TAR_FILE=$1
CONTAINER_NAME=$2
shift 2
DOCKER_RUN_FLAGS="$@"
echo "Stopping and removing old container..."
docker stop ${CONTAINER_NAME} || true
docker rm ${CONTAINER_NAME} || true
echo "Loading new image from /mnt/host/docker_images/${TAR_FILE}..."
IMAGE_NAME=$(docker load -i /mnt/host/docker_images/${TAR_FILE} | grep "Loaded image:" | awk '{print $3}')
if [ -z "$IMAGE_NAME" ]; then
    echo "Failed to get image name from docker load output." >&2
    exit 1
fi
echo "Loaded image: ${IMAGE_NAME}"
export PROVISIONER_IP={{ ds.meta_data.provisioner_ip }}
export DHCP_RANGE_START={{ ds.meta_data.dhcp_range_start }}
export DHCP_RANGE_END={{ ds.meta_data.dhcp_range_end }}
{% if ds.meta_data.dhcp_range_v6_start %}
export DHCP_RANGE_V6_START={{ ds.meta_data.dhcp_range_v6_start }}
export DHCP_RANGE_V6_END={{ ds.meta_data.dhcp_range_v6_end }}
{% endif %}
echo "Starting new container..."
docker run --mount type=bind,source=/mnt/host/vms,target=/mnt/host/vms -d \
  -e PROVISIONER_IP=$PROVISIONER_IP \
  -e DHCP_RANGE_START=$DHCP_RANGE_START \
  -e DHCP_RANGE_END=$DHCP_RANGE_END \
  {% if ds.meta_data.dhcp_range_v6_start %}
  -e DHCP_RANGE_V6_START=$DHCP_RANGE_V6_START \
  -e DHCP_RANGE_V6_END=$DHCP_RANGE_V6_END \
  {% endif %} \
  --name ${CONTAINER_NAME} ${DOCKER_RUN_FLAGS} ${IMAGE_NAME}
echo "Done."
