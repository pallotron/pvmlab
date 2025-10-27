# PXE Boot Implementation Plan

This document outlines the steps to enable target VMs to PXE boot from the provisioner VM over a private network.

## 1. Target VM Configuration (Client-Side)

The QEMU command for launching target VMs needs to be modified to instruct the VM's firmware to attempt a network boot first.

- **File to Modify:** `pvmlab/cmd/vm_start.go`
- **Action:** When the VM role is "target", add the `-boot n` flag to the `qemu-system-aarch64` command.

## 2. Provisioner VM Configuration (Server-Side)

The `pxeboot_stack` Docker container running on the provisioner VM must provide DHCP, TFTP, and HTTP services.

### 2.1. Services Overview

- **DHCP Server:** Assigns an IP address to the target VM and provides the TFTP server address and bootloader filename.
- **TFTP Server:** Serves the initial iPXE bootloader (`ipxe-arm64.efi`).
- **HTTP Server:** Serves the larger OS files (kernel and initrd) for faster transfers.

### 2.2. Implementation with `dnsmasq` and a Web Server

- **DHCP & TFTP:** Both services can be provided by `dnsmasq`.
- **HTTP:** A simple web server (e.g., Nginx, Python's `http.server`) can be added to the container, managed by `supervisord`.

### 2.3. Configuration Files

#### `dnsmasq.conf`

This configuration enables the DHCP and TFTP servers and points the client to the iPXE bootloader and script.

```conf
# Enable TFTP and set the root directory
enable-tftp
tftp-root=/tftpboot

# Point to the iPXE bootloader and chainload the iPXE script
# The IP address should be the static IP of the provisioner VM's private interface
dhcp-boot=ipxe-arm64.efi,,192.168.105.1
pxe-service=ARM64_EFI, "Boot from iPXE", "boot.ipxe"
```

#### `boot.ipxe`

This iPXE script is served by the HTTP server. It instructs the client to download the kernel and initrd, and it passes the necessary kernel parameters to trigger an automated installation.

```ipxe
#!ipxe

# The IP of your provisioner VM's HTTP server
set http_server 192.168.105.1

# Kernel parameters for automated installation
kernel http://${http_server}/vmlinuz console=ttyS0 autoinstall "ds=nocloud-net;s=http://${http_server}/"

initrd http://${http_server}/initrd
boot
```

### 2.4. Required Boot Assets

The following files need to be acquired and placed in the `pxeboot_stack` Docker build context to be included in the final image.

- `ipxe-arm64.efi`: Place in the TFTP root (e.g., `pxeboot_stack/tftpboot/`).
- `vmlinuz`: Place in the HTTP server root (e.g., `pxeboot_stack/www/`).
- `initrd`: Place in the HTTP server root (e.g., `pxeboot_stack/www/`).
- `boot.ipxe`: Place in the HTTP server root (e.g., `pxeboot_stack/www/`).
- `user-data`: Place in the HTTP server root (e.g., `pxeboot_stack/www/`).
- `meta-data`: Place in the HTTP server root (e.g., `pxeboot_stack/www/`).

## 3. Acquiring Kernel and Initrd

You can obtain the required `vmlinuz` (kernel) and `initrd` (initial RAM disk) files by downloading the generic netboot files directly from the Ubuntu repositories.

1. **Create Directories:**

   ```bash
   mkdir -p ubuntu-24.04-amd64-netboot ubuntu-24.04-arm64-netboot
   ```

2. **Download `amd64` (x86_64) Files:**

   ```bash
   # Download the kernel (renaming to vmlinuz)
   wget http://releases.ubuntu.com/24.04/netboot/amd64/linux -O ubuntu-24.04-amd64-netboot/vmlinuz

   # Download the initrd
   wget http://releases.ubuntu.com/24.04/netboot/amd64/initrd -O ubuntu-24.04-amd64-netboot/initrd
   ```

3. **Download `arm64` (AArch64) Files:**

   ```bash
   # Download the kernel (renaming to vmlinuz)
   wget http://cdimage.ubuntu.com/ubuntu/releases/24.04/release/netboot/arm64/linux -O ubuntu-24.04-arm64-netboot/vmlinuz

   # Download the initrd
   wget http://cdimage.ubuntu.com/ubuntu/releases/24.04/release/netboot/arm64/initrd -O ubuntu-24.04-arm64-netboot/initrd
   ```

4. **Verify Downloads:**

   ```bash
   ls -R ubuntu-24.04-amd64-netboot ubuntu-24.04-arm64-netboot
   ```

   _Expected Output:_

   ```text
   ubuntu-24.04-amd64-netboot:
   initrd  vmlinuz

   ubuntu-24.04-arm64-netboot:
   initrd  vmlinuz
   ```

## 4. Automated OS Installation with `autoinstall`

When using the netboot kernel and initrd, you are loading the Debian Installer. For a hands-off installation suitable for `pvmlab`, we use Ubuntu's `autoinstall` mechanism, which is powered by `cloud-init`.

### 4.1. How it Works

1. **Configuration Files:** You create `user-data` and `meta-data` files that contain the answers to all the installer's questions (e.g., disk partitioning, user creation).
2. **HTTP Server:** These files are hosted on the provisioner VM's web server.
3. **Kernel Parameters:** The iPXE script passes a special `autoinstall` parameter to the kernel, telling it where to fetch the configuration files. The installer then runs automatically without user interaction.

### 4.2. Configuration Files

#### `user-data`

This file contains the core autoinstall configuration.

```yaml
#cloud-config
autoinstall:
  version: 1
  locale: en_US
  keyboard:
    layout: en
    variant: us
  network:
    network:
      version: 2
      ethernets:
        enp0s1: # This interface name may vary
          dhcp4: true
  storage:
    layout:
      name: direct
  identity:
    hostname: ubuntu-server
    username: ubuntu
    password: "$6$rounds=4096$..." # A crypted password hash
  packages:
    - openssh-server
  ssh:
    install-server: true
    allow-pw: true
```

> **Note:** You must generate a properly crypted password hash for the `password` field. You can do this with `openssl passwd -6`.

#### `meta-data`

This file is typically minimal and can often be just an empty file, but it must exist.

```yaml
# Empty file
```

### 4.3. Update Boot Assets

Ensure the `user-data` and `meta-data` files are placed in the HTTP server's root directory within the `pxeboot_stack` container, alongside `vmlinuz` and `initrd`.

### 4.4. Per-VM Specific Configuration

The `nocloud-net` datasource allows for both a generic configuration for all booting VMs and specific configurations for individual VMs.

#### Generic Fallback (Default)

The current setup is the simplest case. When the installer is pointed to `s=http://192.168.105.1/`, it fetches `user-data` and `meta-data` from that root directory. Every VM gets the same configuration, which is ideal for creating a cluster of identical nodes.

#### Per-Instance Configuration (Advanced)

For more control, you can create configurations for specific VMs. The installer will first look for a subdirectory named after the VM's **MAC address**.

You can structure your HTTP server's root directory like this:

```shell
/pxeboot_stack/www/
├── user-data                  # Generic fallback user-data
├── meta-data                  # Generic fallback meta-data
│
├── 00:16:3e:11:22:33/           # Directory for a VM with this MAC address
│   ├── user-data              # Specific user-data for this VM
│   └── meta-data              # Specific meta-data for this VM
│
└── 00:16:3e:44:55:66/           # Directory for another VM
    ├── user-data              # Another specific user-data
    └── meta-data
```

**Lookup Process:**

1. A VM with MAC address `00:16:3e:11:22:33` boots.
2. The installer queries the datasource URL.
3. It first attempts to fetch `http://192.168.105.1/00:16:3e:11:22:33/user-data`.
4. **On success**, it uses that specific configuration.
5. **On failure (404 Not Found)**, it falls back to the generic `http://192.168.105.1/user-data`.

This powerful feature allows you to set unique hostnames, static IPs, or SSH keys for each target VM while maintaining a default configuration for all others.
