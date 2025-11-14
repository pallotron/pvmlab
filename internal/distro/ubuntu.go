package distro

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"pvmlab/internal/config"
	"pvmlab/internal/downloader"

	"github.com/fatih/color"
)

//go:embed create-rootfs.sh
var createRootfsScript []byte

// UbuntuExtractor implements the Extractor interface for Ubuntu.
type UbuntuExtractor struct{}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (e *UbuntuExtractor) ExtractKernelAndInitrd(ctx context.Context, cfg *config.Config, distroInfo *config.ArchInfo, distroPath string) error {
	color.Cyan("i Extracting PXE boot assets from rootfs...")

	rootfsPath := filepath.Join(distroPath, "rootfs.tar.gz")
	if _, err := os.Stat(rootfsPath); os.IsNotExist(err) {
		return fmt.Errorf("rootfs.tar.gz not found at %s", rootfsPath)
	}

	// --- Extract Kernel and Initrd ---
	extractDir, err := os.MkdirTemp(distroPath, "extract-")
	if err != nil {
		return fmt.Errorf("failed to create temporary extraction directory: %w", err)
	}
	defer os.RemoveAll(extractDir)

	cmd := exec.CommandContext(ctx, "tar", "-xzf", rootfsPath, "-C", extractDir, distroInfo.KernelPath, distroInfo.InitrdPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		color.Red("! Failed to extract from rootfs.tar.gz. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract kernel and initrd from rootfs: %w", err)
	}

	// Move kernel
	finalVmlinuz := filepath.Join(distroPath, filepath.Base(distroInfo.KernelPath))
	if err := os.Rename(filepath.Join(extractDir, distroInfo.KernelPath), finalVmlinuz); err != nil {
		return fmt.Errorf("failed to move vmlinuz: %w", err)
	}
	if err := os.Chmod(finalVmlinuz, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on vmlinuz: %w", err)
	}

	// Move initrd
	finalInitrd := filepath.Join(distroPath, filepath.Base(distroInfo.InitrdPath))
	if err := os.Rename(filepath.Join(extractDir, distroInfo.InitrdPath), finalInitrd); err != nil {
		return fmt.Errorf("failed to move initrd.img: %w", err)
	}
	if err := os.Chmod(finalInitrd, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on initrd.img: %w", err)
	}

	// --- Create modules.cpio.gz ---
	color.Cyan("i Creating modules.cpio.gz from rootfs...")
	modulesDir, err := os.MkdirTemp(distroPath, "modules-")
	if err != nil {
		return fmt.Errorf("failed to create temporary modules directory: %w", err)
	}
	defer os.RemoveAll(modulesDir)

	// Extract usr/lib/modules from the rootfs
	// Modern Ubuntu uses merged /usr where /lib is a symlink to usr/lib
	// When guestfish creates the tarball, it follows the symlink, so modules are at ./usr/lib/modules
	tarCmd := exec.CommandContext(ctx, "tar", "-xzf", rootfsPath, "-C", modulesDir, "./usr/lib/modules")
	if output, err := tarCmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		color.Red("! Failed to extract modules from rootfs.tar.gz. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract modules from rootfs: %w", err)
	}

	cpioBaseDir := filepath.Join(modulesDir, "usr")

	// Create the cpio archive
	modulesCpioPath := filepath.Join(distroPath, "modules.cpio")
	cpioCmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("find . -print | cpio -o -H newc > %s", modulesCpioPath))
	cpioCmd.Dir = cpioBaseDir // Run from the appropriate base directory
	if output, err := cpioCmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		color.Red("! cpio creation failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to create modules cpio: %w", err)
	}

	// Gzip the archive
	gzipCmd := exec.CommandContext(ctx, "gzip", "-f", modulesCpioPath)
	if output, err := gzipCmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		color.Red("! gzip compression failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to gzip modules cpio: %w", err)
	}

	// Set final permissions
	if err := os.Chmod(modulesCpioPath+".gz", 0644); err != nil {
		return fmt.Errorf("failed to set permissions on modules.cpio.gz: %w", err)
	}

	return nil
}

func (e *UbuntuExtractor) CreateRootfs(ctx context.Context, distroInfo *config.ArchInfo, distroName, distroPath string) error {
	// Step 1: Download the qcow2 image
	qcow2Name := filepath.Base(distroInfo.Qcow2URL)
	qcow2Path := filepath.Join(distroPath, qcow2Name)
	if err := downloader.DownloadImageIfNotExists(ctx, qcow2Path, distroInfo.Qcow2URL); err != nil {
		return err
	}

	// Step 2: Create a temporary script file from the embedded script in distroPath
	tmpfile, err := os.CreateTemp(distroPath, "create-rootfs-*.sh")
	if err != nil {
		return fmt.Errorf("failed to create temporary script file in %s: %w", distroPath, err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(createRootfsScript); err != nil {
		return fmt.Errorf("failed to write to temporary script file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary script file: %w", err)
	}
	if err := os.Chmod(tmpfile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make temporary script executable: %w", err)
	}

	// Step 3: Run the script to create the rootfs tarball
	// On Linux, run natively. On other platforms (macOS), use Docker.
	var cmd *exec.Cmd

	if os.Getenv("GOOS") == "linux" || fileExists("/proc/version") {
		// Running on Linux - execute script directly
		color.Cyan("i Creating rootfs tarball natively on Linux (press Ctrl+C to cancel)...")
		cmd = exec.CommandContext(ctx, "sudo", tmpfile.Name(), qcow2Path, distroName)
	} else {
		color.Cyan("i Creating rootfs tarball via Docker (press Ctrl+C to cancel)...")
		containerImagePath := filepath.Join("/images", qcow2Name)
		containerScriptPath := filepath.Join("/images", filepath.Base(tmpfile.Name()))
		cmd = exec.CommandContext(ctx, "docker", "run", "--rm",
			"--privileged",
			"-v", fmt.Sprintf("%s:/images", distroPath),
			"debian:12",
			containerScriptPath, containerImagePath, distroName,
		)
		fmt.Println(cmd)
	}

	// Stream the output directly to the console
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Check if the command was cancelled by the user
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil // Return nil to avoid showing a scary error message
		}
		return fmt.Errorf("failed to create rootfs tarball: %w", err)
	}

	color.Green("✔ Rootfs tarball created successfully.")
	return nil
}
