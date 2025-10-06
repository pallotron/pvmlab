# Makefile for building the provisioning services docker container

IMAGE_NAME ?= provisioning-services
IMAGE_TAG ?= latest

.PHONY: all build clean

all: build

# Build the docker container
build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Remove the docker container
clean:
	-docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
