package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// AppName is the name of the application
	AppName = "pvmlab"
)

//go:embed distros.yaml
var defaultDistrosYAML []byte

var (
	// Version is the version of the application. It is set at build time.
	Version = "devel"
	// Distros holds the configuration for all supported distributions.
	Distros = make(map[string]Distro)
)

// Distro represents a distribution that can be used for PXE booting.
type Distro struct {
	Name       string              `yaml:"name"`
	// DistroName is the "family" name of the distribution (e.g., "ubuntu", "fedora").
	// This is used by the extractor factory to determine which extraction logic to use.
	DistroName string              `yaml:"distro_name"`
	Version    string              `yaml:"version"`
	Arch       map[string]ArchInfo `yaml:"arch"`
}

// ArchInfo contains architecture-specific information for a distribution.
type ArchInfo struct {
	Qcow2URL   string `yaml:"qcow2_url"`
	KernelPath string `yaml:"kernel_path"`
	InitrdPath string `yaml:"initrd_path"`
}

// LoadOrCreateDistros loads the distro configurations from the user's app directory.
// If the config file doesn't exist, it's created from the embedded default.
func (c *Config) LoadOrCreateDistros() error {
	distrosPath := filepath.Join(c.GetAppDir(), "distros.yaml")

	if _, err := os.Stat(distrosPath); os.IsNotExist(err) {
		if err := os.MkdirAll(c.GetAppDir(), 0755); err != nil {
			return fmt.Errorf("failed to create app directory: %w", err)
		}
		if err := os.WriteFile(distrosPath, defaultDistrosYAML, 0644); err != nil {
			return fmt.Errorf("failed to write default distros config: %w", err)
		}
	}

	data, err := os.ReadFile(distrosPath)
	if err != nil {
		return fmt.Errorf("failed to read distros config: %w", err)
	}

	var distroList []Distro
	if err := yaml.Unmarshal(data, &distroList); err != nil {
		return fmt.Errorf("failed to parse distros config: %w", err)
	}

	// Clear the map before loading to ensure no stale data
	Distros = make(map[string]Distro)
	for _, d := range distroList {
		Distros[d.Name] = d
	}

	return nil
}

// GetDistro returns the configuration for a specific distro and architecture.
var GetDistro = func(distroName, arch string) (*ArchInfo, error) {
	distro, ok := Distros[distroName]
	if !ok {
		return nil, fmt.Errorf("unsupported distro: %s", distroName)
	}
	archInfo, ok := distro.Arch[arch]
	if !ok {
		return nil, fmt.Errorf("unsupported architecture '%s' for distro '%s'", arch, distroName)
	}
	return &archInfo, nil
}

// GetProvisionerImageURL returns the URL for the provisioner image based on the
// application version and architecture.
func GetProvisionerImageURL(arch string) (string, string) {

	var imageName string
	if arch == "aarch64" {
		imageName = "provisioner-custom.arm64.qcow2"
	} else {
		imageName = "provisioner-custom.amd64.qcow2"
	}

	var url string
	if Version == "devel" {
		url = fmt.Sprintf("https://github.com/pallotron/pvmlab/releases/latest/download/%s", imageName)
	} else {
		url = fmt.Sprintf("https://github.com/pallotron/pvmlab/releases/download/%s/%s", Version, imageName)
	}

	return url, imageName
}

// GetPxeBootStackImageURL returns the URL for the pxeboot stack image based on
// the application version.
func GetPxeBootStackImageURL() string {
	version := Version
	if Version == "devel" {
		version = "latest"
	}
	return fmt.Sprintf("ghcr.io/pallotron/pvmlab/pxeboot_stack:%s", version)
}

// GetPxeBootStackImageName returns the local name for the pxeboot stack image.
func GetPxeBootStackImageName() string {
	version := Version
	if Version == "devel" {
		version = "latest"
	}
	return fmt.Sprintf("pxeboot_stack:%s", version)
}

// Config holds the application's configuration.
type Config struct {
	homeDir string
}

// New creates a new Config instance.
var New = func() (*Config, error) {
	var home string
	var err error

	// Check for the override environment variable first.
	// This is useful for testing.
	homeOverride := os.Getenv("PVMLAB_HOME")
	if homeOverride != "" {
		home = homeOverride
	} else {
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}

	cfg := &Config{homeDir: home}
	if err := cfg.LoadOrCreateDistros(); err != nil {
		return nil, fmt.Errorf("failed to load distro configurations: %w", err)
	}

	return cfg, nil
}

// GetAppDir returns the path to the application's hidden directory.
func (c *Config) GetAppDir() string {
	return filepath.Join(c.homeDir, "."+AppName)
}

// SetHomeDir sets the application's home directory.
func (c *Config) SetHomeDir(dir string) {
	c.homeDir = dir
}

// GetProjectRootDir returns the root directory of the project.
func (c *Config) GetProjectRootDir(wd string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		if wd == "/" {
			break
		}
		wd = filepath.Dir(wd)
	}
	return "", os.ErrNotExist
}
