package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
)

type Metadata struct {
	Role             string `json:"role"`
	IP               string `json:"ip,omitempty"`
	MAC              string `json:"mac,omitempty"`
	PxeBootStackTar  string `json:"pxe_boot_stack_tar,omitempty"`
	DockerImagesPath string `json:"docker_images_path,omitempty"`
}

func getVMsDir() (string, error) {
	appDir, err := config.GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "vms"), nil
}

// Save saves the VM's metadata to a file.
func Save(vmName, role, ip, mac, pxeBootStackTar, dockerImagesPath string) error {
	meta := Metadata{
		Role:             role,
		IP:               ip,
		MAC:              mac,
		PxeBootStackTar:  pxeBootStackTar,
		DockerImagesPath: dockerImagesPath,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	vmsDir, err := getVMsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(vmsDir, 0755); err != nil {
		return fmt.Errorf("failed to create vms directory: %w", err)
	}

	metaPath := filepath.Join(vmsDir, vmName+".json")
	return os.WriteFile(metaPath, data, 0644)
}

func Load(vmName string) (*Metadata, error) {
	vmsDir, err := getVMsDir()
	if err != nil {
		return nil, err
	}

	metaPath := filepath.Join(vmsDir, vmName+".json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for %s: %w", vmName, err)
	}
	return &meta, nil
}

func FindProvisioner() (string, error) {
	vmsDir, err := getVMsDir()
	if err != nil {
		return "", err
	}

	files, err := os.ReadDir(vmsDir)
	if err != nil {
		// If the directory doesn't exist, no provisioner can exist.
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			vmName := file.Name()[:len(file.Name())-len(".json")]
			meta, err := Load(vmName)
			if err != nil {
				// Ignore malformed metadata files
				continue
			}
			if meta.Role == "provisioner" {
				return vmName, nil
			}
		}
	}

	return "", nil
}

func FindVM(vmName string) (string, error) {
	vmsDir, err := getVMsDir()
	if err != nil {
		return "", err
	}
	metaPath := filepath.Join(vmsDir, vmName+".json")
	if _, err := os.Stat(metaPath); err == nil {
		return vmName, nil
	}
	return "", nil
}

func Delete(vmName string) error {
	vmsDir, err := getVMsDir()
	if err != nil {
		return err
	}
	metaPath := filepath.Join(vmsDir, vmName+".json")
	if _, err := os.Stat(metaPath); err == nil {
		return os.Remove(metaPath)
	}
	return nil
}

func GetAll() (map[string]*Metadata, error) {
	vmsDir, err := getVMsDir()
	if err != nil {
		return nil, err
	}

	allMeta := make(map[string]*Metadata)

	files, err := os.ReadDir(vmsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return allMeta, nil // No directory means no VMs
		}
		return nil, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			vmName := file.Name()[:len(file.Name())-len(".json")]
			meta, err := Load(vmName)
			if err != nil {
				// Log or ignore malformed metadata files
				continue
			}
			allMeta[vmName] = meta
		}
	}

	return allMeta, nil
}
