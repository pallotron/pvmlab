package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	// Catch any panics and drop to shell
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n==> PANIC: %v\n", r)
			fmt.Println("==> Dropping to debug shell...")
			dropToShell()
		}
	}()

	fmt.Println("==> Go OS Installer started!")

	fmt.Println("\n==> Phase 1: Network Setup")
	netConfig, err := setupNetworking()
	if err != nil {
		fmt.Printf("ERROR: Failed to setup networking: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 2: Fetch Installer Configuration")
	if netConfig.ConfigURL == "" {
		fmt.Println("ERROR: config_url not found in kernel command line")
		dropToShell()
		return
	}
	fmt.Printf("==> Config URL: %s\n", netConfig.ConfigURL)
	configBytes, err := fetchURL(netConfig.ConfigURL)
	if err != nil {
		fmt.Printf("ERROR: Failed to fetch installer config: %v\n", err)
		dropToShell()
		return
	}
	var installerConfig InstallerConfig
	if err := json.Unmarshal(configBytes, &installerConfig); err != nil {
		fmt.Printf("ERROR: Failed to parse installer config: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 3: Fetch Cloud-Init Configuration")
	cloudInit, err := fetchCloudInitData(installerConfig.CloudInitURL)
	if err != nil {
		fmt.Printf("ERROR: Failed to fetch cloud-init data: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 4: Disk Preparation")
	diskPath, err := prepareDisk()
	if err != nil {
		fmt.Printf("ERROR: Failed to prepare disk: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 5: OS Installation")
	if err := installOS(&installerConfig); err != nil {
		fmt.Printf("ERROR: Failed to install OS: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 6: System Configuration")
	if err := configureSystem(cloudInit); err != nil {
		fmt.Printf("ERROR: Failed to configure system: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 7: Finalization")
	if err := finalize(installerConfig.RebootOnSuccess, installerConfig.Arch, installerConfig.Distro, diskPath); err != nil {
		fmt.Printf("ERROR: Failed to finalize: %v\n", err)
		dropToShell()
		return
	}

	os.Exit(0)
}
