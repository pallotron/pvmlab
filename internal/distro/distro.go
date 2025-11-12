package distro

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"pvmlab/internal/config"

	"github.com/fatih/color"
)

func Pull(ctx context.Context, cfg *config.Config, distroName, arch string) error {
	if _, err := exec.LookPath("7z"); err != nil {
		return fmt.Errorf("7z is not installed. Please install it to extract PXE boot assets")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not installed. Please install it to create rootfs tarballs")
	}

	imagesDir := filepath.Join(cfg.GetAppDir(), "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}
	if err := os.Chmod(imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to enforce permissions on images directory: %w", err)
	}

	distroPath := filepath.Join(imagesDir, distroName, arch)
	if err := os.MkdirAll(distroPath, 0755); err != nil {
		return fmt.Errorf("failed to create distro directory: %w", err)
	}
	if err := os.Chmod(distroPath, 0755); err != nil {
		return fmt.Errorf("failed to enforce permissions on distro directory: %w", err)
	}

	distro, ok := config.Distros[distroName]
	if !ok {
		return fmt.Errorf("distro configuration not found for: %s", distroName)
	}

	distroInfo, ok := distro.Arch[arch]
	if !ok {
		return fmt.Errorf("architecture '%s' not found for distro '%s'", arch, distroName)
	}

	extractor, err := NewExtractor(distro.DistroName)
	if err != nil {
		return err
	}

	if err := extractor.CreateRootfs(ctx, &distroInfo, distroPath); err != nil {
		return err
	}

	if err := extractor.ExtractKernelAndInitrd(ctx, cfg, &distroInfo, distroPath); err != nil {
		return err
	}

	color.Green("✔ PXE boot assets prepared successfully (vmlinuz and initrd extracted).\n")

	return nil
}
