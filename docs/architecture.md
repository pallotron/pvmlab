# Architecture

```mermaid
---
config:
    layout: elk
    theme: dark
    elk:
        mergeEdges: true
        nodePlacementStrategy: NETWORK_SIMPLEX
---
flowchart TD
%% Define style classes for different components
classDef hostStyle fill:#cde4ff,stroke:black,stroke-width:2px,color:#000
classDef vmStyle fill:#d5f0d5,stroke:black,stroke-width:2px,color:#000
classDef dockerStyle fill:#fff2cc,stroke:black,stroke-width:2px,color:#000
classDef interfaceStyle fill:#9cced6,stroke:black,stroke-width:2px,color:#000
classDef cliStyle fill:#d69ca4,stroke:black,stroke-width:2px,color:#000
classDef socketVmnetStyle fill:#69a3bf,stroke:black,stroke-width:2px,color:#000

subgraph H [Hypervisor Host, ie MacOs]
    cli[pvmlab CLI]
    en0
    subgraph V [Provisioner VM]
        provisioner_vm_enp0s1(enp0s1)
        provisioner_vm_enp0s2(enp0s2)
        Docker[Docker daemon]
        subgraph "pxeboot_stack" [pxeboot_stack container]
            pxeboot_services
        end
    end
    subgraph T [Target VM]
        target_vm_enp0s1(enp0s1)
    end
    subgraph N [socket_vmnet<br/>Apple vmnet framework]
        virtual_net1_private(net1 vmnet.host - no dhcp)
        virtual_net0_shared(net0 vmnet.shared - macos dhcp)
    end
end
Internet

en0 <----> Internet
Docker -- manages --> pxeboot_stack
provisioner_vm_enp0s1 <--> virtual_net0_shared
provisioner_vm_enp0s2 <---> virtual_net1_private
provisioner_vm_enp0s2 <-- NAT --> provisioner_vm_enp0s1
target_vm_enp0s1 <---> virtual_net1_private
pxeboot_services <-- bind to --> provisioner_vm_enp0s2
virtual_net0_shared <--> en0
cli -- manages via QEMU --> V
cli -- manages via QEMU --> T

%% Apply the defined classes to the nodes
class H hostStyle
class pxeboot_stack,Docker,pxeboot_services dockerStyle
class V,T vmStyle
class target_vm_enp0s1,provisioner_vm_enp0s1,provisioner_vm_enp0s2,provisioner_vm_enp0s3,en0,virtual_net0_shared,virtual_net1_private interfaceStyle
class cli cliStyle
class N socketVmnetStyle
class Internet interfaceStyle
linkStyle default stroke:black
```

## Features

- **Go-based CLI:** A modern, easy-to-use command-line interface (`pvmlab`) for managing the entire lab lifecycle.
- **Clean Project Directory:** All generated files are stored outside the project's directory in `~/.pvmlab/`.
- **Two role VM Architecture:**
  - **Provisioner VM:** An `aarch64` Ubuntu server that provides pxeboot and NAT services for the target VMs.
  - **Target VM:** An `aarch64` Ubuntu server that sits in the private network and is provisioned by the provisioner VM. The provisioner VM also provides internet access for the target VMs.
- **Isolated Provisioning Network:** Utilizes `socket_vmnet` to create a private host-only network for provisioning services.
- **Internet Access:** The provisioner VM is connected to a shared network for internet access, and provides NAT for the target VMs on the private network.
- **Declarative VM Configuration:** Uses `cloud-init` to declaratively configure both VMs on first boot.
- **Docker Containerization:** Utilizes `Docker` to run a `supervisord` container to manage the pxeboot stack:
  - DHCP server to hand over IP settings to the target VMs
  - TFTP server to serve the iPXE boot files to the target VMs
  - HTTP server to serve initrd, ramdisk, OS images and cloud-init ISOs to the target VMs

## VMs

**Provisioner VM:**

- **OS:** Ubuntu Server 24.04 (aarch64)
- **Role:** `provisioner`, there could be only one provisioner per lab
- **Network Interfaces:**
  - `enp0s1` (WAN): Connects to a shared network with DHCP for internet access.
  - `enp0s2` (LAN): Connects to the private network with a static IP and NAT for the target VMs.
- **Services:** Configured via `cloud-init` to enable IP forwarding and configure NAT.
- **Docker**: Utilizes `Docker` to run the pxeboot stack

**Target VMs:**

- **OS:** Ubuntu Server 24.04 (aarch64)
- **Role:** `target`
- **Network Interface:**
  - `enp0s1`: Connects to the private network and obtains its IP from the dhcpd server running on the provisioner VM.
