# Makefile for building the pvmlab initrd using a containerized u-root build.
# This approach ensures that all dependencies, including shared libraries for tools,
# are correctly captured in the initrd.

INSTALLER_NAME=os-installer
OUTPUT_DIR=./output
GO_VERSION=1.25-alpine
UROOT_SRC_DIR=./u-root

# Map Go architecture to Alpine musl architecture
MUSL_ARCH_amd64=x86_64
MUSL_ARCH_arm64=aarch64

# List of tools to include in the initrd
TOOLS := parted mkfs.ext4 mke2fs mkfs.vfat mkfs.fat sgdisk busybox bash
PACKAGES := parted e2fsprogs dosfstools sgdisk kmod bash ncurses-terminfo-base make git

.PHONY: all clean initrd-x86_64 initrd-aarch64

all: initrd-x86_64 initrd-aarch64

# Rule to build the x86_64 (amd64) initrd.
initrd-x86_64:
	@echo "==> Building initrd for x86_64..."
	@docker run --rm --platform "linux/amd64" \
		-v "$(CURDIR)":/work \
		-w /work \
		golang:$(GO_VERSION) \
		sh -c "apk add --no-cache $(PACKAGES) && \
		/work/build_initrd.sh \
			x86_64 \
			amd64 \
			$(MUSL_ARCH_amd64) \
			$(INSTALLER_NAME) \
			$(OUTPUT_DIR) \
			$(UROOT_SRC_DIR) \
			'$(PACKAGES)' \
			'$(TOOLS)' \
			'$(GO_VERSION)'"

# Rule to build the aarch64 (arm64) initrd.
initrd-aarch64:
	@echo "==> Building initrd for aarch64..."
	@docker run --rm --platform "linux/arm64" \
		-v "$(CURDIR)":/work \
		-w /work \
		golang:$(GO_VERSION) \
		sh -c "apk add --no-cache $(PACKAGES) && \
		/work/build_initrd.sh \
			aarch64 \
			arm64 \
			$(MUSL_ARCH_arm64) \
			$(INSTALLER_NAME) \
			$(OUTPUT_DIR) \
			$(UROOT_SRC_DIR) \
			'$(PACKAGES)' \
			'$(TOOLS)' \
			'$(GO_VERSION)'"

clean:
	@echo "==> Cleaning up..."
	@rm -rf "$(OUTPUT_DIR)"
