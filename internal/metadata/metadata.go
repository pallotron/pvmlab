package metadata

import (
	"encoding/json"
	"fmt"
	"net"
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

var GetProvisioner = func(cfg *config.Config) (*Metadata, error) {
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

var CheckForDuplicateIPs = func(cfg *config.Config, ip, ipv6 string) error {
	allVMs, err := GetAll(cfg)
	if err != nil {
		return fmt.Errorf("failed to get all VMs: %w", err)
	}

	var newIP net.IP
	if ip != "" {
		var err error
		newIP, _, err = net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("invalid new ip/cidr '%s': %w", ip, err)
		}
	}

	var newIPv6 net.IP
	if ipv6 != "" {
		var err error
		newIPv6, _, err = net.ParseCIDR(ipv6)
		if err != nil {
			return fmt.Errorf("invalid new ipv6/cidr '%s': %w", ipv6, err)
		}
	}

	for vmName, meta := range allVMs {
		if meta.IP != "" && newIP != nil {
			existingIP := net.ParseIP(meta.IP)
			if existingIP == nil {
				return fmt.Errorf("invalid IP address format for existing VM '%s': %s", vmName, meta.IP)
			}
			if newIP.Equal(existingIP) {
				return fmt.Errorf("IP address from %s is already in use by VM '%s'", ip, vmName)
			}
		}
		if meta.IPv6 != "" && newIPv6 != nil {
			existingIPv6 := net.ParseIP(meta.IPv6)
			if existingIPv6 == nil {
				return fmt.Errorf("invalid IPv6 address format for existing VM '%s': %s", vmName, meta.IPv6)
			}
			if newIPv6.Equal(existingIPv6) {
				return fmt.Errorf("IPv6 address from %s is already in use by VM '%s'", ipv6, vmName)
			}
		}
	}

	return nil
}
