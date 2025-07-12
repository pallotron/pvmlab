# QEMU Provisioning Lab for macOS

This project automates the setup of a simple virtual provisioning lab on macOS using QEMU, cloud-init, and `make`. The lab consists of two Ubuntu Linux VMs: a provisioning server and a target device, connected via a private host-only network.

The provisioning server is configured to act as a network gateway, providing internet access to the target VM, simulating a real-world provisioning environment.

## Features

- **Automated Setup:** Uses a `Makefile` to automate the entire setup and teardown process.
- **Two-VM Architecture:**
    - **Provisioning VM:** An `aarch64` Ubuntu server that provides network services.
    - **Target VM:** An `x86_64` Ubuntu server that is configured by the lab.
- **Isolated Provisioning Network:** Utilizes QEMU's `vmnet-host` for a private network between the VMs.
- **Internet Access:** The provisioning VM uses QEMU's `user` mode networking for internet access and is configured to act as a gateway for the target VM.
- **Declarative VM Configuration:** Uses `cloud-init` to declaratively configure both VMs on first boot (hostname, static IPs, user accounts, gateway services, etc.).

## Architecture

The lab is composed of two QEMU virtual machines:

1.  **Provisioning VM (`provisioner-vm`):**
    - **OS:** Ubuntu Server 24.04 (aarch64)
    - **Network Interfaces:**
        - `enp0s1`: Connects to the internet via QEMU's `user` network (NAT). This allows the VM to download packages and updates.
        - `enp0s2`: Connects to a private `vmnet-host` network with a static IP of `192.168.100.1`. This interface is used to provision the target VM.
    - **Services:** Configured via `cloud-init` to perform IP forwarding and NAT (masquerading) to act as a gateway for the target VM.

2.  **Target VM (`target-vm`):**
    - **OS:** Ubuntu Server 24.04 (x86_64)
    - **Network Interfaces:**
        - `eth0`: Connects to the same private `vmnet-host` network with a static IP of `192.168.100.2`.
    - **Configuration:** Configured via `cloud-init` to use the provisioning VM (`192.168.100.1`) as its default gateway.

## Prerequisites

- macOS
- [Homebrew](https://brew.sh/)
- QEMU (`qemu`)
- CDRTools (`cdrtools` for `mkisofs`)

## Setup

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd qemu-provisioning-lab
    ```

2.  **Install Dependencies & Configure Sudo:**
    Run the `setup` target. This will use Homebrew to install QEMU and `cdrtools`. It will also prompt for your password to create a file in `/etc/sudoers.d/` that allows your user to run the necessary QEMU binaries with `sudo` without a password.
    ```bash
    make setup
    ```

3.  **Download Cloud Images:**
    Download the required Ubuntu cloud images.
    ```bash
    make download-cloud-images
    ```

## Usage

All generated VM disks (`.qcow2`) and cloud-init ISOs (`.iso`) are stored in the `runtime_images/` directory.

1.  **Start the Provisioning VM:**
    Open a terminal window and run:
    ```bash
    make run-provisioner-vm
    ```
    This will first build the VM disk and cloud-init ISO if they don't exist.

2.  **Start the Target VM:**
    Open a *second* terminal window and run:
    ```bash
    make run-target-vm
    ```

3.  **Access the VMs:**
    - You can SSH into the **provisioning VM** via the port forward set up on the `user` network:
      ```bash
      ssh ubuntu@localhost -p 2222
      ```
    - The password for both `ubuntu` and `root` users is `pass` (as configured in `cloudconfig-provisioner-vm/user-data`).
    - To access the target VM, you must first SSH into the provisioning VM and then SSH from there to the target VM's private IP:
      ```bash
      # From your Mac
      ssh ubuntu@localhost -p 2222

      # From inside the provisioning VM
      ssh ubuntu@192.168.100.2
      ```

4.  **Verify Connectivity:**
    - From the provisioning VM, ping the target VM: `ping 192.168.100.2`
    - From the target VM, ping the provisioning VM: `ping 192.168.100.1`
    - From the target VM, ping the internet to verify the gateway is working: `ping 8.8.8.8`

## Makefile Targets

| Target                           | Description                                                                              |
| -------------------------------- | ---------------------------------------------------------------------------------------- |
| `setup`                          | Installs Homebrew dependencies and sets up the required sudoers file.                    |
| `download-cloud-images`          | Downloads the necessary Ubuntu cloud images into the `cloud_images/` directory.          |
| `run-provisioner-vm`             | Runs the provisioning VM. Depends on `provisioner-vm-disk` and `provisioner-vm-cloudconfig.iso`. |
| `run-target-vm`                  | Runs the target VM. Depends on `target-vm-disk` and `target-vm-cloudconfig.iso`.         |
| `provisioner-vm-disk`            | Creates the `.qcow2` disk for the provisioning VM.                                       |
| `provisioner-vm-cloudconfig.iso` | Creates the cloud-init `.iso` for the provisioning VM.                                   |
| `target-vm-disk`                 | Creates the `.qcow2` disk for the target VM.                                             |
| `target-vm-cloudconfig.iso`      | Creates the cloud-init `.iso` for the target VM.                                         |
| `clean-provisioner-vm`           | Deletes the generated disk and ISO for the provisioning VM.                              |
| `clean-target-vm`                | Deletes the generated disk and ISO for the target VM.                                    |
| `clean-all`                      | Deletes all generated runtime files.                                                     |

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
│   └── network-config
├── runtime_images/             # Stores generated .qcow2 disks and .iso files (ignored by git)
├── .gitignore                  # Git ignore file
├── Makefile                    # Main automation file
└── README.md                   # This file
```