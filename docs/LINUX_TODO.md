# Porting pvmlab to Linux: A TODO List

This document outlines the necessary steps and effort required to port the `pvmlab` tool to run on standard Linux distributions.

## Overview

The core application is largely cross-platform, but the networking layer is currently tied to macOS. The primary goal of this port is to replace the macOS-specific `socket_vmnet` component with a native Linux networking implementation.

## Component-by-Component Breakdown

### 1. Core Application (Low Effort)

The following components are already cross-platform or have direct equivalents on Linux.

-   **`pvmlab` CLI (Go):** The main CLI is written in Go and will compile on Linux with minimal to no changes.
-   **QEMU:** The project uses QEMU, which is the standard for virtualization on Linux. The QEMU commands may require minor adjustments, but the core functionality is fully supported.
-   **Docker:** The provisioner VM relies on Docker, which runs natively on Linux. This workflow will remain unchanged.
-   **Dependencies:** The existing dependencies (`cdrtools`, `socat`) are standard Linux packages. New dependencies like `bridge-utils` (`brctl`) and `tunctl` will need to be added to the setup checks.

### 2. Networking Layer (High Effort)

This is the most significant part of the porting effort. The `socket_vmnet` component, which wraps Apple's proprietary `vmnet.framework`, must be replaced.

-   **Problem:** `socket_vmnet` is macOS-specific and has no direct equivalent on Linux.
-   **Solution:** The standard Linux approach for VM networking is to use a combination of **Linux Bridges** and **TAP devices**.

#### Implementation Plan

1.  **Abstract the Networking Code in Go:**
    -   Refactor the existing code that calls `pvmlab socket_vmnet start/stop`.
    -   Define a generic Go `interface` for network management (e.g., `NetworkManager`). This interface should have methods like `Setup()`, `Teardown()`, `CreatePrivateNetwork()`, and `CreateSharedNetwork()`.

2.  **Create a Linux-Specific Implementation:**
    -   Write a new Go file (`network_linux.go`) that implements the `NetworkManager` interface.
    -   This implementation will execute shell commands (`ip`, `brctl`, `tunctl`) to perform the following:
        -   **Private Network (`virtual_net1_private`):**
            -   Create a new bridge (e.g., `pvmlab-priv`).
            -   For each VM, create a TAP device (e.g., `tap-vm1`).
            -   Attach the TAP device to the `pvmlab-priv` bridge.
            -   Update the QEMU command to use the TAP device (e.g., `-netdev tap,id=net0,ifname=tap-vm1,script=no,downscript=no`).
        -   **Shared Network (`virtual_net0_shared`):**
            -   Create a second bridge (e.g., `pvmlab-pub`).
            -   Attach the host's primary physical network interface (e.g., `eth0`) to this bridge.
            -   Attach the provisioner VM's "WAN" TAP device to this bridge.
            -   Configure `iptables` rules on the host for IP forwarding and NAT to allow the provisioner VM to access the internet.

3.  **Use Go Build Tags:**
    -   Keep the existing `socket_vmnet` logic in a file named `network_darwin.go`.
    -   Place the new Linux bridge/TAP logic in `network_linux.go`.
    -   Use Go build tags (`// +build darwin` and `// +build linux`) at the top of these files. This will allow the Go compiler to automatically select the correct implementation based on the target operating system.

4.  **Update `pvmlab setup`:**
    -   Modify the `setup` command to check for Linux-specific dependencies (`bridge-utils`, `tunctl`, `iptables`).
    -   The command will need to handle permissions, as creating bridges and TAP devices requires `sudo` privileges. This may involve adding the user to a specific group or running parts of the setup with elevated permissions.

## Summary of Work

| Component | macOS Implementation | Linux Equivalent | Effort |
| :--- | :--- | :--- | :--- |
| **CLI** | Go | Go | **Low** |
| **Virtualization** | QEMU | QEMU | **Low** |
| **Containerization**| Docker | Docker | **Low** |
| **Networking** | `socket_vmnet` | Linux Bridge + TAP devices | **High** |

## Conclusion

Porting `pvmlab` to Linux is a very achievable task. The core application logic is portable, and the main effort is concentrated on implementing a new networking backend for Linux. A developer with experience in Go, Linux networking (`ip`, `brctl`, `iptables`), and QEMU could likely complete this work in a few days to a week.
