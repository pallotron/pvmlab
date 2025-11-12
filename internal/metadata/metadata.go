package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"pvmlab/internal/config"
)

type Metadata struct {
	Name             string `json:"name"`
	Role             string `json:"role"`
	Arch             string `json:"arch"`
	IP               string `json:"ip,omitempty"`
	Subnet           string `json:"subnet,omitempty"`
	IPv6             string `json:"ipv6,omitempty"`
	SubnetV6         string `json:"subnetv6,omitempty"`
	MAC              string `json:"mac,omitempty"`
	PxeBootStackTar  string `json:"pxe_boot_stack_tar,omitempty"`
	DockerImagesPath string `json:"docker_images_path,omitempty"`
	VMsPath          string `json:"vms_path,omitempty"`
	SSHPort          int    `json:"ssh_port,omitempty"`
	PxeBoot          bool   `json:"pxeboot,omitempty"`
	Distro           string `json:"distro,omitempty"`
	SSHKey           string `json:"ssh_key,omitempty"`
	Kernel           string `json:"kernel,omitempty"`
	Initrd           string `json:"initrd,omitempty"`
}

func getVMsDir(cfg *config.Config) string {
	return filepath.Join(cfg.GetAppDir(), "vms")
}

// Save saves the VM's metadata to a file.
var Save = func(cfg *config.Config, vmName, role, arch, ip, subnet, ipv6, subnetv6, mac, pxeBootStackTar, dockerImagesPath, vmsPath, sshKey, kernel, initrd string, sshPort int, pxeBoot bool, distro string) error {
	meta := Metadata{
		Name:             vmName,
		Role:             role,
		Arch:             arch,
		IP:               ip,
		Subnet:           subnet,
		IPv6:             ipv6,
		SubnetV6:         subnetv6,
		MAC:              mac,
		PxeBootStackTar:  pxeBootStackTar,
		DockerImagesPath: dockerImagesPath,
		VMsPath:          vmsPath,
		SSHPort:          sshPort,
		PxeBoot:          pxeBoot,
		Distro:           distro,
		SSHKey:           sshKey,
		Kernel:           kernel,
		Initrd:           initrd,
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

var Delete = func(cfg *config.Config, vmName string) error {
	vmsDir := getVMsDir(cfg)
	metaPath := filepath.Join(vmsDir, vmName+".json")
	if _, err := os.Stat(metaPath); err == nil {
		return os.Remove(metaPath)
	}
	return nil
}

var GetAll = func(cfg *config.Config) (map[string]*Metadata, error) {
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