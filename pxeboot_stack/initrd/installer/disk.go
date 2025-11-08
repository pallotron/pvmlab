package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// prepareDisk partitions, formats, and mounts the target disk
func prepareDisk() error {
	fmt.Println("  -> Detecting disks...")

	// Find the first available disk (usually /dev/sda or /dev/vda)
	var targetDisk string
	for _, disk := range []string{"/dev/vda", "/dev/sda", "/dev/nvme0n1"} {
		if _, err := os.Stat(disk); err == nil {
			targetDisk = disk
			break
		}
	}

	if targetDisk == "" {
		return fmt.Errorf("no suitable disk found")
	}

	fmt.Printf("  -> Found disk: %s\n", targetDisk)

	// Print disk information for debugging
	fmt.Println("  -> Dumping disk information...")
	if err := runCommand("parted", "-s", targetDisk, "print"); err != nil {
		fmt.Printf("WARNING: Failed to print disk info: %v\n", err)
	}

	// Zap any existing partition table to ensure a clean slate.
	fmt.Println("  -> Wiping existing partition table...")
	if err := runCommand("sgdisk", "--zap-all", targetDisk); err != nil {
		return fmt.Errorf("failed to wipe partition table: %w", err)
	}

	// Partition the disk
	fmt.Println("  -> Partitioning disk...")

	// Use sgdisk for GPT partitioning
	// Create EFI partition (+512MB)
	if err := runCommand("sgdisk", "-n", "1:1M:+512M", "-t", "1:ef00", "-c", "1:EFI", targetDisk); err != nil {
		return fmt.Errorf("failed to create EFI partition: %w", err)
	}

	// Create root partition (rest of disk)
	if err := runCommand("sgdisk", "-n", "2:0:0", "-t", "2:8300", "-c", "2:root", targetDisk); err != nil {
		return fmt.Errorf("failed to create root partition: %w", err)
	}

	fmt.Println("  -> Partitioning complete")

	// Wait for partitions to appear
	fmt.Println("  -> Waiting for partitions...")
	time.Sleep(2 * time.Second)

	// Format partitions
	fmt.Println("  -> Formatting partitions...")

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
	if err := runCommand("mkfs.vfat", "-F", "32", "-n", "EFI", efiPart); err != nil {
		return fmt.Errorf("failed to format EFI partition: %w (mkfs.vfat not available - needs to be added to initrd)", err)
	}

	// Format root partition
	if err := runCommand("mkfs.ext4", "-L", "root", rootPart); err != nil {
		return fmt.Errorf("failed to format root partition: %w (mkfs.ext4 not available - needs to be added to initrd)", err)
	}

	fmt.Println("  -> Disk preparation complete")

	// Mount partitions
	fmt.Println("  -> Mounting partitions...")

	if err := os.MkdirAll("/mnt/target", 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	if err := runCommand("mount", "-t", "ext4", rootPart, "/mnt/target"); err != nil {
		return fmt.Errorf("failed to mount root partition: %w", err)
	}

	if err := os.MkdirAll("/mnt/target/boot/efi", 0755); err != nil {
		return fmt.Errorf("failed to create EFI mount point: %w", err)
	}

	if err := runCommand("mount", "-t", "vfat", efiPart, "/mnt/target/boot/efi"); err != nil {
		return fmt.Errorf("failed to mount EFI partition: %w", err)
	}

	fmt.Println("  -> Partitions mounted")

	return nil
}
