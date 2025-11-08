# PXE Boot Architecture and Implementation Plan v3: Simplified Boot Flow

This document outlines the final, simplified architecture for a robust, flexible, and distro-agnostic OS installation for `pvmlab` VMs. This approach uses a custom Go-based installation engine and leverages the natural firmware boot order to eliminate complex state management.

## 1. High-Level Architecture

The core of this architecture is a custom `initrd` (initial RAM disk) that contains a Go-based installer. The boot process relies on a "disk first, network second" strategy.

- **Custom `initrd` (The Installer Environment):**
  - Built using the `u-root` framework for a minimal, fast, and reliable boot environment.
  - Contains a statically compiled **Go OS Installer** binary.
  - Includes essential Linux utilities (e.g., `dhclient`, `wget`, `gpt`, `mount`, `tar`, `btrfs`) provided by `u-root`.

- **Go OS Installer (The "Brains"):**
  - A Go application that runs as the main process inside the `initrd`.
  - It fetches a JSON "Installation Profile" from the provisioning server to determine the installation method and OS image.
  - It supports multiple installation methods:
        1. **`btrfs_stream`**: A high-speed method that streams a `btrfs` send stream file directly onto a `btrfs`-formatted partition.
        2. **`tarball`**: A highly compatible method that streams a root filesystem tarball and extracts it onto a formatted partition (e.g., `ext4`, `xfs`).

- **Provisioning Server (`pxEboot_stack`):**
  - A Docker container that runs all the necessary services for PXE booting.
  - **`dnsmasq`**: Provides DHCP and TFTP to serve the iPXE binaries.
  - **`nginx`**: Serves the custom kernel (`vmlinuz`), `initrd`, OS images (tarballs or `btrfs` streams), and acts as a reverse proxy.
  - **`boot_handler`**: A Go server that **statically** serves the iPXE script to load the installer and the JSON installation profile for the requesting VM. It is now stateless regarding the VM's installation status.

- **Offline Build Pipeline:**
  - A `Makefile`-based process to prepare OS images.
  - For `btrfs_stream`, it converts a root filesystem into a compressed `btrfs` send stream file.
  - For `tarball`, it produces a standard compressed root filesystem tarball.

## 2. Boot and Installation Flow

This flow relies on the QEMU boot order being set to **disk first, then network (`-boot order=cn`)**.

1. **First Boot (Blank Disk)**:
    - A new VM is started. The firmware tries to boot from the disk.
    - The disk is blank and has no bootloader, so the boot attempt **fails**.
    - The firmware automatically falls back to the next device: **network boot**.
    - The standard PXE process kicks off: `dnsmasq` -> `iPXE` -> `boot_handler`.
    - The `boot_handler` serves the iPXE script to load our custom installer.
    - The Go installer runs, partitions the disk, installs the OS, and reboots.

2. **Subsequent Boots (Installed OS)**:
    - The VM starts. The firmware tries to boot from the disk.
    - The disk now has a valid bootloader (GRUB). The boot attempt **succeeds**.
    - The VM boots directly into the installed OS. The network boot process is never initiated.

This design elegantly handles re-provisioning: cleaning the VM's disk (`pvmlab vm clean`) naturally causes the next boot to fail over to the network, automatically triggering the installer again.

## 3. Go OS Installer: Detailed Steps

The installer's logic is now simpler as it no longer needs to report its status upon completion.

### Phase 1: Initialization and Configuration

1. **Bring Up Network**:
    - Execute the `dhclient` command provided by `u-root` to get an IP address for the primary network interface (e.g., `eth0`).
    - `exec.Command("dhclient", "-v", "eth0").Run()`

2. **Discover Self**:
    - Read the kernel command line from `/proc/cmdline`.
    - Parse the `installer_mac` parameter from the command line to get the MAC address. This is the MAC address to use for identifying the VM to the provisioning server.

3. **Fetch Installation Profile**:
    - Perform an HTTP GET request to `http://<provisioner_ip>/install-config?mac=<mac_address>`. The provisioner IP is the default gateway provided by DHCP.
    - Parse the JSON response into a Go struct containing all necessary parameters (install method, image URL, filesystem type, hostname, SSH key, etc.).

### Phase 2: Disk Preparation

