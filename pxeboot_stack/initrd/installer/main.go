package main

import (
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

	cloudInitURL := os.Getenv("CLOUD_INIT_URL")
	if cloudInitURL == "" {
		fmt.Println("ERROR: CLOUD_INIT_URL not set")
		dropToShell()
		return
	}

	fmt.Printf("==> Cloud-init URL: %s\n", cloudInitURL)

	fmt.Println("\n==> Phase 1: Network Setup")
	if err := setupNetworking(); err != nil {
		fmt.Printf("ERROR: Failed to setup networking: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 2: Fetch Cloud-Init Configuration")
	cloudInit, err := fetchCloudInitData(cloudInitURL)
	if err != nil {
		fmt.Printf("ERROR: Failed to fetch cloud-init data: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 4: Disk Preparation")
	if err := prepareDisk(); err != nil {
		fmt.Printf("ERROR: Failed to prepare disk: %v\n", err)
		dropToShell()
		return
	}

	fmt.Println("\n==> Phase 5: OS Installation")
	if err := installOS(); err != nil {
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
	if err := finalize(); err != nil {
		fmt.Printf("ERROR: Failed to finalize: %v\n", err)
		dropToShell()
		return
	}

	os.Exit(0)
}
