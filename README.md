# QEMU Provisioning Lab for macOS

This project automates the setup of a simple virtual provisioning lab on macOS using QEMU, `socket_vmnet`, `cloud-init`, and a `Makefile`. The lab consists of two Ubuntu Linux VMs: a provisioning server and a target device, connected via a private host-only network.

The provisioning server is configured to act as a network gateway, providing internet access to the target VM, simulating a real-world provisioning environment where the target machine is on an isolated network.

## Features

- **Fully Automated:** Uses a `Makefile` to automate the entire setup, execution, and teardown process.
- **Two-VM Architecture:**
    - **Provisioning VM:** An `aarch64` Ubuntu server that provides network services.
    - **Target VM:** An `x86_64` Ubuntu server that is configured by the lab.
- **Isolated Provisioning Network:** Utilizes `socket_vmnet` to create a private host-only network between the VMs, allowing them to communicate without external access.
- **Internet Access via NAT:** The provisioning VM has access to the internet and acts as a gateway for the target VM, forwarding its traffic using NAT.
- **Declarative VM Configuration:** Uses `cloud-init` to declaratively configure both VMs on first boot, including:
    - Setting hostnames and static IP addresses.
    - Creating user accounts and setting passwords.
    - Installing packages (`iptables-persistent`).
    - Configuring the provisioning VM as a network gateway.

## Architecture

The lab is composed of two QEMU virtual machines connected to a private virtual network.

1.  **Provisioning VM (`provisioner-vm`):**
    - **OS:** Ubuntu Server 24.04 (aarch64)
    - **Network Interfaces:**
        - `enp0s1` (WAN): Connects to the internet via QEMU's `user` network (NAT).
        - `enp0s2` (LAN): Connects to the private `vmnet-host` network with a static IP of `192.168.100.1`.
    - **Services:** Configured via `cloud-init` to:
        - Set the password for the `ubuntu` and `root` users to `pass`.
        - Install `iptables-persistent` to save firewall rules.
        - Enable IP forwarding and configure NAT to provide internet access to the target VM.

2.  **Target VM (`target-vm`):**
    - **OS:** Ubuntu Server 24.04 (x86_64)
    - **Network Interface:**
        - `eth0`: Connects to the private `vmnet-host` network with a static IP of `192.168.100.2`.
    - **Configuration:** Configured via `cloud-init` to use the provisioning VM (`192.168.100.1`) as its default gateway.

## Prerequisites

- macOS
- [Homebrew](https://brew.sh/)
- QEMU (`qemu`)
- CDRTools (`cdrtools`)
- `socat`
- `socket_vmnet`

## Setup

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd <repository-directory>
    ```

2.  **Install Dependencies & Configure Environment:**
    Run the `setup` target. This will use Homebrew to install all required packages and generate an SSH key for accessing the VMs.
    ```bash
    make setup
    ```

3.  **Download Cloud Images:**
    Download the required Ubuntu cloud images for both architectures.
    ```bash
    make download-cloud-images
    ```

## Usage

All generated files, including VM disks (`.qcow2`), cloud-init ISOs (`.iso`), logs, and PID files, are stored in the project's root directory and cleaned up automatically.

1.  **Start the `socket_vmnet` Service:**
    This service manages the private virtual network. You may be prompted for your password.
    ```bash
    make start-service
    ```

2.  **Start the Provisioning VM:**
    Open a terminal and run:
    ```bash
    make run-provisioner-vm
    ```

3.  **Start the Target VM:**
    In a second terminal, run:
    ```bash
    make run-target-vm
    ```

4.  **Access the VMs:**
    - **Provisioning VM:** SSH into the VM using the forwarded port:
      ```bash
      make shell-provisioner-vm
      ```
    - **Target VM:** To access the target VM, you must first SSH into the provisioning VM and then connect to the target's private IP:
      ```bash
      # From your Mac
      make shell-provisioner-vm

      # From inside the provisioning VM
      ssh ubuntu@192.168.100.2
      ```
    The password for the `ubuntu` user is `pass` on both VMs.

5.  **Monitor Logs:**
    You can tail the console logs for each VM:
    ```bash
    make tail-provisioner-vm-logs
    make tail-target-vm-logs
    ```

6.  **Stopping the VMs:**
    To stop the VMs gracefully:
    ```bash
    make stop-provisioner-vm
    make stop-target-vm
    ```

## Makefile Targets

| Target                           | Description                                                                              |
| -------------------------------- | ---------------------------------------------------------------------------------------- |
| `setup`                          | Installs dependencies and generates an SSH key.                                          |
| `download-cloud-images`          | Downloads the Ubuntu cloud images.                                                       |
| `start-service`                  | Starts the `socket_vmnet` background service.                                            |
| `stop-service`                   | Stops the `socket_vmnet` service.                                                        |
| `run-provisioner-vm`             | Creates the disk and ISO, then runs the provisioning VM.                                 |
| `stop-provisioner-vm`            | Stops the provisioning VM gracefully.                                                    |
| `shell-provisioner-vm`           | Opens an SSH session to the provisioning VM.                                             |
| `tail-provisioner-vm-logs`       | Tails the console logs for the provisioning VM.                                          |
| `run-target-vm`                  | Creates the disk and ISO, then runs the target VM.                                       |
| `stop-target-vm`                 | Stops the target VM gracefully.                                                          |
| `clean-provisioner-vm`           | Stops the VM and deletes its generated files.                                            |
| `clean-target-vm`                | Stops the VM and deletes its generated files.                                            |
| `clean-all`                      | Stops both VMs and the network service, and deletes all generated files.                 |

## Project Structure

```
.
├── cloud_images/               # Stores downloaded Ubuntu cloud images (ignored by git)
├── cloudconfig-provisioner-vm/ # Cloud-init configuration for the provisioning VM
│   ├── meta-data
│   ├── network-config
│   └── user-data
├── cloudconfig-target-vm/      # Cloud-init configuration for the target VM
│   ├── meta-data
│   ├── network-config
│   └── user-data
├── logs/                       # Stores VM console logs (ignored by git)
├── .ssh/                       # Stores the generated SSH key (ignored by git)
├── .gitignore
├── Makefile
└── README.md
```