1. **Find Target Disk**:
    - Identify the primary block device to install to (e.g., `/dev/vda`, `/dev/sda`).
    - A robust strategy is to iterate through `/sys/block` and select the first non-removable, non-loopback device.

2. **Partition Disk**:
    - Wipe any existing partition table.
    - Create a new GPT partition table.
    - Create the necessary partitions by shelling out to `sfdisk`. The partition scheme should support both BIOS and UEFI booting.
    - **Example `sfdisk` script:**

        ```bash
        label: gpt
        device: /dev/vda
        unit: sectors

        1 : size=2048, type=21686148-6449-6E6F-744E-656564454649, name="BIOS boot" # BIOS Boot
        2 : size=1G, type=C12A7328-F81F-11D2-BA4B-00A0C93EC93B, name="EFI System" # EFI System Partition
        3 : type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, name="Linux root" # Linux Root
        ```

    - This script would be passed to `sfdisk /dev/vda < script.txt`.

3. **Format Partitions**:
    - Format the EFI partition as FAT32: `exec.Command("mkfs.vfat", "-F32", "/dev/vda2").Run()`
    - Format the root partition based on the `filesystem_type` from the JSON profile:
        - `exec.Command("mkfs.btrfs", "/dev/vda3").Run()`
        - or `exec.Command("mkfs.ext4", "/dev/vda3").Run()`

### Phase 3: OS Installation

1. **Mount Partitions**:
    - Mount the newly formatted root partition to `/mnt`: `exec.Command("mount", "/dev/vda3", "/mnt").Run()`
    - Create the EFI mount point: `os.MkdirAll("/mnt/boot/efi", 0755)`
    - Mount the EFI partition: `exec.Command("mount", "/dev/vda2", "/mnt/boot/efi").Run()`

2. **Stream and Apply OS Image**:
    - Use a `switch` statement on the `installation_method` from the profile.
    - **Case `btrfs_stream`**:
        - Create the Go pipeline to execute `wget -O - <url> | gunzip | btrfs receive /mnt`.
    - **Case `tarball`**:
        - Create the Go pipeline to execute `wget -O - <url> | tar -xzf - -C /mnt`.

### Phase 4: System Configuration (in `chroot`)

1. **Prepare for `chroot`**:
    - Bind-mount necessary kernel filesystems into the new root so that commands like `grub-install` can function correctly.
    - `exec.Command("mount", "--bind", "/dev", "/mnt/dev").Run()`
    - `exec.Command("mount", "--bind", "/proc", "/mnt/proc").Run()`
    - `exec.Command("mount", "--bind", "/sys", "/mnt/sys").Run()`

2. **Install Bootloader (GRUB)**:
    - Run `grub-install` inside the chroot: `exec.Command("chroot", "/mnt", "grub-install", "--target=x86_64-efi", "--efi-directory=/boot/efi", "--bootloader-id=ubuntu", "--no-nvram").Run()` (adjust target for arch).
    - Generate the GRUB config: `exec.Command("chroot", "/mnt", "grub-mkconfig", "-o", "/boot/grub/grub.cfg").Run()`

3. **Generate `fstab`**:
    - Get the UUIDs of the newly created partitions by parsing the output of `blkid`.
    - Create a new `/mnt/etc/fstab` file and write the correct entries for the root and EFI partitions.

4. **Configure Network**:
    - Create a simple network configuration file in `/mnt/etc/netplan/` (for Ubuntu) or `/mnt/etc/systemd/network/` to enable DHCP on boot.

5. **Configure Hostname and SSH**:
    - Write the `hostname` from the profile to `/mnt/etc/hostname`.
    - Create the user's home directory (e.g., `/mnt/home/ubuntu/.ssh`).
    - Write the public SSH key from the profile to `/mnt/home/ubuntu/.ssh/authorized_keys`.

### Phase 5: Finalization (Simplified)

1. **Unmount Everything**:
    - Unmount the bind-mounted kernel filesystems and the root/EFI partitions in the reverse order they were mounted.

2. **Reboot**:
    - Execute `exec.Command("reboot", "-f").Run()` to restart the machine into its new OS. **No status reporting is needed.**

## 4. Implementation Details and Code

### 4.1. `pvmlab` CLI (`pvmlab/cmd/vm_create.go`)

