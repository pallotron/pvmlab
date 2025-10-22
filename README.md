<p align="center">
  <img src="images/pvmlab_logo.png" alt="pvmlab logo" width="500"/>
</p>

# pvmlab: your local pxeboot virtual lab

This project provides a command-line tool, `pvmlab`, to automate the setup of a simple virtual provisioning lab on macOS. It uses `QEMU`, `socket_vmnet`, `cloud-init`, and `Docker` to create and manage an environment where multiple "target" VMs are deployed in a private virtual network and one VM functions as the "provisioner" VM offering pxeboot services and working as default gw for internet access for those VMs.

All generated artifacts (VM disks, ISOs, logs, etc.) are stored neatly in `~/.pvmlab/`, keeping the project repository clean.

## Use Cases

This virtual lab is ideal for a variety of network-based provisioning and testing scenarios, such as:

- **Testing OS Installers:** Develop and test automated OS installation configurations like Ubuntu Autoinstall, Debian Preseed, or Red Hat Kickstart. Or test your own ;)
- **Developing Network Boot Environments:** Experiment with and develop iPXE scripts or other network boot setups.
- **Simulating Bare-Metal Provisioning:** Mimic the process of provisioning physical servers in a fully virtualized, local environment.
- **Network Service Testing:** Safely test DHCP, TFTP, and HTTP services in an isolated network.
- **Playing with Kubernetes or Other Orchestration Tools:** Use the lab to experiment with cluster provisioning and management tools like kubernetes, Tinkerbell, Canonical MAAS, etc.
- **Learning and Experimentation:** Provide a hands-on environment for learning about PXE boot, network automation, and cloud-init.

## Core Concepts

### Features

- **Automated VM Creation:** Quickly create Ubuntu VMs from cloud images.
- **Provisioner/Target Architecture:** Set up a dedicated "provisioner" VM to serve network resources (DHCP, PXE boot) to multiple "target" VMs.
- **Private Networking:** Uses `socket_vmnet` to create an isolated virtual network for your lab environment.
- **Dual-stack Networking:** Configure VMs with both IPv4 and IPv6 addresses on the private network.
- **Direct SSH Access:** Connect directly to any VM (`provisioner` or `target`) with a single command.
- **Simple CLI:** Manage the entire lab lifecycle with intuitive `pvmlab` commands.

### Architecture

For a detailed explanation of the project's architecture, features, and VM roles, please see the [Architecture documentation](docs/architecture.md).

## Getting Started

### Installation

1. **Install Dependencies with Homebrew:**
   The `pvmlab setup` command will check for dependencies, but it will not install them for you. You must install them manually:

   ```bash
   brew install qemu cdrtools socat socket_vmnet docker
   ```

   > **Note:** `docker` refers to the Docker CLI, which is included with Docker Desktop for Mac.

2. **Clone the Repository:**

   ```bash
   git clone --recurse-submodules https://github.com/pallotron/pvmlab.git
   cd pvmlab
   ```

3. **Build and Install:**
   This command compiles the `pvmlab` CLI, installs the `socket_vmnet` daemons, builds the pxeboot stack Docker container, and sets up shell completion.

   ```bash
   make install
   ```

4. **Set up the Lab Environment:**
   This command creates the `~/.pvmlab/` directory structure and generates an SSH key for accessing the VMs. It may require your `sudo` password to configure `socket_vmnet`.

   ```bash
   pvmlab setup
   ```

5. **Source Shell Completions (Optional but Recommended):**
   The `make all` command attempts to install shell completions. If it doesn't work for your shell, you may need to source them manually. Either open a new terminal or reload your shell's rc file.
   - For Zsh: `source ~/.zshrc`
   - For Bash: `source ~/.bashrc`

After installation, you can run the command as `pvmlab` from any directory.

## Usage: A Typical Workflow

Here is a step-by-step guide to a common workflow.

### Step 1: Start the Network Service

This service manages the private virtual network. You may be prompted for your password.
The `make install` command should have already set this up for you, but run this commands to ensure.

```bash
pvmlab socket_vmnet status
# if not running, start it:
pvmlab socket_vmnet start
```

