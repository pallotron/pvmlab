# Makefile for building the provisioning services docker container

IMAGE_NAME ?= pxeboot_stack
IMAGE_TAG ?= latest
DOCKER_IMAGES_DIR ?= $(CURDIR)

.PHONY: all save save-amd64 save-arm64 download-assets clean initrds build-boot-handler

all: save

save: save-amd64 save-arm64

download-assets:
	@mkdir -p tftpboot
	chmod 755 tftpboot/
	echo "==> Downloading iPXE bootloader binaries..."
	curl -L -o tftpboot/ipxe-arm64.efi http://boot.ipxe.org/ipxe-arm64.efi
	curl -L -o tftpboot/ipxe-x86_64.efi http://boot.ipxe.org/ipxe.efi
	chmod 644 tftpboot/*

initrds:
	make -C initrd all

build-boot-handler:
	@echo "==> Building boot_handler..."
	@cd boot_handler && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/boot_handler_amd64 .
	@cd boot_handler && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/boot_handler_arm64 .

# Build and save the amd64 docker container to a tar file
save-amd64: download-assets initrds build-boot-handler
	@mkdir -p $(DOCKER_IMAGES_DIR)
	@echo "==> Building and saving linux/amd64 image to $(DOCKER_IMAGES_DIR)/$(IMAGE_NAME)_amd64.tar"
	docker buildx build \
		--platform linux/amd64 \
		-t $(IMAGE_NAME):$(IMAGE_TAG)-amd64 \
		--output type=docker,dest=$(DOCKER_IMAGES_DIR)/$(IMAGE_NAME)_amd64.tar \
		.

# Build and save the arm64 docker container to a tar file
save-arm64: download-assets initrds build-boot-handler
	@mkdir -p $(DOCKER_IMAGES_DIR)
	@echo "==> Building and saving linux/arm64 image to $(DOCKER_IMAGES_DIR)/$(IMAGE_NAME)_arm64.tar"
	docker buildx build \
		--platform linux/arm64 \
		-t $(IMAGE_NAME):$(IMAGE_TAG)-arm64 \
		--output type=docker,dest=$(DOCKER_IMAGES_DIR)/$(IMAGE_NAME)_arm64.tar \
		.

clean:
	-docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
	rm -f pxeboot_stack.tar
	make -C initrd clean