The `vm create` command, specifically the `handlePxeBootAssets` function, is responsible for preparing all necessary host-side assets.

- It downloads the official distro ISO (e.g., for Ubuntu 24.04).
- It uses `7z` to extract the `vmlinuz` kernel from the ISO.
- It places the extracted kernel in a structured path on the host: `~/.pvmlab/images/<distro>/<arch>/vmlinuz`.
- This makes the correct kernel available to be mounted into the `pxeboot_stack` container at runtime.

### 4.2. `pxeboot_stack` Container

The `pxeboot_stack` container is now completely generic and distro-agnostic.

#### `pxeboot_stack/Dockerfile`

The `Dockerfile`'s sole responsibility for the installer is to build our custom `initrd.gz` and place it in a known static location. It does not handle the kernel.

```dockerfile
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache make
WORKDIR /app

# Build the boot_handler Go application
COPY boot_handler/ ./boot_handler/
RUN make -C boot_handler build

# Build the os-installer and package it into a generic initrd
COPY initrd/ ./initrd/
RUN make -C initrd/ initrd

FROM alpine:3.18
RUN apk update && apk add --no-cache dnsmasq supervisor jq inotify-tools gettext nginx

# Create directories for web content and TFTP
RUN mkdir -p /www/images/installer
RUN mkdir /tftpboot

# Copy static assets
COPY tftpboot/* /tftpboot/
# Note: /www/images/ is populated at runtime by a Docker volume mount from the host
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
# ... other COPY commands

# Copy the boot handler and the generic installer initrd from the builder stage
COPY --from=builder /app/boot_handler/bin/boot_handler /usr/local/bin/boot_handler
COPY --from=builder /app/initrd/output/initrd.gz /www/images/installer/initrd.gz

# ... rest of Dockerfile
```

#### `pxeboot_stack/boot_handler/server.go`

The `boot_handler` is the dynamic component that ties everything together.

```go
// ... VM struct definition ...

func (s *httpServer) ipxeHandler(w http.ResponseWriter, r *http.Request) {
    mac := r.URL.Query().Get("mac")
    vm, err := s.findVMByMAC(mac)
    if err != nil {
        http.Error(w, "VM not found", http.StatusNotFound)
        return
    }

    // Dynamically construct the kernel path based on the VM's metadata
    kernelPath := fmt.Sprintf("http://${next-server}/images/%s/%s/vmlinuz", vm.Distro, vm.Arch)
    // The initrd path is always the same, as it's our generic installer
    initrdPath := "http://${next-server}/images/installer/initrd.gz"

    w.Header().Set("Content-Type", "text/plain")
    fmt.Fprintln(w, "#!ipxe")
    fmt.Fprintf(w, "kernel %s ip=dhcp installer_mac=${mac}\n", kernelPath)
    fmt.Fprintf(w, "initrd %s\n", initrdPath)
    fmt.Fprintln(w, "boot")
}

// ... rest of server.go ...
```

#### `pxeboot_stack/nginx.conf`

Nginx configuration is simplified as it no longer needs to proxy the `/vm-status` endpoint.

```nginx
http {
    server {
        listen 80;

        location /ipxe {
            proxy_pass http://localhost:8080;
        }

        location /install-config {
            proxy_pass http://localhost:8080;
        }

        location / {
            root /www; # Serves kernels, initrds, OS images
            autoindex on;
        }
    }
}
```

### 4.3. VM Start (`pvmlab/cmd/vm_start.go`)

The `vm start` command is simplified to **always** use `-boot order=cn`. The user-facing `--boot` flag is removed as it's no longer necessary for this workflow.

```go
// In buildQEMUArgs function

// ...
// Set the boot order to disk, then network.
// This is the core of the simplified logic.
qemuArgs = append(qemuArgs, "-boot", "order=cn")

// The ISO drive is only attached if the VM was NOT created for PXE boot.
if !opts.meta.PxeBoot {
    qemuArgs = append(qemuArgs, "-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", isoPath))
}
// ...

// In init() for vmStartCmd
// The --boot flag is removed.
// vmCreateCmd.Flags().StringVar(&bootOverride, "boot", "", "Override boot device (disk or pxe)")
```
