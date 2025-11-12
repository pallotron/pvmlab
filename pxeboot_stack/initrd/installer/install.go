package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// installOS downloads and installs the base operating system
func installOS(config *InstallerConfig) error {
	fmt.Println("  -> Installing base system...")
	fmt.Printf("  -> Downloading rootfs from %s\n", config.RootfsURL)

	// We will stream the download directly into the tar command to avoid
	// saving the tarball to disk in the initrd.
	// sh -c 'wget -O - <url> | tar -xz -C /mnt/target'
	cmdStr := fmt.Sprintf("wget -O - %s | tar -xz -C /mnt/target", config.RootfsURL)
	if err := runCommand("sh", "-c", cmdStr); err != nil {
		return fmt.Errorf("failed to download and extract rootfs: %w", err)
	}

	fmt.Println("  -> Rootfs extracted successfully.")

	// Install the kernel
	fmt.Println("  -> Installing kernel...")

	// Construct the Kernel URL using the filename from the config
	kernelURL := config.KernelURL
	fmt.Printf("  -> Kernel URL: %s\n", kernelURL)

	// The destination path for the kernel
	kernelDestPath := filepath.Join("/mnt/target/boot", filepath.Base(config.KernelURL))
	fmt.Printf("  -> Kernel Destination: %s\n", kernelDestPath)

	// Download the kernel using wget
	if err := runCommand("wget", "-O", kernelDestPath, kernelURL); err != nil {
		return fmt.Errorf("failed to download kernel: %w", err)
	}

	// Set permissions to be world-readable for GRUB
	if err := os.Chmod(kernelDestPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on kernel: %w", err)
	}

	fmt.Println("  -> Kernel installed successfully.")

	return nil
}
