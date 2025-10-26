package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// AppName is the name of the application
	AppName = "pvmlab"
	// UbuntuCloudImageBaseURL is the base URL for Ubuntu cloud images
	UbuntuCloudImageBaseURL = "https://cloud-images.ubuntu.com/releases/24.04/release/"
	// UbuntuARMImageName is the filename for the Ubuntu aarch64 cloud image
	UbuntuARMImageName = "ubuntu-24.04-server-cloudimg-arm64.img"
	// UbuntuARMImageURL is the full URL for the Ubuntu aarch64 cloud image
	UbuntuARMImageURL = UbuntuCloudImageBaseURL + UbuntuARMImageName
	// UbuntuAMD64ImageName is the filename for the Ubuntu x86_64 cloud image
	UbuntuAMD64ImageName = "ubuntu-24.04-server-cloudimg-amd64.img"
	// UbuntuAMD64ImageURL is the full URL for the Ubuntu x86_64 cloud image
	UbuntuAMD64ImageURL = UbuntuCloudImageBaseURL + UbuntuAMD64ImageName
)

// Version is the version of the application. It is set at build time.
var Version = "devel"

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
