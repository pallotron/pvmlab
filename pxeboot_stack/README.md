# PXE Boot Stack

This directory contains the source code and configuration for the `pxeboot_stack` Docker container. This container provides a complete, self-contained environment for network booting and provisioning virtual machines managed by `pvmlab`.

## Overview

The `pxeboot_stack` container runs a collection of services to handle the entire lifecycle of a VM from initial boot to a fully provisioned operating system. It uses a combination of standard network boot protocols (DHCP, TFTP, iPXE) and custom HTTP services to dynamically configure and install Linux distributions onto VMs based on simple JSON definition files.

## Services

The container uses `supervisord` to manage the following services:

- **dnsmasq**: Provides DHCP, DNS, and TFTP services.
  - **DHCP**: Assigns IP addresses to VMs based on their MAC address. The configuration is dynamically generated from the VM JSON files.
  - **DNS**: Provides local DNS resolution for VMs within the `pvmlab.local` domain.
  - **TFTP**: Serves the initial iPXE bootloader firmware (`.efi` files) to the VMs.
- **nginx**: A web server that acts as a reverse proxy and file server.
  - It serves the OS installation assets (kernels, root filesystems, initrds) from the `/www/images` and `/www/initrds` directories.
  - It proxies dynamic requests (`/ipxe`, `/cloud-init`, `/config`) to the `boot_handler` service.
- **boot\_handler**: A custom Go HTTP server that is the "brains" of the operation.
  - It serves dynamic iPXE boot scripts tailored to each specific VM. When a VM boots, iPXE makes a request to `/ipxe?mac=<mac_address>`. The `boot_handler` finds the corresponding VM JSON file and generates a script that tells the VM which kernel and initrd to download.
  - It provides cloud-init metadata (`/cloud-init/<vm-name>/*`) for post-installation configuration (e.g., setting hostnames, SSH keys).
  - It serves a JSON configuration (`/config/<mac_address>`) to the custom OS installer running in the initrd.
- **vm-watcher**: A shell script that monitors the `/mnt/host/vms` directory for changes to VM definition files. When a file is added, removed, or changed, it triggers `generate_dnsmasq_hosts.sh` and sends a `SIGHUP` to `dnsmasq` to reload its configuration without restarting. This allows for hot-reloading of VM network configurations.

## Boot Process Flow

1. A VM is started by the host hypervisor. Its virtual firmware is configured to PXE boot.
2. The VM broadcasts a DHCP request on the private network.
3. `dnsmasq` in the container receives the request, finds a matching MAC address in its configuration, and replies with an IP address, the TFTP server address, and the name of the iPXE bootloader firmware (`ipxe-x86_64.efi` or `ipxe-arm64.efi`).
4. The VM downloads and executes the iPXE firmware via TFTP.
5. iPXE starts and is configured by `dnsmasq` to download a simple script, `boot.ipxe`, from the TFTP server.
6. `boot.ipxe` contains a single command to chainload a more complex script from the `boot_handler` via HTTP: `chain http://${next-server}/ipxe?mac=${mac}`.
7. The `boot_handler` service receives the request. It looks up the VM's JSON definition file in `/mnt/host/vms` using the provided MAC address.
8. Based on the JSON file, it generates and serves a detailed iPXE script. This script contains logic to download the correct Linux kernel, a custom installer `initrd`, and a separate `modules.cpio.gz` archive containing kernel modules.
9. The VM executes this script, downloads the kernel and both initrd archives into memory, and boots them. The kernel will combine both archives into a single root filesystem.

## Custom Installer Initrd

The `initrd` is a minimal boot environment responsible for preparing the disk and installing the base OS.

- **Build Process**: It is built using a tiny Alpine Linux Docker container to ensure a small footprint and reproducible builds. It includes a statically compiled `os-installer` Go application and common command-line utilities provided by `busybox`.
- **Device and Module Handling**: The initrd uses `udev` to automatically discover hardware devices (like network cards and disk controllers) as they are detected by the kernel. When `udev` finds a new device, it loads the necessary kernel module for it.
- **External Kernel Modules**: To keep the main `initrd` small and flexible, the kernel modules (`.ko` files) are not embedded within it. Instead, they are packaged into a separate `modules.cpio.gz` archive. This archive is downloaded by iPXE alongside the main `initrd` and chainloaded by the kernel, which merges the two archives. This allows the same installer initrd to be used with different kernel versions and module sets. The modules.cpio.gz archive is generated using the `pvmlab distro pull` command

## OS Installation Process

1. The custom installer `initrd` starts, and its `init` script (PID 1) executes the `os-installer` Go application.
2. The `os-installer` fetches its configuration from the `boot_handler`'s `/config/<mac_address>` endpoint. This configuration tells it where to find the OS root filesystem, kernel, etc.
3. It discovers the VM's virtual disk (`/dev/vda` or `/dev/sda`).
4. It partitions and formats the disk (creating an EFI boot partition and a root partition).
5. It downloads the root filesystem tarball (e.g., `rootfs.tar.gz`) from `nginx` and extracts it to the newly created root partition. This is done streaming the tarball over the network using the `tar` command to extract it in real-time and avoid loading the entire tarball into memory.
6. It fetches cloud-init data (`meta-data`, `user-data`, `network-config`) from the `boot_handler` and writes it to `/var/lib/cloud/seed/nocloud-net` on the new filesystem.
7. It installs the GRUB bootloader to the EFI partition and generates a `grub.cfg` file, it also generates the initramfs for GRUB.
8. If the installation is successful, it reboots the VM. On the next boot, the VM starts the newly installed OS from its virtual disk, which then runs cloud-init to perform the final configuration.

## Building the Container

The container can be built for `amd64` and `arm64` architectures using the provided `Makefile`.

- `make all`: Builds the container for both architectures and saves them as tarballs (`pxeboot_stack_amd64.tar`, `pxeboot_stack_arm64.tar`).

The build process involves multiple stages, including:

- compiling the `boot_handler`
- compiling the`os-installer` Go application
- building the custom `initrd` within a containerized environment using a Alpine Linux base image to ensure all dependencies are met and the dimensions are small.
- downloading x86_64 and aarch64 ipxe binaries from boot.ipxe.org and copying them into the container, in the tftp root dir.

If you want to deploy the container on the provisioner vm just run:

```bash
❯ make -C pxeboot_stack all && pvmlab provisioner docker start --docker-tar pxeboot_stack/pxeboot_stack_arm64.tar --network-host --privileged
```
