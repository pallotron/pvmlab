# pvmlab: A Simple QEMU based provisioning lab for macOS

This project provides a command-line tool, `pvmlab`, to automate the setup of a simple virtual provisioning lab on macOS. It uses `QEMU`, `socket_vmnet`, `cloud-init`, and `Docker` to create and manage an environment where multiple "target" VMs are deployed in a private virtual network and one VM functions as the "provisioner" VM offering pxeboot services and working as default gw for internet access for those VMs.

All generated artifacts (VM disks, ISOs, logs, etc.) are stored neatly in `~/.pvmlab/`, keeping the project repository clean.

## Architecture

For a detailed explanation of the project's architecture, features, and VM roles, please see the [Architecture documentation](docs/architecture.md).

## Artifacts Directory

For details on the directory structure used to store generated artifacts, please see the [Artifacts documentation](docs/artifacts.md).

## Getting Started

### Prerequisites

- macOS
- [Homebrew](https://brew.sh/)
- The Go language ecosystem (e.g., `brew install go`)

### Installation for developers

1. **Install Dependencies with Homebrew:**

   The `pvmlab setup` command will check for dependencies, but it will not install them for you. You must install them manually:

   ```bash
   brew install qemu cdrtools socat socket_vmnet docker
   ```

   > **Note:** `docker` refers to the Docker CLI, which is included with Docker Desktop for Mac.

2. **Clone the Repository:**

   ```bash
   git clone https://github.com/pallotron/pvmlab.git
   cd pvmlab
   ```

3. **Build and Install:**

   This command compiles the `pvmlab` CLI, installs the `socket_vmnet` daemons, builds the pxeboot stack Docker container, and sets up shell completion.

   ```bash
   make all
   ```

4. **Set up the Lab Environment:**

   This command creates the `~/.pvmlab/` directory structure and generates an SSH key for accessing the VMs. It may require your `sudo` password to configure `socket_vmnet`.

   ```bash
   pvmlab setup
   ```

5. **Source Shell Completions (Optional but Recommended):**

   The `make all` command attempts to install shell completions. If it doesn't work for your shell, you may need to source them manually.
   Either open a new terminal or reload your shell's rc file.

   For Zsh:

   ```bash
   source ~/.zshrc
   ```

   For Bash:

   ```bash
   source ~/.bashrc
   ```

After installation, you can run the command as `pvmlab` from any directory.

## Usage

**Start the `socket_vmnet` Service:**

This service manages the private virtual network. You may be prompted for your password.

```bash
pvmlab socket_vmnet start
```

**Create the VMs:**

This command downloads cloud images, creates VM disks, and generates cloud-init configurations in the `~/.pvmlab/` directory.

```bash
# Create the provisioner
pvmlab vm create provisioner --role provisioner --ip 192.168.100.1

# Create a target VM
pvmlab vm create target1 --role target --ip 192.168.100.2 --role target
```

**Start the VMs:**

```bash
pvmlab vm start provisioner
pvmlab vm start target1
```

**List VMs:**

```bash
❯ pvmlab vm list
┌─────────────┬─────────────┬───────────────┬──────────────────────────────────┬───────────────────┬─────────┐
│    NAME     │    ROLE     │  PRIVATE IP   │            SSH ACCESS            │        MAC        │ STATUS  │
├─────────────┼─────────────┼───────────────┼──────────────────────────────────┼───────────────────┼─────────┤
│ client      │ target      │ 192.168.100.2 │ 192.168.100.2 (from provisioner) │ 46:cc:46:cd:02:76 │ Stopped │
│ provisioner │ provisioner │ 192.168.100.1 │ localhost:49000                  │ be:f9:b8:75:70:0d │ Running │
└─────────────┴─────────────┴───────────────┴──────────────────────────────────┴───────────────────┴─────────┘
```

**Access the Provisioner VM:**

```bash
pvmlab vm shell provisioner
```

**Access a Target VM:**
First, SSH into the provisioner, then connect to the target's private IP:

```bash
# From your Mac
pvmlab vm shell provisioner
# From inside the provisioner VM, take the IP from dnsmasq (this will improve)
ssh ubuntu@<private IP from DHCPD>
```

[Issue #4](https://github.com/pallotron/pvmlab/issues/4) will make this better.

**Monitor Logs:**

```bash
pvmlab vm logs provisioner
pvmlab vm logs target1
```

**Stopping the VMs:**

```bash
pvmlab vm stop provisioner
pvmlab vm stop target1
```

**Cleanup:**
To stop a VM and remove its files from `~/.pvmlab/`:

```bash
pvmlab vm clean provisioner
```

To clean up everything, including the `socket_vmnet` service and the entire `~/.pvmlab/` directory:

```bash
pvmlab clean
```

### Interacting with the pxeboot Container inside the provisioner VM

For details on how to interact with and manage the pxeboot Docker container, please see the [pxeboot Container documentation](docs/pxeboot_container.md).

## CLI Commands

For a full list of commands and their descriptions, please see the [CLI Reference](docs/cli_reference.md).

## Project Structure

For details on the project's directory structure, please see the [Project Structure documentation](docs/project_structure.md).
