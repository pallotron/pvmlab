package distro

import (
	"fmt"

	"pvmlab/internal/config"
)

// FedoraExtractor implements the Extractor interface for Fedora.
type FedoraExtractor struct{}

func (e *FedoraExtractor) ExtractKernelAndModules(cfg *config.Config, distroInfo *config.ArchInfo, isoPath, distroPath string) error {
	return fmt.Errorf("fedora extraction is not yet implemented")
}
