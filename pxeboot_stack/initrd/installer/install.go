package main

import (
	"fmt"
	"os"
)

// installOS downloads and installs the base operating system
func installOS() error {
	fmt.Println("  -> Installing base system...")

	// For now, this is a placeholder
	// In a real implementation, you would:
	// 1. Download the base OS image (e.g., Ubuntu cloud image)
	//    Stream and extract the http tarball file directly to avoid using extra disk space
	// 2. Extract it to /mnt/target
	// 3. Set up basic directories

	fmt.Println("  -> TODO: Download and extract OS image")
	fmt.Println("  -> Creating basic directory structure...")

	dirs := []string{
		"/mnt/target/etc",
		"/mnt/target/var",
		"/mnt/target/proc",
		"/mnt/target/sys",
		"/mnt/target/dev",
		"/mnt/target/tmp",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	return nil
}
