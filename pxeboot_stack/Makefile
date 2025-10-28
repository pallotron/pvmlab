# Makefile for building the provisioning services docker container

IMAGE_NAME ?= pxeboot_stack
IMAGE_TAG ?= latest
DOCKER_IMAGES_DIR ?= $(HOME)/.pvmlab/docker_images

.PHONY: all build download-assets save clean

all: build save

download-assets:
	@mkdir -p www
	@mkdir -p tftpboot
	curl -L -o www/vmlinuz.arm64 http://cdimage.ubuntu.com/ubuntu/releases/24.04/release/netboot/arm64/linux
	curl -L -o www/initrd.arm64 http://cdimage.ubuntu.com/ubuntu/releases/24.04/release/netboot/arm64/initrd
	curl -L -o www/vmlinuz.amd64 https://releases.ubuntu.com/24.04/netboot/amd64/linux
	curl -L -o www/initrd.amd64 https://releases.ubuntu.com/24.04/netboot/amd64/initrd
	curl -L -o tftpboot/ipxe-arm64.efi http://boot.ipxe.org/ipxe-arm64.efi
	curl -L -o tftpboot/ipxe-x86_64.efi http://boot.ipxe.org/ipxe.efi

# Build the docker container
build: download-assets
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

tar: build
	docker save -o $(IMAGE_NAME).tar $(IMAGE_NAME):$(IMAGE_TAG)

# Save the docker container to a tar file
save: build
	@mkdir -p $(DOCKER_IMAGES_DIR)
	docker save -o $(DOCKER_IMAGES_DIR)/$(IMAGE_NAME).tar $(IMAGE_NAME):$(IMAGE_TAG)

clean:
	-docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
