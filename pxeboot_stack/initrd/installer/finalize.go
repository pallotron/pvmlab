package main

import (
	"fmt"
	"installer/log"
	"os"
	"path/filepath"
	"strings"
)

// finalize completes the installation process by installing the bootloader.
func finalize(rebootOnSuccess bool, arch string, distro string, diskPath string) error {
	log.Info("Finalizing installation...")

	// Mount pseudo-filesystems needed for chroot
	mounts := [][]string{
		{"/proc", "/mnt/target/proc", "proc", "bind"},
		{"/sys", "/mnt/target/sys", "sysfs", "bind"},
		{"/dev", "/mnt/target/dev", "devtmpfs", "bind"},
		{"/dev/pts", "/mnt/target/dev/pts", "devpts", "bind"},
		{"/sys/firmware/efi/efivars", "/mnt/target/sys/firmware/efi/efivars", "efivarfs", "bind"},
	}

	log.Info("Mounting pseudo-filesystems for chroot...")
	// Ensure the dev/pts mountpoint exists, as it may not be in the minimal rootfs
	if err := os.MkdirAll("/mnt/target/dev/pts", 0755); err != nil {
		return fmt.Errorf("failed to create /mnt/target/dev/pts: %w", err)
	}

	for _, m := range mounts {
		if err := runCommand("mount", "--bind", m[0], m[1]); err != nil {
			return fmt.Errorf("failed to mount %s: %w", m[0], err)
		}
	}

	// Copy resolv.conf for network access inside chroot.
	// This will be overridden by cloud-init on first boot.
	log.Info("Configuring DNS for chroot...")
	resolvConf, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		log.Warn("Could not read initrd's /etc/resolv.conf: %v. Creating a default one.", err)
		resolvConf = []byte("nameserver 8.8.8.8\n") // Fallback to Google DNS
	}

	// Handle the case where /etc/resolv.conf is a symlink in the target,
	// which is common on Ubuntu/systemd systems (../run/systemd/resolve/stub-resolv.conf)
	resolvConfPath := "/mnt/target/etc/resolv.conf"
	if l, err := os.Lstat(resolvConfPath); err == nil && l.Mode()&os.ModeSymlink != 0 {
		log.Info("Removing existing resolv.conf symlink...")
		if err := os.Remove(resolvConfPath); err != nil {
			return fmt.Errorf("failed to remove resolv.conf symlink: %w", err)
		}
	}

	if err := os.WriteFile(resolvConfPath, resolvConf, 0644); err != nil {
		return fmt.Errorf("failed to write /etc/resolv.conf to chroot: %w", err)
	}

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
	var bootloaderID, grubConfigCmd, grubInstallCmd string
	var requiredPkgs []string
	var pkgManagerCmd []string

	switch {
	case strings.HasPrefix(distro, "ubuntu"):
		bootloaderID = "ubuntu"
		grubConfigCmd = "update-grub"
		grubInstallCmd = "grub-install"
		pkgManagerCmd = []string{"apt-get", "install", "-y"}
		if arch == "x86_64" {
			requiredPkgs = []string{"grub-efi-amd64"}
		} else {
			requiredPkgs = []string{"grub-efi-arm64"}
		}
		log.Info("Updating package lists...")
		if err := runCommand("chroot", "/mnt/target", "apt-get", "update"); err != nil {
			return fmt.Errorf("apt-get update failed: %w", err)
		}

	case strings.HasPrefix(distro, "fedora"):
		bootloaderID = "fedora"
		grubConfigCmd = "grub2-mkconfig -o /boot/grub2/grub.cfg"
		grubInstallCmd = "grub2-install"
		pkgManagerCmd = []string{"dnf", "install", "-y"}
		if arch == "x86_64" {
			requiredPkgs = []string{"grub2-efi-x64", "dracut-config-generic"}
		} else {
			requiredPkgs = []string{"grub2-efi-aa64", "dracut-config-generic"}
		}

	default:
		return fmt.Errorf("unsupported distro for grub config generation: %s", distro)
	}

	// Generate a proper fstab, since the cloud images are broken in this regard
	log.Info("Generating a sane /etc/fstab...")
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

	// Install required packages inside the chroot
	log.Info("Installing required packages (%s) inside chroot...", strings.Join(requiredPkgs, " "))
	installArgs := append(pkgManagerCmd, requiredPkgs...)
	chrootInstallArgs := append([]string{"chroot", "/mnt/target"}, installArgs...)
	if err := runCommand(chrootInstallArgs[0], chrootInstallArgs[1:]...); err != nil {
		return fmt.Errorf("failed to install required packages: %w", err)
	}

	// Create /var/tmp (for dracut mainly)
	log.Info("Creating /var/tmp in chroot for dracut...")
	if err := os.MkdirAll("/mnt/target/var/tmp", 0755); err != nil {
		return fmt.Errorf("failed to create /mnt/target/var/tmp: %w", err)
	}

	// Install the bootloader inside the chroot
	log.Info("Installing GRUB bootloader...")
	if err := runCommand(
		"chroot", "/mnt/target",
		grubInstallCmd,
		fmt.Sprintf("--target=%s", grubTarget),
		fmt.Sprintf("--bootloader-id=%s", bootloaderID),
		"--efi-directory=/boot/efi", "--recheck", "--force",
	); err != nil {
		return fmt.Errorf("grub-install failed: %w", err)
	}

	log.Info("Generating initramfs...")
	switch {
	case strings.HasPrefix(distro, "ubuntu"):
		if err := runCommand("chroot", "/mnt/target", "update-initramfs", "-c", "-k", "all"); err != nil {
			return fmt.Errorf("update-initramfs failed: %w", err)
		}
	case strings.HasPrefix(distro, "fedora"):
		kernelVersion, err := findKernelVersion("/mnt/target/lib/modules")
		if err != nil {
			return fmt.Errorf("could not determine kernel version for initramfs generation: %w", err)
		}
		log.Info("Found kernel version %s for initramfs generation", kernelVersion)
		// dracut will automatically create the initramfs in /boot for the specified kernel version
		if err := runCommand("chroot", "/mnt/target", "dracut", "--force", kernelVersion); err != nil {
			return fmt.Errorf("dracut failed: %w", err)
		}

		// Read /proc/cmdline from the initrd to set sane GRUB defaults
		cmdline, err := getKernelCmdline()
		if err != nil {
			return fmt.Errorf("failed to read initrd's /proc/cmdline: %w", err)
		}

		// Filter out initrd-specific parameters and kernel image name
		var filteredArgs []string
		for _, arg := range strings.Fields(cmdline) {
			if !strings.HasPrefix(arg, "initrd.mode=") && !strings.HasPrefix(arg, "config_url=") && !strings.HasPrefix(arg, "installer_mac=") && !strings.HasPrefix(arg, "vmlinuz-") {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		// Add SELinux disable parameter for the first boot
		// TODO: make SELinux work.
		filteredArgs = append(filteredArgs, "selinux=0")
		newCmdline := strings.Join(filteredArgs, " ")

		// Construct the new GRUB_CMDLINE_LINUX_DEFAULT line
		newGrubCmdline := fmt.Sprintf("GRUB_CMDLINE_LINUX_DEFAULT=\"%s\"", newCmdline)

		// Read, modify, and write /etc/default/grub in Go to avoid using sed
		log.Info("Updating GRUB_CMDLINE_LINUX_DEFAULT in /etc/default/grub...")
		grubDefaultPath := "/mnt/target/etc/default/grub"
		grubDefaultBytes, err := os.ReadFile(grubDefaultPath)
		if err != nil {
			log.Warn("failed to read /etc/default/grub: %v", err)
		} else {
			lines := strings.Split(string(grubDefaultBytes), "\n")
			var newLines []string
			found := false
			for _, line := range lines {
				if strings.HasPrefix(line, "GRUB_CMDLINE_LINUX_DEFAULT=") {
					newLines = append(newLines, newGrubCmdline)
					found = true
				} else {
					newLines = append(newLines, line)
				}
			}
			if !found {
				newLines = append(newLines, newGrubCmdline)
			}

			output := strings.Join(newLines, "\n")
			if err := os.WriteFile(grubDefaultPath, []byte(output), 0644); err != nil {
				log.Warn("failed to write updated /etc/default/grub: %v", err)
			}
		}

		// Create /var/lib/chrony directory for chronyd.service
		log.Info("Creating /var/lib/chrony directory...")
		if err := runCommand("chroot", "/mnt/target", "mkdir", "-p", "/var/lib/chrony"); err != nil {
			log.Warn("failed to create /var/lib/chrony: %v", err)
		}

		// Create symlinks in / for GRUB to find the kernel and initrd.
		// This is a workaround for grub2-mkconfig in some cloud images that
		// generates incorrect paths.
		log.Info("Creating kernel and initramfs symlinks in / for GRUB...")
		kernelFile := fmt.Sprintf("vmlinuz-%s", kernelVersion)
		initramfsFile := fmt.Sprintf("initramfs-%s.img", kernelVersion)
		if err := runCommand("chroot", "/mnt/target", "ln", "-sf", "boot/"+kernelFile, kernelFile); err != nil {
			log.Warn("failed to create symlink for kernel: %v", err)
		}
		if err := runCommand("chroot", "/mnt/target", "ln", "-sf", "boot/"+initramfsFile, initramfsFile); err != nil {
			log.Warn("failed to create symlink for initramfs: %v", err)
		}
	default:
		return fmt.Errorf("unsupported distro for initramfs generation: %s", distro)
	}

	log.Info("Generating GRUB config...")
	if err := runCommand("chroot", "/mnt/target", "sh", "-c", grubConfigCmd); err != nil {
		return fmt.Errorf("grub config generation failed: %w", err)
	}

	// Unmount filesystems only if we are rebooting
	if rebootOnSuccess {
		log.Info("Unmounting filesystems...")
		// Unmount pseudo-filesystems in reverse order
		for i := len(mounts) - 1; i >= 0; i-- {
			m := mounts[i]
			if err := runCommand("umount", m[1]); err != nil {
				log.Warn("failed to unmount %s: %v", m[1], err)
			}
		}

		if err := runCommand("umount", "/mnt/target/boot/efi"); err != nil {
			log.Warn("failed to unmount EFI: %v", err)
		}

		if err := runCommand("umount", "/mnt/target"); err != nil {
			log.Warn("failed to unmount root: %v", err)
		}
	}

	log.Info("Finalization complete.")

	if rebootOnSuccess {
		log.Title("Go OS Installer finished successfully!")
		log.Title("Rebooting...")
		if err := runCommand("reboot", "-f"); err != nil {
			// As a fallback, use the sysrq trigger
			log.Info("reboot command failed, trying sysrq trigger...")
			_ = os.WriteFile("/proc/sysrq-trigger", []byte("b"), 0644)
		}
	} else {
		log.Title("Go OS Installer finished successfully! (Reboot suppressed)")
		log.Title("Dropping to debug shell...")
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
