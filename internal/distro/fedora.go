package distro

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"pvmlab/internal/config"

	"github.com/fatih/color"
)

// FedoraExtractor implements the Extractor interface for Fedora.
type FedoraExtractor struct{}

func (e *FedoraExtractor) ExtractKernelAndModules(cfg *config.Config, distroInfo *config.ArchInfo, isoPath, distroPath string) error {
	// Step 1: Extract the kernel
	color.Cyan("i Extracting vmlinuz kernel...")
	extractKernelCmd := exec.Command("7z", "x", "-y", isoPath, "-o"+distroPath, distroInfo.KernelFile)
	if output, err := extractKernelCmd.CombinedOutput(); err != nil {
		color.Red("! 7z kernel extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract kernel: %w", err)
	}

	extractedVmlinuzPath := filepath.Join(distroPath, distroInfo.KernelFile)
	targetVmlinuzPath := filepath.Join(distroPath, "vmlinuz")

	if _, err := os.Stat(extractedVmlinuzPath); os.IsNotExist(err) {
		return fmt.Errorf("vmlinuz not found at expected path after extraction: %s", extractedVmlinuzPath)
	}

	if err := os.Rename(extractedVmlinuzPath, targetVmlinuzPath); err != nil {
		return fmt.Errorf("failed to move kernel: %w", err)
	}

	// Clean up the directory structure created by the extraction
	if dir := filepath.Dir(distroInfo.KernelFile); dir != "." {
		if err := os.RemoveAll(filepath.Join(distroPath, dir)); err != nil {
			color.Yellow("Warning: failed to clean up temporary kernel directory: %v", err)
		}
	}
	if err := os.Chmod(targetVmlinuzPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on vmlinuz: %w", err)
	}

	// Step 2: Extract and create the modules cpio
	color.Cyan("i Extracting kernel modules from initrd...")

	initrdFile := "images/pxeboot/initrd.img"

	// Create a temporary directory for initrd extraction
	tempInitrdDir, err := os.MkdirTemp(distroPath, "initrd-extract-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for initrd extraction: %w", err)
	}
	defer os.RemoveAll(tempInitrdDir) // Clean up temp directory

	// Extract the full initrd.img to the temporary directory
	extractInitrdCmd := exec.Command("7z", "x", "-y", isoPath, "-o"+tempInitrdDir, initrdFile)
	if output, err := extractInitrdCmd.CombinedOutput(); err != nil {
		color.Red("! 7z initrd extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract initrd to temp dir: %w", err)
	}

	extractedInitrdPath := filepath.Join(tempInitrdDir, initrdFile)
	if _, err := os.Stat(extractedInitrdPath); os.IsNotExist(err) {
		return fmt.Errorf("initrd not found at expected path after extraction to temp dir: %s", extractedInitrdPath)
	}

	// Decompress the initrd.img (which is a gzipped cpio archive)
	decompressedCpioPath := filepath.Join(tempInitrdDir, "initrd.cpio")
	gunzipCmd := exec.Command("gunzip", "-c", extractedInitrdPath)
	cpioFile, err := os.Create(decompressedCpioPath)
	if err != nil {
		return fmt.Errorf("failed to create decompressed cpio file: %w", err)
	}
	defer cpioFile.Close()

	gunzipCmd.Stdout = cpioFile
	// Use Run() instead of CombinedOutput() because we have already set Stdout.
	if err := gunzipCmd.Run(); err != nil {
		return fmt.Errorf("failed to decompress initrd: %w", err)
	}

	// Create a directory to extract only the modules
	modulesExtractDir := filepath.Join(tempInitrdDir, "modules_content")
	if err := os.MkdirAll(modulesExtractDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for module content extraction: %w", err)
	}

	// Extract only usr/lib/modules from the decompressed cpio
	extractModulesCmd := exec.Command("sh", "-c", "cpio -idmv 'usr/lib/modules/*'")
	extractModulesCmd.Dir = modulesExtractDir
	inputFile, err := os.Open(decompressedCpioPath)
	if err != nil {
		return fmt.Errorf("failed to open decompressed cpio for module extraction: %w", err)
	}
	defer inputFile.Close()
	extractModulesCmd.Stdin = inputFile
	if output, err := extractModulesCmd.CombinedOutput(); err != nil {
		color.Red("! cpio module extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract usr/lib/modules from cpio: %w", err)
	}

	// Create a new cpio archive from the extracted modules
	finalCpioPath := filepath.Join(distroPath, "modules.cpio")
	cpioCmd := exec.Command("sh", "-c", fmt.Sprintf("find . -print | cpio -o -H newc > %s", finalCpioPath))
	cpioCmd.Dir = filepath.Join(modulesExtractDir, "usr", "lib", "modules") // Change directory to where modules are
	if output, err := cpioCmd.CombinedOutput(); err != nil {
		color.Red("! cpio creation failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to create modules cpio: %w", err)
	}

	// Gzip the new cpio archive
	targetInitrdPath := filepath.Join(distroPath, "modules.cpio.gz")
	gzipCmd := exec.Command("gzip", "-f", finalCpioPath)
	if output, err := gzipCmd.CombinedOutput(); err != nil {
		color.Red("! gzip compression failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to gzip modules cpio: %w", err)
	}

	if err := os.Chmod(targetInitrdPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on modules.cpio.gz: %w", err)
	}

	return nil
}