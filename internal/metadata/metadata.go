package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"pvmlab/internal/config"
)

type Metadata struct {
	Role             string `json:"role"`
	IP               string `json:"ip,omitempty"`
	Subnet           string `json:"subnet,omitempty"`
	MAC              string `json:"mac,omitempty"`
	PxeBootStackTar  string `json:"pxe_boot_stack_tar,omitempty"`
	DockerImagesPath string `json:"docker_images_path,omitempty"`
	VMsPath          string `json:"vms_path,omitempty"`
	SSHPort          int    `json:"ssh_port,omitempty"`
}

func getVMsDir(cfg *config.Config) string {
	return filepath.Join(cfg.GetAppDir(), "vms")
}

// Save saves the VM's metadata to a file.
var Save = func(cfg *config.Config, vmName, role, ip, subnet, mac, pxeBootStackTar, dockerImagesPath, vmsPath string, sshPort int) error {
	meta := Metadata{
		Role:             role,
		IP:               ip,
		Subnet:           subnet,
		MAC:              mac,
		PxeBootStackTar:  pxeBootStackTar,
		DockerImagesPath: dockerImagesPath,
		VMsPath:          vmsPath,
		SSHPort:          sshPort,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	vmsDir := getVMsDir(cfg)

	if err := os.MkdirAll(vmsDir, 0755); err != nil {
		return fmt.Errorf("failed to create vms directory: %w", err)
	}

	metaPath := filepath.Join(vmsDir, vmName+".json")
	return os.WriteFile(metaPath, data, 0644)
}

var Load = func(cfg *config.Config, vmName string) (*Metadata, error) {
	vmsDir := getVMsDir(cfg)

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

var FindProvisioner = func(cfg *config.Config) (string, error) {
	vmsDir := getVMsDir(cfg)

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
			meta, err := Load(cfg, vmName)
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

func GetProvisioner(cfg *config.Config) (*Metadata, error) {
	vms, err := GetAll(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not get all VMs: %w", err)
	}

	for _, meta := range vms {
		if meta.Role == "provisioner" {
			return meta, nil
		}
	}

	return nil, fmt.Errorf("no provisioner found")
}

var FindVM = func(cfg *config.Config, vmName string) (string, error) {
	vmsDir := getVMsDir(cfg)
	metaPath := filepath.Join(vmsDir, vmName+".json")
	if _, err := os.Stat(metaPath); err == nil {
		return vmName, nil
	}
	return "", nil
}

func Delete(cfg *config.Config, vmName string) error {
	vmsDir := getVMsDir(cfg)
	metaPath := filepath.Join(vmsDir, vmName+".json")
	if _, err := os.Stat(metaPath); err == nil {
		return os.Remove(metaPath)
	}
	return nil
}

func GetAll(cfg *config.Config) (map[string]*Metadata, error) {
	vmsDir := getVMsDir(cfg)

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
			meta, err := Load(cfg, vmName)
			if err != nil {
				// Log or ignore malformed metadata files
				continue
			}
			allMeta[vmName] = meta
		}
	}

	return allMeta, nil
}