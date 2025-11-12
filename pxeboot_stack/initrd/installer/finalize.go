package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// finalize completes the installation process by installing the bootloader.
func finalize(rebootOnSuccess bool, arch string, distro string, diskPath string) error {
	fmt.Println("  -> Finalizing installation...")

	// Determine GRUB target based on architecture
	var grubTarget string
	switch arch {
	case "x86_64":
		grubTarget = "x86_64-efi"
	case "aarch64":
		grubTarget = "aarch64-efi"
	default:
		return fmt.Errorf("unsupported architecture for GRUB installation: %s", arch)
	}

	// Determine distro-specific bootloader settings
	var bootloaderID, grubConfigCmd string
	switch {
	case strings.HasPrefix(distro, "ubuntu"):
		bootloaderID = "ubuntu"
		grubConfigCmd = "update-grub"
	case strings.HasPrefix(distro, "fedora"):
		bootloaderID = "fedora"
		grubConfigCmd = "grub2-mkconfig -o /boot/grub2/grub.cfg"
	default:
		return fmt.Errorf("unsupported distro for grub config generation: %s", distro)
	}

	// Generate a proper fstab, since the cloud images are broken in this regard
	fmt.Println("  -> Generating a sane /etc/fstab...")
	fstabContent := `
# /etc/fstab: static file system information.
#
# Use 'blkid' to print the universally unique identifier for a
# device; this may be used with UUID= as a more robust way to name devices
# that works even if disks are added and removed. See fstab(5).
#
# <file system> <mount point>   <type>  <options>       <dump>  <pass>
LABEL=cloudimg-rootfs    /               ext4    errors=remount-ro 0       1
LABEL=UEFI      /boot/efi       vfat    umask=0077        0       1
`
	if err := os.WriteFile("/mnt/target/etc/fstab", []byte(strings.TrimSpace(fstabContent)), 0644); err != nil {
		return fmt.Errorf("failed to write fstab: %w", err)
	}

	// Mount pseudo-filesystems needed for chroot
	mounts := [][]string{
		{"/proc", "/mnt/target/proc", "proc", "bind"},
		{"/sys", "/mnt/target/sys", "sysfs", "bind"},
		{"/dev", "/mnt/target/dev", "devtmpfs", "bind"},
		{"/dev/pts", "/mnt/target/dev/pts", "devpts", "bind"},
		{"/sys/firmware/efi/efivars", "/mnt/target/sys/firmware/efi/efivars", "efivarfs", "bind"},
	}

	fmt.Println("  -> Mounting pseudo-filesystems for chroot...")
	// Ensure the dev/pts mountpoint exists, as it may not be in the minimal rootfs
	if err := os.MkdirAll("/mnt/target/dev/pts", 0755); err != nil {
		return fmt.Errorf("failed to create /mnt/target/dev/pts: %w", err)
	}

	for _, m := range mounts {
		if err := runCommand("mount", "--bind", m[0], m[1]); err != nil {
			return fmt.Errorf("failed to mount %s: %w", m[0], err)
		}
	}

	// Install the bootloader inside the chroot
	fmt.Println("  -> Installing GRUB bootloader...")
	if err := runCommand("chroot", "/mnt/target", "grub-install", fmt.Sprintf("--target=%s", grubTarget), fmt.Sprintf("--bootloader-id=%s", bootloaderID), "--efi-directory=/boot/efi", "--recheck"); err != nil {
		return fmt.Errorf("grub-install failed: %w", err)
	}

	fmt.Println("  -> Generating initramfs...")
	if err := runCommand("chroot", "/mnt/target", "update-initramfs", "-c", "-k", "all"); err != nil {
		return fmt.Errorf("update-initramfs failed: %w", err)
	}

	fmt.Println("  -> Generating GRUB config...")
	if err := runCommand("chroot", "/mnt/target", "sh", "-c", grubConfigCmd); err != nil {
		return fmt.Errorf("grub config generation failed: %w", err)
	}

	// Unmount filesystems only if we are rebooting
	if rebootOnSuccess {
		fmt.Println("  -> Unmounting filesystems...")
		// Unmount pseudo-filesystems in reverse order
		for i := len(mounts) - 1; i >= 0; i-- {
			m := mounts[i]
			if err := runCommand("umount", m[1]); err != nil {
				fmt.Printf("  -> Warning: failed to unmount %s: %v\n", m[1], err)
			}
		}

		if err := runCommand("umount", "/mnt/target/boot/efi"); err != nil {
			fmt.Printf("  -> Warning: failed to unmount EFI: %v\n", err)
		}

		if err := runCommand("umount", "/mnt/target"); err != nil {
			fmt.Printf("  -> Warning: failed to unmount root: %v\n", err)
		}
	}

	fmt.Println("  -> Finalization complete.")

	if rebootOnSuccess {
		fmt.Println("==> Go OS Installer finished successfully!")
		fmt.Println("==> Rebooting...")
		if err := runCommand("reboot", "-f"); err != nil {
			// As a fallback, use the sysrq trigger
			fmt.Println("  -> reboot command failed, trying sysrq trigger...")
			_ = os.WriteFile("/proc/sysrq-trigger", []byte("b"), 0644)
		}
	} else {
		fmt.Println("==> Go OS Installer finished successfully! (Reboot suppressed)")
		fmt.Println("==> Dropping to debug shell...")
		dropToShell()
	}

	return nil
}

// findKernelVersion finds the kernel version string from the /lib/modules directory.
func findKernelVersion(modulesDir string) (string, error) {
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", modulesDir, err)
	}

	// Find the first directory that looks like a kernel version
	for _, entry := range entries {
		if entry.IsDir() {
			// A simple heuristic: a kernel version directory will contain modules.dep
			modulesDepPath := filepath.Join(modulesDir, entry.Name(), "modules.dep")
			if _, err := os.Stat(modulesDepPath); err == nil {
				return entry.Name(), nil
			}
		}
	}

	return "", fmt.Errorf("no kernel version directory found in %s", modulesDir)
}
