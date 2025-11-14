# Makefile for building the pvmlab initrd using a containerized build.
# This approach ensures that all dependencies, including shared libraries for tools,
# are correctly captured in the initrd.

INSTALLER_NAME=os-installer
OUTPUT_DIR=./output
BUILD_IMAGE=alpine/git:latest # Using alpine/git for the initrd build environment

# Map Go architecture to Alpine musl architecture
MUSL_ARCH_amd64=x86_64
MUSL_ARCH_arm64=aarch64

# List of binaries to include in the initrd
TOOLS := parted mkfs.ext4 mke2fs sgdisk busybox xz udevd udevadm
# The packages requires to install the binaries above
PACKAGES := parted e2fsprogs dosfstools sgdisk kmod bash ncurses-terminfo-base make git busybox xz eudev hwids

.PHONY: all clean initrd-x86_64 initrd-aarch64

all: clean initrd-x86_64 initrd-aarch64

# Rule to build the x86_64 (amd64) initrd.
initrd-x86_64:
	@echo "==> Building installer for x86_64..."
	@mkdir -p $(OUTPUT_DIR)
	@cd installer && GOWORK=off go mod tidy
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(OUTPUT_DIR)/$(INSTALLER_NAME) ./installer
	@chmod +x ./init
	@echo "==> Building initrd for x86_64..."
	@docker run --rm --platform "linux/amd64" \
		--entrypoint sh \
		-v "$(CURDIR)":/work \
		-w /work \
		$(BUILD_IMAGE) \
		-c "apk add --no-cache $(PACKAGES) cpio gzip && \
		/work/build_custom_initrd.sh \
			x86_64 \
			$(INSTALLER_NAME) \
			$(OUTPUT_DIR) \
			'$(PACKAGES)' \
			'$(TOOLS)'"

# Rule to build the aarch64 (arm64) initrd.
initrd-aarch64:
	@echo "==> Building installer for aarch64..."
	@mkdir -p $(OUTPUT_DIR)
	@cd installer && GOWORK=off go mod tidy
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(OUTPUT_DIR)/$(INSTALLER_NAME) ./installer
	@chmod +x ./init
	@echo "==> Building initrd for aarch64..."
	@docker run --rm --platform "linux/arm64" \
		--entrypoint sh \
		-v "$(CURDIR)":/work \
		-w /work \
		$(BUILD_IMAGE) \
		-c "apk add --no-cache $(PACKAGES) cpio gzip && \
		/work/build_custom_initrd.sh \
			aarch64 \
			$(INSTALLER_NAME) \
			$(OUTPUT_DIR) \
			'$(PACKAGES)' \
			'$(TOOLS)'"

clean:
	@echo "==> Cleaning up..."
	@rm -rf "$(OUTPUT_DIR)"
	@rm -rf installer/vendor
