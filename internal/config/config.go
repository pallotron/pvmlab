package config

import (
	"os"
	"path/filepath"
)

const (
	// AppName is the name of the application
	AppName = "provisioning-vm-lab"
	// UbuntuCloudImageBaseURL is the base URL for Ubuntu cloud images
	UbuntuCloudImageBaseURL = "https://cloud-images.ubuntu.com/releases/24.04/release/"
	// UbuntuARMImageName is the filename for the Ubuntu aarch64 cloud image
	UbuntuARMImageName = "ubuntu-24.04-server-cloudimg-arm64.img"
	// UbuntuARMImageURL is the full URL for the Ubuntu aarch64 cloud image
	UbuntuARMImageURL = UbuntuCloudImageBaseURL + UbuntuARMImageName
)

// Config holds the application's configuration.
type Config struct {
	homeDir string
}

// New creates a new Config instance.
func New() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Config{homeDir: home}, nil
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
