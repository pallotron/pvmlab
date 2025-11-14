package main

import (
	"fmt"
	"installer/log"
	"os"
	"strings"
	"time"
)

// prepareDisk partitions, formats, and mounts the target disk, returning the disk path.
func prepareDisk() (string, error) {
	log.Info("Detecting disks...")

	// Find the first available disk (usually /dev/sda or /dev/vda)
	var targetDisk string
	for _, disk := range []string{"/dev/vda", "/dev/sda", "/dev/nvme0n1"} {
		if _, err := os.Stat(disk); err == nil {
			targetDisk = disk
			break
		}
	}

	if targetDisk == "" {
		return "", fmt.Errorf("no suitable disk found")
	}

	log.Info("Found disk: %s", targetDisk)

	// Print disk information for debugging
	log.Info("Dumping disk information...")
	if err := runCommand("parted", "-s", targetDisk, "print"); err != nil {
		log.Warn("Failed to print disk info: %v", err)
	}

	// Zap any existing partition table to ensure a clean slate.
	log.Info("Wiping existing partition table...")
	if err := runCommand("sgdisk", "--zap-all", targetDisk); err != nil {
		return "", fmt.Errorf("failed to wipe partition table: %w", err)
	}

	// Partition the disk
	log.Info("Partitioning disk...")

	// Use sgdisk for GPT partitioning
	// Create EFI partition (+512MB)
	if err := runCommand("sgdisk", "-n", "1:1M:+512M", "-t", "1:ef00", "-c", "1:EFI", targetDisk); err != nil {
		return "", fmt.Errorf("failed to create EFI partition: %w", err)
	}

	// Create root partition (rest of disk)
	if err := runCommand("sgdisk", "-n", "2:0:0", "-t", "2:8300", "-c", "2:root", targetDisk); err != nil {
		return "", fmt.Errorf("failed to create root partition: %w", err)
	}

	log.Info("Partitioning complete")

	// Wait for partitions to appear
	log.Info("Waiting for partitions...")
	time.Sleep(2 * time.Second)

	// Format partitions
	log.Info("Formatting partitions...")

	// Determine partition naming scheme
	var efiPart, rootPart string
	if strings.Contains(targetDisk, "nvme") {
		efiPart = targetDisk + "p1"
		rootPart = targetDisk + "p2"
	} else {
		efiPart = targetDisk + "1"
		rootPart = targetDisk + "2"
	}

	// Format EFI partition
	// TODO: make sure all distros are ok with this label, or if it is only an Ubuntu thing
	if err := runCommand("mkfs.vfat", "-F", "32", "-n", "UEFI", efiPart); err != nil {
		return "", fmt.Errorf("failed to format EFI partition: %w (mkfs.vfat not available - needs to be added to initrd)", err)
	}

	// Format root partition
	// TODO: change label to something more specific for all distros
	if err := runCommand("mkfs.ext4", "-L", "cloudimg-rootfs", rootPart); err != nil {
		return "", fmt.Errorf("failed to format root partition: %w (mkfs.ext4 not available - needs to be added to initrd)", err)
	}

	log.Info("Disk preparation complete")

	// Mount partitions
	log.Info("Mounting partitions...")

	if err := os.MkdirAll("/mnt/target", 0755); err != nil {
		return "", fmt.Errorf("failed to create mount point: %w", err)
	}

	if err := runCommand("mount", "-t", "ext4", rootPart, "/mnt/target"); err != nil {
		return "", fmt.Errorf("failed to mount root partition: %w", err)
	}

	if err := os.MkdirAll("/mnt/target/boot/efi", 0755); err != nil {
		return "", fmt.Errorf("failed to create EFI mount point: %w", err)
	}

	if err := runCommand("mount", "-t", "vfat", efiPart, "/mnt/target/boot/efi"); err != nil {
		return "", fmt.Errorf("failed to mount EFI partition: %w", err)
	}

	log.Info("Partitions mounted")

	return targetDisk, nil
}
