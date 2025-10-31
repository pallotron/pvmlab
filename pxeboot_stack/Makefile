# Makefile for building the provisioning services docker container

IMAGE_NAME ?= pxeboot_stack
IMAGE_TAG ?= latest
DOCKER_IMAGES_DIR ?= $(HOME)/.pvmlab/docker_images

.PHONY: all build download-assets save clean initrds

all: build save

download-assets:
	@mkdir -p www
	@mkdir -p tftpboot
	echo "==> Downloading iPXE bootloader binaries..."
	curl -L -o tftpboot/ipxe-arm64.efi http://boot.ipxe.org/ipxe-arm64.efi
	curl -L -o tftpboot/ipxe-x86_64.efi http://boot.ipxe.org/ipxe.efi

initrds:
	make -C initrd all

# Build the docker container
build: download-assets initrds
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE_NAME):$(IMAGE_TAG) --load .

tar: build
	docker save -o $(IMAGE_NAME).tar $(IMAGE_NAME):$(IMAGE_TAG)

# Save the docker container to a tar file
save: build
	@mkdir -p $(DOCKER_IMAGES_DIR)
	docker save -o $(DOCKER_IMAGES_DIR)/$(IMAGE_NAME).tar $(IMAGE_NAME):$(IMAGE_TAG)

clean:
	-docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
	rm -f pxeboot_stack.tar
	make -C initrd clean
