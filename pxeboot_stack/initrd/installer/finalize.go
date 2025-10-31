package main

import (
	"fmt"
)

// finalize completes the installation process
func finalize() error {
	fmt.Println("  -> Installing bootloader...")

	// TODO: Install GRUB or systemd-boot
	fmt.Println("  -> TODO: Install bootloader")

	// Unmount filesystems
	fmt.Println("  -> Unmounting filesystems...")

	if err := runCommand("umount", "/mnt/target/boot/efi"); err != nil {
		fmt.Printf("  -> Warning: failed to unmount EFI: %v\n", err)
	}

	if err := runCommand("umount", "/mnt/target"); err != nil {
		fmt.Printf("  -> Warning: failed to unmount root: %v\n", err)
	}

	fmt.Println("  -> Finalization complete")

	return nil
}
