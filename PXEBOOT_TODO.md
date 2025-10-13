# PXE Boot Implementation Plan

This document outlines the steps required to make the client (target) VMs PXE boot over the private network from the provisioner VM.

## 1. Provisioner VM (Server-Side)

The `pxeboot_stack` Docker container running on the provisioner VM must provide the necessary network services.

### DHCP Server
- **Purpose:** Assign an IP address to the target VM and provide it with the TFTP server address and the bootloader filename.
- **Implementation:** Use `dnsmasq` within the container.
- **Configuration (`pxeboot_stack/dnsmasq.conf`):**
  ```conf
  # Point to the iPXE bootloader
  dhcp-boot=ipxe-arm64.efi
  ```

### TFTP Server
- **Purpose:** Serve the initial iPXE bootloader (`ipxe-arm64.efi`).
- **Implementation:** Use the built-in TFTP server in `dnsmasq`.
- **Configuration (`pxeboot_stack/dnsmasq.conf`):**
  ```conf
  # Enable TFTP and set the root directory
  enable-tftp
  tftp-root=/tftpboot
  ```
- **Action:** The `ipxe-arm64.efi` binary needs to be placed in the TFTP root directory within the Docker image.

### HTTP Server
- **Purpose:** Serve the larger OS files (kernel and initrd) efficiently.
- **Implementation:** Add a simple web server (e.g., Nginx, Python's `http.server`) to the `pxeboot_stack` container. This can be managed by `supervisord`.
- **Action:** The `vmlinuz` and `initrd` files need to be placed in the web server's root directory within the Docker image.

### iPXE Script
- **Purpose:** An iPXE script will be downloaded and executed by the target VM after iPXE starts. This script tells iPXE how to download the kernel and initrd and boot the OS.
- **Implementation:** Create a simple text file (e.g., `boot.ipxe`) in the TFTP/HTTP root.
- **Example `boot.ipxe`:**
  ```ipxe
  #!ipxe
  kernel http://192.168.105.1/vmlinuz console=ttyS0
  initrd http://192.168.105.1/initrd
  boot
  ```
- **`dnsmasq.conf` update:**
  ```conf
  # Chainload the iPXE script after the initial bootloader
  dhcp-boot=ipxe-arm64.efi,,192.168.105.1
  pxe-service=ARM64_EFI, "Boot from iPXE", "boot.ipxe"
  ```

## 2. Target VM (Client-Side)

The QEMU command for launching target VMs needs to be modified.

- **Purpose:** Instruct the VM's firmware to attempt a network boot first.
- **Implementation:** Modify the QEMU command-line arguments in `pvmlab/cmd/vm_start.go`.
- **Action:** When the VM role is "target", add the `-boot n` flag to the `qemu-system-aarch64` command.

## 3. Acquiring Kernel and Initrd

We need to obtain the `vmlinuz` (kernel) and `initrd` (initial RAM disk) files.

### Method 1: Extract from Cloud Image (Recommended)
This ensures version consistency with the disk images `pvmlab` already uses.

1.  **Install `guestmount`:**
    ```bash
    brew install --cask macfuse
    brew install guestmount
    ```
2.  **Mount the Cloud Image:**
    ```bash
    mkdir /tmp/ubuntu-img
    IMAGE_PATH="$HOME/.pvmlab/images/noble-server-cloudimg-arm64.img"
    guestmount -a "$IMAGE_PATH" -m /dev/sda1 --ro /tmp/ubuntu-img
    ```
3.  **Copy Boot Files:**
    ```bash
    cp /tmp/ubuntu-img/boot/vmlinuz-* ./vmlinuz
    cp /tmp/ubuntu-img/boot/initrd.img-* ./initrd
    ```
4.  **Unmount:**
    ```bash
    guestunmount /tmp/ubuntu-img
    ```
5.  **Action:** These `vmlinuz` and `initrd` files should be added to the `pxeboot_stack` directory to be included in the Docker image.

### Method 2: Download from Ubuntu's Netboot Archive
A simpler but potentially less consistent alternative.

- **URL:** `http://ports.ubuntu.com/ubuntu-ports/dists/noble/main/installer-arm64/current/images/netboot/`
- **Commands:**
  ```bash
  wget http://ports.ubuntu.com/ubuntu-ports/dists/noble/main/installer-arm64/current/images/netboot/ubuntu-installer/arm64/linux -O vmlinuz
  wget http://ports.ubuntu.com/ubuntu-ports/dists/noble/main/installer-arm64/current/images/netboot/ubuntu-installer/arm64/initrd.gz -O initrd
  ```
