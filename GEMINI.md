# Plan for Go-based CLI

This document outlines the plan to replace the existing `Makefile` with a
Go-based command-line interface (CLI) for managing the provisioning VM lab. This
change will improve maintainability, portability, and user experience.

## 1. Project Structure Changes

All project-generated artifacts will be moved from the project's root directory
to a dedicated hidden directory in the user's home directory to keep the project
repository clean.

- **New Artifacts Directory:** `~/.provisioning-vm-lab/`
- **This directory will contain:**
  - `vms/`: for VM disk images (`.qcow2` files)
  - `configs/`: for generated ISO images (`.iso` files)
  - `images/`: for downloaded cloud images
  - `logs/`: for VM console logs
  - `pids/`: for VM process IDs
  - `monitors/`: for VM monitor sockets
  - `ssh/`: for generated SSH keys (`vm_rsa`, `vm_rsa.pub`)
  - `uuidgen`: for the generated UUID

The main project directory will now contain the Go source code for the CLI.

- **New Go CLI Directory:** `cmd/pvmlab/`
- **Go module file:** `go.mod`

## 2. Go CLI Implementation

A new CLI tool named `pvmlab` will be created using the `cobra` library to
provide a structured and user-friendly command interface. The commands will
mirror the functionality of the existing `Makefile` targets.

### CLI Command Structure:

```shell
pvmlab <command> <subcommand> [flags]
```

### 2.1. Dynamic Configuration and Target Management

The CLI will include logic for dynamically generating cloud-init configuration
files and managing target VMs.

- **Cloud-Init Rendering:**
  - The CLI will render `user-data` and `meta-data` files for cloud-init.
  - The `meta-data` will be populated with the public key from the generated
    SSH key pair, enabling secure access to the VMs.
- **Target VM Management:**
  - Subcommands will be available to `add` and `remove` target VMs.
  - The `add` command will include an optional `--mac` flag to specify a MAC
    address for the VM.

### Proposed Commands

- **`pvmlab setup`**:

  - Checks for and installs dependencies (Homebrew, `cdrtools`, `socat`,
    `socket_vmnet`, `qemu`).
  - Creates the `~/.provisioning-vm-lab/` directory structure.
  - Generates the SSH key pair and saves it to
    `~/.provisioning-vm-lab/ssh/`.
  - Make sure launchd is configured to launch the socket_vmnet service.
    `sudo ${HOMEBREW_PREFIX}/bin/brew services start socket_vmnet`

- **`pvmlab service start`**:

  - Starts the `socket_vmnet` service using `brew services`.

- **`pvmlab service stop`**:

  - Stops the `socket_vmnet` service.

- **`pvmlab provisioner create`**:

  - Downloads the aarch64 Ubuntu cloud image to
    `~/.provisioning-vm-lab/images/`.
  - Creates and resizes the provisioner VM disk.
  - Generates the cloud-config ISO, ensuring `user-data` is configured to
    support running Docker containers.

- **`pvmlab provisioner start`**:

  - Starts the provisioner VM using `qemu`.

- **`pvmlab provisioner stop`**:

  - Stops the provisioner VM gracefully.

- **`pvmlab provisioner shell`**:

  - Connects to the provisioner VM via SSH.

- **`pvmlab provisioner logs`**:

  - Tails the provisioner VM console logs.

- **`pvmlab provisioner clean`**:

  - Stops the VM and removes all its associated files (disk, iso, pid,
    monitor).

- **`pvmlab target add [flags]`**:

  - Downloads the amd64 Ubuntu cloud image.
  - Creates and resizes the target VM disk.
  - Generates the cloud-config ISO.
  - Optional `--mac` flag to specify a MAC address.

- **`pvmlab target remove <vm-name>`**:

  - Removes a specified target VM and its associated files.

- **`pvmlab target start <vm-name>`**:

  - Starts the target VM.

- **`pvmlab target stop <vm-name>`**:

  - Stops the target VM.

- **`pvmlab target shell <vm-name>`**:

  - Connects to the target VM... This is tricky because the VM is not
    reacheable from the mac... need to find a solution.
    Set up an ssh port forward that goes thru the VM? any suggestion?

- **`pvmlab target logs <vm-name>`**:

  - Tails the target VM console logs.

- **`pvmlab target clean <vm-name>`**:

  - Stops the VM and removes its files.

- **`pvmlab clean`**:

  - Runs `clean` for both `provisioner` and `target`.
  - Stops the `socket_vmnet` service.
  - Optionally, removes the entire `~/.provisioning-vm-lab/` directory.

- **`pvmlab status`**:

  - Displays the status of all VMs (running, stopped, etc.).

## 3. Benefits of this Approach

- **Clean Project Directory:** The project's root directory will no longer be
  cluttered with generated files.
- **Improved Maintainability:** Go code is easier to read, test, and maintain
  than complex `Makefile` logic.
- **Cross-Platform Portability:** A compiled Go binary can run on different
  systems without requiring `make` or specific shell environments (though
  dependencies like Homebrew and QEMU will still be required).
- **Better Error Handling:** Go provides robust error handling, making the tool
  more reliable.
- **Extensibility:** Adding new commands, flags, and logic is much simpler
  with a proper CLI structure.
- **User-Friendly:** A well-structured CLI with help messages is easier for
  users to understand and use.

## 4. Implementation Steps

1. Create `GEMINI.md` with this plan.
2. Initialize a Go module: `go mod init <module_name>`.
3. Add the `cobra` dependency: `go get -u github.com/spf13/cobra`.
4. Create the initial CLI structure in `cmd/pvmlab/main.go`.
5. Implement each command, starting with `setup` and `service`.
6. Refactor the logic from the `Makefile` into the corresponding Go functions.
7. Update `.gitignore` to exclude the compiled binary and other Go-related
   files.
8. Update `README.md` with instructions on how to build and use the new
   `pvmlab` CLI.
9. (Optional) Remove the `Makefile`.
