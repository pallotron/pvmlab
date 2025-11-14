package main

import (
	"fmt"
	"installer/log"
	"os"
	"path/filepath"
)

// installOS downloads and installs the base operating system
func installOS(config *InstallerConfig) error {
	log.Info("Installing base system...")
	log.Info("Downloading rootfs from %s", config.RootfsURL)

	// We will stream the download directly into the tar command to avoid
	// saving the tarball to disk in the initrd.
	// sh -c 'wget -O - <url> | tar -xz -C /mnt/target'
	cmdStr := fmt.Sprintf("wget -O - %s | tar -xz -C /mnt/target", config.RootfsURL)
	if err := runCommand("sh", "-c", cmdStr); err != nil {
		return fmt.Errorf("failed to download and extract rootfs: %w", err)
	}

	log.Info("Rootfs extracted successfully.")

	// Install the kernel
	log.Info("Installing kernel...")

	// Construct the Kernel URL using the filename from the config
	kernelURL := config.KernelURL
	log.Info("Kernel URL: %s", kernelURL)

	// The destination path for the kernel
	kernelDestPath := filepath.Join("/mnt/target/boot", filepath.Base(config.KernelURL))
	log.Info("Kernel Destination: %s", kernelDestPath)

	// Download the kernel using wget
	if err := runCommand("wget", "-O", kernelDestPath, kernelURL); err != nil {
		return fmt.Errorf("failed to download kernel: %w", err)
	}

	// Set permissions to be world-readable for GRUB
	if err := os.Chmod(kernelDestPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on kernel: %w", err)
	}

	log.Info("Kernel installed successfully.")

	return nil
}
