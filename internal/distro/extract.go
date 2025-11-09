package distro

import (
	"context"
	"fmt"
	"pvmlab/internal/config"
)

// Extractor defines the interface for distribution-specific asset extraction.
type Extractor interface {
	ExtractKernelAndModules(ctx context.Context, cfg *config.Config, distroInfo *config.ArchInfo, isoPath, distroPath string) error
	CreateRootfs(ctx context.Context, distroInfo *config.ArchInfo, distroPath string) error
}

// NewExtractor is a factory function that returns the correct extractor for a given distro.
func NewExtractor(distroName string) (Extractor, error) {
	switch distroName {
	case "ubuntu":
		return &UbuntuExtractor{}, nil
	case "fedora":
		return &FedoraExtractor{}, nil
	default:
		return nil, fmt.Errorf("no extractor available for distribution: %s", distroName)
	}
}
