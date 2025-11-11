package distro

import (
	"context"
	"fmt"

	"pvmlab/internal/config"
)

// FedoraExtractor implements the Extractor interface for Fedora.
type FedoraExtractor struct{}

func (e *FedoraExtractor) ExtractKernelAndInitrd(ctx context.Context, cfg *config.Config, distroInfo *config.ArchInfo, distroPath string) error {
	// Not implemented for Fedora yet, as we don't have a clear way to extract kernel/initrd from Fedora cloud images directly.
	return fmt.Errorf("kernel and initrd extraction not implemented for Fedora from cloud image")
}

func (e *FedoraExtractor) CreateRootfs(ctx context.Context, distroInfo *config.ArchInfo, distroPath string) error {
	// Not implemented for Fedora
	return fmt.Errorf("not implemented for Fedora yet")
}
