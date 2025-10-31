package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// fetchURL fetches a URL using wget (available in u-root)
func fetchURL(url string) ([]byte, error) {
	// u-root's wget seems to have issues with output capture
	// Write to a temp file and read it back
	tmpfile := "/tmp/fetch-" + filepath.Base(url)

	cmd := exec.Command("wget", "-O", tmpfile, url)
	// Capture both stdout and stderr for debugging
	combinedOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wget failed: %w (output: %s)", err, string(combinedOutput))
	}

	// Read the downloaded file
	data, err := os.ReadFile(tmpfile)
	if err != nil {
		return nil, fmt.Errorf("failed to read temp file: %w", err)
	}

	// Clean up
	os.Remove(tmpfile)

	return data, nil
}

// fetchCloudInitData fetches cloud-init configuration from the given base URL
func fetchCloudInitData(baseURL string) (*CloudInitData, error) {
	fmt.Println("  -> Fetching cloud-init configuration...")
	fmt.Println("  -> Using wget (Go HTTP client has issues in u-root)")

	cloudInit := &CloudInitData{}

	// Fetch meta-data
	fmt.Println("  -> Fetching meta-data...")
	metaDataURL := baseURL + "/meta-data"
	metaDataBytes, err := fetchURL(metaDataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta-data: %w", err)
	}

	cloudInit.MetaData = string(metaDataBytes)

	fmt.Printf("  -> Meta-data fetched (%d bytes)\n", len(metaDataBytes))

	// Fetch user-data
	fmt.Println("  -> Fetching user-data...")
	userDataURL := baseURL + "/user-data"
	userDataBytes, err := fetchURL(userDataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user-data: %w", err)
	}

	cloudInit.UserData = string(userDataBytes)
	fmt.Printf("  -> User-data fetched (%d bytes)\n", len(userDataBytes))

	// Fetch network-config
	fmt.Println("  -> Fetching network-config...")
	networkConfigURL := baseURL + "/network-config"
	networkConfigBytes, err := fetchURL(networkConfigURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch network-config: %w", err)
	}

	cloudInit.NetworkConfig = string(networkConfigBytes)
	fmt.Printf("  -> Network-config fetched (%d bytes)\n", len(networkConfigBytes))

	return cloudInit, nil
}

// configureSystem writes cloud-init configuration to the target system
func configureSystem(cloudInit *CloudInitData) error {
	fmt.Println("  -> Configuring system with cloud-init data...")

	// Create cloud-init directory
	cloudInitDir := "/mnt/target/var/lib/cloud/seed/nocloud-net"
	if err := os.MkdirAll(cloudInitDir, 0755); err != nil {
		return fmt.Errorf("failed to create cloud-init directory: %w", err)
	}

	// Write meta-data
	fmt.Println("  -> Writing meta-data...")
	metaDataPath := filepath.Join(cloudInitDir, "meta-data")
	if err := os.WriteFile(metaDataPath, []byte(cloudInit.MetaData), 0644); err != nil {
		return fmt.Errorf("failed to write meta-data: %w", err)
	}

	// Write user-data
	fmt.Println("  -> Writing user-data...")
	userDataPath := filepath.Join(cloudInitDir, "user-data")
	if err := os.WriteFile(userDataPath, []byte(cloudInit.UserData), 0644); err != nil {
		return fmt.Errorf("failed to write user-data: %w", err)
	}

	// Write network-config
	fmt.Println("  -> Writing network-config...")
	networkConfigPath := filepath.Join(cloudInitDir, "network-config")
	if err := os.WriteFile(networkConfigPath, []byte(cloudInit.NetworkConfig), 0644); err != nil {
		return fmt.Errorf("failed to write network-config: %w", err)
	}

	fmt.Println("  -> Cloud-init configuration written")

	return nil
}
