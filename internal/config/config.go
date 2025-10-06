package config

import (
	"os"
	"path/filepath"
)

const (
	// AppName is the name of the application
	AppName = "provisioning-vm-lab"
	// UbuntuARMImageURL is the URL for the Ubuntu aarch64 cloud image
	UbuntuARMImageURL = "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-arm64.img"
)

// GetAppDir returns the path to the application's hidden directory.
func GetAppDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "."+AppName), nil
}
