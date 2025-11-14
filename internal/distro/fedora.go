package distro

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"pvmlab/internal/config"
	"pvmlab/internal/downloader"

	"github.com/fatih/color"
)

// FedoraExtractor implements the Extractor interface for Fedora.
type FedoraExtractor struct{}

func (e *FedoraExtractor) ExtractKernelAndInitrd(ctx context.Context, cfg *config.Config, distroInfo *config.ArchInfo, distroPath string) error {
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

	// Extract the boot directory to find the kernel and initrd
	cmd := exec.CommandContext(ctx, "tar", "-xzf", rootfsPath, "-C", extractDir, "./boot")
	if output, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		color.Red("! Failed to extract from rootfs.tar.gz. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract boot directory from rootfs: %w", err)
	}

	// Find kernel and initrd files
	bootDir := filepath.Join(extractDir, "boot")
	files, err := os.ReadDir(bootDir)
	if err != nil {
		return fmt.Errorf("failed to read boot directory: %w", err)
	}

	var kernelPath, initrdPath string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "vmlinuz-") {
			kernelPath = filepath.Join(bootDir, file.Name())
		}
		if strings.HasPrefix(file.Name(), "initramfs-") && strings.HasSuffix(file.Name(), ".img") {
			initrdPath = filepath.Join(bootDir, file.Name())
		}
	}

	if kernelPath == "" || initrdPath == "" {
		return fmt.Errorf("could not find kernel or initrd in boot directory")
	}

	// Move kernel
	finalVmlinuz := filepath.Join(distroPath, filepath.Base(kernelPath))
	if err := os.Rename(kernelPath, finalVmlinuz); err != nil {
		return fmt.Errorf("failed to move vmlinuz: %w", err)
	}
	if err := os.Chmod(finalVmlinuz, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on vmlinuz: %w", err)
	}

	// Move initrd
	finalInitrd := filepath.Join(distroPath, filepath.Base(initrdPath))
	if err := os.Rename(initrdPath, finalInitrd); err != nil {
		return fmt.Errorf("failed to move initrd.img: %w", err)
	}
	if err := os.Chmod(finalInitrd, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on initrd.img: %w", err)
	}

	// --- Create modules.cpio.gz from rootfs ---
	color.Cyan("i Creating modules.cpio.gz from rootfs...")

	modulesExtractDir, err := os.MkdirTemp(distroPath, "extract-modules-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for modules: %w", err)
	}
	defer os.RemoveAll(modulesExtractDir)

	// Extract /usr/lib/modules from the rootfs tarball
	// Tar may exit with a non-zero status if it encounters issues like changing file sizes,
	// so we check for the directory's existence as the primary success metric.
	extractCmd := exec.CommandContext(ctx, "tar", "-xzf", rootfsPath, "-C", modulesExtractDir, "./usr/lib/modules")
	if output, err := extractCmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		// If the directory doesn't exist after the command, it's a hard failure.
		if _, statErr := os.Stat(filepath.Join(modulesExtractDir, "usr/lib/modules")); os.IsNotExist(statErr) {
			color.Red("! Failed to extract modules from rootfs.tar.gz. Output:\n%s", string(output))
			return fmt.Errorf("failed to extract ./usr/lib/modules from rootfs: %w", err)
		}
		color.Yellow("! 'tar' exited with an error but modules directory was found. Proceeding. Output:\n%s", string(output))
	}

	// The modules are in modulesExtractDir/usr/lib/modules.
	// We need to run cpio from the 'usr' directory to maintain the correct path structure (lib/modules/...).
	cpioBaseDir := filepath.Join(modulesExtractDir, "usr")
	if _, err := os.Stat(cpioBaseDir); os.IsNotExist(err) {
		return fmt.Errorf("expected 'usr' directory not found after extraction")
	}

	// Create the cpio archive from the extracted modules
	modulesCpioPath := filepath.Join(distroPath, "modules.cpio")
	cpioCmdStr := fmt.Sprintf("find lib/modules -print0 | cpio --null -o -H newc > %s", modulesCpioPath)
	cpioCmd := exec.CommandContext(ctx, "sh", "-c", cpioCmdStr)
	cpioCmd.Dir = cpioBaseDir
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

func (e *FedoraExtractor) CreateRootfs(ctx context.Context, distroInfo *config.ArchInfo, distroName, distroPath string) error {
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

	// Step 3: Run the Docker container to create the rootfs tarball
	color.Cyan("i Creating rootfs tarball via Docker (press Ctrl+C to cancel)...")

	containerImagePath := filepath.Join("/images", qcow2Name)
	containerScriptPath := filepath.Join("/images", filepath.Base(tmpfile.Name()))
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"--privileged",
		"-v", fmt.Sprintf("%s:/images", distroPath),
		"debian:12",
		"sh", "-c", fmt.Sprintf("'%s' '%s' '%s'", containerScriptPath, containerImagePath, distroName),
	)

	// Stream the output directly to the console
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Check if the command was cancelled by the user
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil // Return nil to avoid showing a scary error message
		}
		return fmt.Errorf("failed to create rootfs tarball via Docker: %w", err)
	}

	color.Green("✔ Rootfs tarball created successfully.")
	return nil
}