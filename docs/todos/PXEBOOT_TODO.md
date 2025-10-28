# PXE Boot Architecture and Implementation Plan

This document outlines the architecture for enabling distro-agnostic, automated OS installation for `pvmlab` VMs via PXE boot.

## 1. High-Level Architecture

The PXE boot environment is designed to be dynamic, flexible, and scalable. It separates the concerns of asset management, dynamic configuration, and static file serving.

- **`pvmlab` Application (The Controller):** The main `pvmlab` binary is responsible for all state management and asset preparation. It manages VM definitions, downloads OS ISOs, extracts the required kernel/initrd assets, and places them in a local cache (`~/.pvmlab/images`).
- **`pxeboot_stack` Container (The Service):** This container runs the necessary network services to boot a VM. It is designed to be generic and stateless, with all configuration and OS assets mounted from the host.

The container employs a hybrid web server model:

- **Dynamic Go Server:** A lightweight Go application provides the "brains" of the operation. It dynamically generates iPXE boot scripts for each VM based on its JSON definition file.
- **Nginx Static Server:** A high-performance Nginx server is used to serve large, static files like OS kernels, initrds, and ISOs.

## 2. Boot Flow

1. A new VM is started with `pvmlab vm start`. QEMU is configured to attempt a network boot.
2. The VM's UEFI firmware sends a DHCP request.
3. `dnsmasq` (in the `pxeboot_stack` container) assigns an IP address and tells the VM to download a generic iPXE binary (`ipxe-*.efi`) via TFTP.
4. The iPXE firmware starts and runs an embedded script that chainloads to the dynamic Go server, passing the VM's MAC address as a parameter (e.g., `http://192.168.100.1/boot?mac=${net0/mac}`).
5. The Go server receives the request. It finds the corresponding VM's JSON file in the mounted `/mnt/host/vms` directory by matching the MAC address.
6. It reads the VM's configuration (e.g., `arch`, `distro`) and generates a custom iPXE script using a Go template.
7. The custom script is returned to the VM. It contains the precise URLs for the required kernel and initrd, pointing to the Nginx server.
8. The VM's iPXE client downloads the kernel and initrd from Nginx and boots the OS installer with the correct `autoinstall` or `kickstart` parameters.

## 3. Known Issues

### 3.1. `aarch64` PXE Boot

Currently, enabling PXE boot on `aarch64` VMs in QEMU is problematic. The `-boot n` flag does not appear to be reliably supported or may require a specific combination of firmware and machine type that has not yet been identified. `x86_64` PXE booting, however, is functional.

## 4. Implementation Action Items

1. **Asset Management in `pvmlab`:**

    - Implement logic within the `pvmlab` application to download OS ISOs (starting with Ubuntu 24.04 Live Server).
    - Add functionality to extract `vmlinuz` and `initrd` from the ISO's `/casper` directory.
    - Store these assets in a structured way, e.g., `~/.pvmlab/images/ubuntu-24.04/`.
    - Modify the `pvmlab vm create` command to include a `"distro"` field in the VM's JSON definition.

2. **`pxeboot_stack` Container Implementation:**

    - **Create Go Server:** Implement the dynamic boot script server (`server.go`).
    - **Create Go Template:** Create a Go template for the iPXE script (`boot.ipxe.go.template`) that uses variables from the VM JSON file.
    - **Integrate Services:**
      - Update `nginx.conf` to act as a reverse proxy, forwarding `/boot` requests to the Go server, while serving all other requests from the `/www` directory.
      - Update `supervisord.conf` to manage both the `nginx` and `go-server` processes.
    - **Update `dnsmasq.conf`:** Simplify the configuration to point all clients to a generic iPXE chainloader script.
    - **Update `Dockerfile`:**
      - Add the Go server source and build it into a binary.
      - Ensure the iPXE binaries (`.efi` files) are copied to the TFTP root.
      - Set the container's `CMD` to run `supervisord`.

3. **VM Start Configuration:**
    - Ensure the `pxeboot_stack` container is started with the necessary volume mounts:
      - `~/.pvmlab/vms` -> `/mnt/host/vms`
      - `~/.pvmlab/images` -> `/www/images`
