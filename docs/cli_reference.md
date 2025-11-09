# CLI Commands

This document provides a reference for all `pvmlab` CLI commands.

## `pvmlab setup`

Installs dependencies, creates the artifacts directory, and generates an SSH key pair.

**Usage:**
`pvmlab setup`

**Details:**
This command performs the following actions:

- Checks for required dependencies (`brew`, `mkisofs`, `socat`, `qemu-system-aarch64`, `docker`, `socket_vmnet`).
- Creates the `~/.pvmlab` directory and its subdirectories (`images`, `vms`, `pids`, `logs`, `monitors`, `ssh`, `configs`).
- Generates an RSA key pair for SSH access to the VMs and stores it in `~/.pvmlab/ssh/`.
- Downloads the Ubuntu cloud image if it's not already present.
- Checks the status of the `socket_vmnet` service.

---

## `pvmlab clean`

Stops all VMs and services, and removes generated files.

**Usage:**
`pvmlab clean [--purge]`

**Flags:**

- `--purge`: If set, removes the entire `~/.pvmlab` directory. Otherwise, only the contents of subdirectories are removed.

---

## `pvmlab socket_vmnet`

Manages the `socket_vmnet` background service. This service is required for VMs to have network access.

### `pvmlab socket_vmnet start`

Starts the `socket_vmnet` service. Requires `sudo`.

**Usage:**
`pvmlab socket_vmnet start`

### `pvmlab socket_vmnet stop`

Stops the `socket_vmnet` service. Requires `sudo`.

**Usage:**
`pvmlab socket_vmnet stop`

### `pvmlab socket_vmnet status`

Checks the status of the `socket_vmnet` service.

**Usage:**
`pvmlab socket_vmnet status`

---

## `pvmlab vm`

Manages virtual machines.

### `pvmlab vm create <name>`

Creates a new target VM.

**Usage:**
`pvmlab vm create <name> [flags]`

**Arguments:**

- `<name>`: The name for the new VM.

**Flags:**

- `--mac`: The MAC address for the VM's private network interface. If not provided, a random one is generated.
- `--disk-size`: The size of the VM's disk (e.g., `10G`, `20G`). Defaults to `15G`.
- `--arch`: The architecture of the VM. Can be `aarch64` or `x86_64`. Defaults to `aarch64`.
- `--pxeboot`: If set, creates a VM that boots from the network for installation.
- `--distro`: The distribution for the VM (e.g. `ubuntu-24.04`). Required for `--pxeboot`.

**Example:**

```bash
# Create a target VM that boots from a local disk image
pvmlab vm create my-target

# Create a target VM that will be installed via PXE boot
pvmlab vm create my-pxe-target --pxeboot --distro ubuntu-24.04
```

### `pvmlab provisioner create <name>`

Creates the provisioner VM.

**Usage:**
`pvmlab provisioner create <name> [flags]`

**Arguments:**

- `<name>`: The name for the new provisioner VM.

**Flags:**

- `--ip`: (Required) The static IPv4 address for the VM's private network interface, in CIDR notation (e.g., `192.168.254.1/24`).
- `--ipv6`: The static IPv6 address for the VM's private network interface, in CIDR notation (e.g., `fd00:cafe:babe::1/64`).
- `--mac`: The MAC address for the VM's private network interface. If not provided, a random one is generated.
- `--disk-size`: The size of the VM's disk (e.g., `10G`, `20G`). Defaults to `15G`.
- `--arch`: The architecture of the VM. Can be `aarch64` or `x86_64`. Defaults to `aarch64`.
- `--docker-pxeboot-stack-tar`: Path to a custom `pxeboot_stack.tar` file.
- `--docker-pxeboot-stack-image`: Docker image for the pxeboot stack to pull from a registry.
- `--docker-images-path`: Path to a directory of Docker images to share with the provisioner VM.
- `--vms-path`: Path to a directory of VMs to share with the provisioner VM.

**Example:**

```bash
# Create a provisioner VM
pvmlab provisioner create my-provisioner --ip 192.168.254.1/24
```

### `pvmlab vm start <name>`

Starts the specified VM.

**Usage:**
`pvmlab vm start <name> [flags]`

**Flags:**

- `-i`, `--interactive`: Attach to the VM's serial console for interactive use.
- `--wait`: Wait for the VM's cloud-init process to complete before exiting.
- `--boot`: Override the default boot device. Can be `disk` or `pxe`.

### `pvmlab vm stop <name>`

Stops the specified VM.

**Usage:**
`pvmlab vm stop <name>`

### `pvmlab vm shell <name>`

Opens an SSH session to the specified VM.

**Usage:**
`pvmlab vm shell <name>`

### `pvmlab vm logs <name>`

Tails the console logs for the specified VM.

**Usage:**
`pvmlab vm logs <name>`

### `pvmlab vm list`

Lists all created VMs and their status.

**Usage:**
`pvmlab vm list`

### `pvmlab vm clean <name>`

Stops the VM and deletes its generated files (disk, ISO, logs, etc.).

**Usage:**
`pvmlab vm clean <name>`

---

## `pvmlab provisioner docker`

Manages Docker containers inside a VM.

### `pvmlab provisioner docker start <tar>`

Starts a Docker container inside a VM from a given tarball.

**Usage:**
`pvmlab provisioner docker start --docker-tar <tar> [flags]`

**Arguments:**

- `<vm>`: The name of the VM where the container will run.

**Flags:**

- `--docker-tar`: (Required) Path to the Docker container tarball.
- `--privileged`: Run the container in privileged mode.
- `--network-host`: Use the host's network stack inside the container.

### `pvmlab provisioner docker stop <container>`

Stops a Docker container inside a VM.

**Usage:**
`pvmlab provisioner docker stop <container>`

### `pvmlab provisioner docker status`

Checks the status of Docker containers inside a VM.

**Usage:**
`pvmlab provisioner docker status`
