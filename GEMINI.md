# Project Context: bring up a Provisioning Lab using QEMU VMs

The purpose of this project is to bring up a simple provisioning lab using QEMU VMs. The lab will consist of two VMs: one acting as a provisioning server and the other as a target device to be provisioned. The provisioning server will provide PXE booting and installation services to the target device. The target device will be configured to boot from the network and install a minimal Linux OS using the services provided by the provisioning server. Use x86_64 for the target VM architecturem, but use aarch64 for the provisioning VM.

## Architectural guidelines

The architecture of the provisioning lab is based on the following guidelines:
- 2 QEMU VMs
  - 1 VM for the provisioning services (aarch64)
  - 1 VM for the target device being pxebooted and provisioned (x86_64)
- The provisioning VM will run a minimal Linux distribution with the necessary services to provide PXE booting, DHCP, TFTP, and HTTP services. Use a minimal Ubuntu cloud image.
- The target VM will run a qemu x86_64 set up to pxeboot using EFI.
- Networking between the two VMs will follow similar setup you will find in this [website](https://amf3.github.io/articles/virtualization/macos_qemu_networks/)
- Configure the provisioning VM to have a static IP address on the vmnet.host network.
- Run the dhcpd service on the vmnet.host network interface to provide IP addresses to the target VM.
- Configure the TFTP server to serve the necessary boot files for PXE booting (use ipxe.efi and a minimal Linux kernel and initrd).
- Configure the HTTP server to serve the installation files for the minimal Linux OS. It needs ot provide a ubuntu kernel/initrd to ipxe and the ubuntu cloud image to be installed. It also needs to serve as renderer for the cloud-init user-data file and ipxe script.
- Use iPXE to chainload the Linux kernel and initrd from the HTTP server.
- Use cloud-init to automate the installation and configuration of the minimal Linux OS on the target VM
- For the provisioning VM use an ubuntu cloud image and customize it with a cloud-init user-data file to install and configure the necessary services.

## Implementation steps
- Use a Makefile to automate the setup and teardown of the VMs and services.