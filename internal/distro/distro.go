package distro

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"pvmlab/internal/config"
	"pvmlab/internal/downloader"

	"github.com/fatih/color"
)

func Pull(cfg *config.Config, distroName, arch string) error {
	if _, err := exec.LookPath("7z"); err != nil {
		return fmt.Errorf("7z is not installed. Please install it to extract PXE boot assets")
	}

	distroPath := filepath.Join(cfg.GetAppDir(), "images", distroName, arch)
	if err := os.MkdirAll(distroPath, 0755); err != nil {
		return fmt.Errorf("failed to create distro directory: %w", err)
	}

	distro, ok := config.Distros[distroName]
	if !ok {
		return fmt.Errorf("distro configuration not found for: %s", distroName)
	}

	distroInfo, ok := distro.Arch[arch]
	if !ok {
		return fmt.Errorf("architecture '%s' not found for distro '%s'", arch, distroName)
	}

	isoPath := filepath.Join(distroPath, distroInfo.ISOName)
	if err := downloader.DownloadImageIfNotExists(isoPath, distroInfo.ISOURL); err != nil {
		return err
	}

	extractor, err := NewExtractor(distro.DistroName)
	if err != nil {
		return err
	}

	if err := extractor.ExtractKernelAndModules(cfg, &distroInfo, isoPath, distroPath); err != nil {
		return err
	}

	color.Green("✔ PXE boot assets prepared successfully (vmlinuz and modules.cpio.gz extracted).\n")
	return nil
}