### Step 2: Create the VMs

This command downloads cloud images, creates VM disks, and generates cloud-init configurations.

```bash
# Create the provisioner
pvmlab vm create provisioner --role provisioner --ip 192.168.100.1/24 --ipv6 fd00:cafe:babe::1/64

# Create a target VM
pvmlab vm create client1 --role target --ip 192.168.100.2/24 --ipv6 fd00:cafe:babe::2/64
```

You can create `x86_64` or `aarch64` VMs by specifying the `--arch` flag. By default, `aarch64` is used.
You can also use `--disk` or `--pxeboot` flags to customize how the VM should boot.
This is still a work in progress.

### Step 3: Manage the VMs

Once created, you can start, stop, and interact with your VMs.

**Start VMs:**

```bash
pvmlab vm start provisioner
pvmlab vm start client1
```

The start command accepts `--interactive` mode, which attaches your terminal to the VM's console.
You can provide the `--wait` flag to block the terminal until the VM has reached `cloud-init.target`.
Providing no flags starts the VM in the background. You can monitor logs via the `pvmlab vm logs` command (see below).

**List VMs:**

```bash
❯ pvmlab vm list
┌─────────────┬─────────┬───────────┬───────────────┬───────────────────┬───────────────────┬─────────┐
│    NAME     │  ARCH   │ BOOT TYPE │  PRIVATE IP   │   PRIVATE IPV 6   │        MAC        │ STATUS  │
├─────────────┼─────────┼───────────┼───────────────┼───────────────────┼───────────────────┼─────────┤
│ client1     │ x86_64  │ disk      │ 192.168.100.2 │ fd00:cafe:babe::2 │ 0a:5d:ea:18:55:b6 │ Running │
│ provisioner │ aarch64 │ disk      │ 192.168.100.1 │ fd00:cafe:babe::1 │ 62:a8:6a:f6:5b:3c │ Running │
└─────────────┴─────────┴───────────┴───────────────┴───────────────────┴───────────────────┴─────────┘
```

**Access a VM's Shell:**

```bash
# Access the Provisioner VM
pvmlab vm shell provisioner

# Access a Target VM
pvmlab vm shell client1
```

**Monitor Logs:**

```bash
pvmlab vm logs provisioner
pvmlab vm logs client1
```

**Stop VMs:**

```bash
pvmlab vm stop provisioner
pvmlab vm stop client1
```

### Step 4: Clean Up

You can clean up individual VMs or the entire lab environment.

**Clean a single VM:**
This stops the VM and removes all its associated files from `~/.pvmlab/`.

```bash
pvmlab vm clean provisioner
```

**Clean the entire lab:**
This removes everything, including the `socket_vmnet` service and the `~/.pvmlab/` directory.

```bash
pvmlab clean
```

## Artifacts Directory

All files generated by `pvmlab` are stored in a hidden directory in your home folder to keep the project's working directory clean.

The structure of this directory is as follows:

```shell
~/.pvmlab/
├── configs/        # Generated cloud-init ISO files (.iso) for each VM
├── docker_images/  # Docker images saved as .tar files to be shared with the provisioner VM
├── images/         # Downloaded Ubuntu cloud image templates
├── logs/           # VM console logs
├── monitors/       # QEMU monitor sockets for interacting with the hypervisor
├── pids/           # Process ID files for running VMs
├── ssh/            # Generated SSH key pair (vm_rsa, vm_rsa.pub) for VM access
└── vms/            # VM disk images (.qcow2) created from the base images
```

## Missing Features

Missing features are being tracked as issues in the [GitHub repository](https://github.com/pallotron/pvmlab/issues). Please feel free to contribute!

## Advanced Topics & Further Reading

For more detailed information, please refer to the documentation in the `docs/` directory:

- **[CLI Command Reference](docs/cli_reference.md):** A full list of all commands and their flags.
- **[Architecture](docs/architecture.md):** A detailed explanation of the project's design and components.
- **[Interacting with the pxeboot Container](docs/pxeboot_container.md):** How to manage the pxeboot Docker container inside the provisioner VM.
