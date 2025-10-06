package config

import (
	"os"
	"path/filepath"
)

const (
	// AppName is the name of the application
	AppName = "provisioning-vm-lab"
	// UbuntuARMImageURL is the URL for the Ubuntu aarch64 cloud image
	UbuntuARMImageURL = "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-arm64.img"
	// UbuntuAMDImageURL is the URL for the Ubuntu amd64 cloud image
	UbuntuAMDImageURL = "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img"
)

// GetAppDir returns the path to the application's hidden directory.
func GetAppDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "."+AppName), nil
}

// GetProjectRootDir returns the root directory of the project.
func GetProjectRootDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
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
