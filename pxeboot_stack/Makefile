# Makefile for building the provisioning services docker container

IMAGE_NAME ?= pxeboot_stack
IMAGE_TAG ?= latest
DOCKER_IMAGES_DIR ?= $(HOME)/.provisioning-vm-lab/docker_images

.PHONY: all build save clean

all: build save

# Build the docker container
build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Save the docker container to a tar file
save: build
	@mkdir -p $(DOCKER_IMAGES_DIR)
	docker save -o $(DOCKER_IMAGES_DIR)/$(IMAGE_NAME).tar $(IMAGE_NAME):$(IMAGE_TAG)

clean:
	-docker rmi $(IMAGE_NAME):$(IMAGE_TAG)