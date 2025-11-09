package distro

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"pvmlab/internal/config"

	"github.com/fatih/color"
)

// UbuntuExtractor implements the Extractor interface for Ubuntu.
type UbuntuExtractor struct{}

func (e *UbuntuExtractor) ExtractKernelAndModules(cfg *config.Config, distroInfo *config.ArchInfo, isoPath, distroPath string) error {
	// Step 1: Extract the kernel
	color.Cyan("i Extracting vmlinuz kernel...")
	extractCmd := exec.Command("7z", "x", "-y", isoPath, "-o"+distroPath, distroInfo.KernelFile)
	if output, err := extractCmd.CombinedOutput(); err != nil {
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
	color.Cyan("i Extracting kernel modules...")
	extractPoolCmd := exec.Command("7z", "x", "-y", isoPath, "-o"+distroPath, "pool")
	if output, err := extractPoolCmd.CombinedOutput(); err != nil {
		color.Red("! 7z pool extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract pool directory: %w", err)
	}

	modulesDebPath, err := findFileWithPrefix(filepath.Join(distroPath, "pool", "main", "l", "linux"), "linux-modules-")
	if err != nil {
		return fmt.Errorf("failed to find linux-modules package: %w", err)
	}

	modulesExtractDir := filepath.Join(distroPath, "modules_extract")
	if err := os.MkdirAll(modulesExtractDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for module extraction: %w", err)
	}
	extractModulesCmd := exec.Command("7z", "x", "-y", modulesDebPath, "-o"+modulesExtractDir)
	if output, err := extractModulesCmd.CombinedOutput(); err != nil {
		color.Red("! 7z module extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract modules from .deb: %w", err)
	}

	dataTarPath := filepath.Join(modulesExtractDir, "data.tar")
	extractDataTarCmd := exec.Command("7z", "x", "-y", dataTarPath, "-o"+modulesExtractDir)
	if output, err := extractDataTarCmd.CombinedOutput(); err != nil {
		color.Red("! 7z data.tar extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract data.tar from .deb: %w", err)
	}

	modulesCpioPath := filepath.Join(distroPath, "modules.cpio")
	cpioCmd := exec.Command("sh", "-c", fmt.Sprintf("find lib -print | cpio -o -H newc > %s", modulesCpioPath))
	cpioCmd.Dir = modulesExtractDir
	if output, err := cpioCmd.CombinedOutput(); err != nil {
		color.Red("! cpio creation failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to create modules cpio: %w", err)
	}

	gzipCmd := exec.Command("gzip", "-f", modulesCpioPath)
	if output, err := gzipCmd.CombinedOutput(); err != nil {
		color.Red("! gzip compression failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to gzip modules cpio: %w", err)
	}

	if err := os.RemoveAll(filepath.Join(distroPath, "pool")); err != nil {
		color.Yellow("Warning: failed to clean up temporary pool directory: %v", err)
	}
	if err := os.RemoveAll(modulesExtractDir); err != nil {
		color.Yellow("Warning: failed to clean up temporary module extraction directory: %v", err)
	}

	return nil
}

func findFileWithPrefix(dir, prefix string) (string, error) {
	var foundPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), prefix) {
			foundPath = path
			return filepath.SkipDir // Stop searching once found
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if foundPath == "" {
		return "", fmt.Errorf("no file with prefix '%s' found in '%s'", prefix, dir)
	}
	return foundPath, nil
}
