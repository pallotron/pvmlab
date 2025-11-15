package cmd

import (
	"context"
	"io"
	"os"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/metadata"
	"pvmlab/internal/qemu"
	"pvmlab/internal/ssh"
	"pvmlab/internal/util"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestSuggestNextIP(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true

	tests := []struct {
		name           string
		vms            map[string]*metadata.Metadata
		provisioner    *metadata.Metadata
		expectedOutput string
		expectError    bool
	}{
		{
			name: "No VMs, suggest first IP",
			provisioner: &metadata.Metadata{
				IP: "192.168.100.1",
			},
			vms: map[string]*metadata.Metadata{
				"provisioner": {IP: "192.168.100.1"},
			},
			expectedOutput: "pvmlab vm create <vm-name> --distro <distro> --ip 192.168.100.2/24",
		},
		{
			name: "Contiguous IPs, suggest next",
			provisioner: &metadata.Metadata{
				IP: "192.168.100.1",
			},
			vms: map[string]*metadata.Metadata{
				"provisioner": {IP: "192.168.100.1"},
				"vm1":         {IP: "192.168.100.2"},
				"vm2":         {IP: "192.168.100.3"},
			},
			expectedOutput: "pvmlab vm create <vm-name> --distro <distro> --ip 192.168.100.4/24",
		},
		{
			name: "Gap in IPs, suggest first gap",
			provisioner: &metadata.Metadata{
				IP: "192.168.100.1",
			},
			vms: map[string]*metadata.Metadata{
				"provisioner": {IP: "192.168.100.1"},
				"vm1":         {IP: "192.168.100.2"},
				"vm3":         {IP: "192.168.100.4"},
			},
			expectedOutput: "pvmlab vm create <vm-name> --distro <distro> --ip 192.168.100.3/24",
		},
		{
			name: "Highest IP is not last, suggest correctly",
			provisioner: &metadata.Metadata{
				IP: "192.168.100.1",
			},
			vms: map[string]*metadata.Metadata{
				"provisioner": {IP: "192.168.100.1"},
				"vm1":         {IP: "192.168.100.10"},
				"vm2":         {IP: "192.168.100.5"},
			},
			expectedOutput: "pvmlab vm create <vm-name> --distro <distro> --ip 192.168.100.2/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock metadata functions
			originalGetProvisioner := metadata.GetProvisioner
			metadata.GetProvisioner = func(cfg *config.Config) (*metadata.Metadata, error) {
				return tt.provisioner, nil
			}
			defer func() { metadata.GetProvisioner = originalGetProvisioner }()

			originalGetAll := metadata.GetAll
			metadata.GetAll = func(cfg *config.Config) (map[string]*metadata.Metadata, error) {
				return tt.vms, nil
			}
			defer func() { metadata.GetAll = originalGetAll }()

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := suggestNextIP(&config.Config{})

			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = old

			if (err != nil) != tt.expectError {
				t.Errorf("suggestNextIP() error = %v, wantErr %v", err, tt.expectError)
				return
			}

			if !strings.Contains(string(out), tt.expectedOutput) {
				t.Errorf("suggestNextIP() output = %q, want to contain %q", string(out), tt.expectedOutput)
			}
		})
	}
}

func TestVMCreateCommand(t *testing.T) {
	// Disable color output for consistent testing
	color.NoColor = true

	// Mock external dependencies
	originalConfigNew := config.New
	config.New = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	defer func() { config.New = originalConfigNew }()

	originalGetProvisioner := metadata.GetProvisioner
	metadata.GetProvisioner = func(cfg *config.Config) (*metadata.Metadata, error) {
		return &metadata.Metadata{IP: "192.168.100.1"}, nil
	}
	defer func() { metadata.GetProvisioner = originalGetProvisioner }()

	originalGetAll := metadata.GetAll
	metadata.GetAll = func(cfg *config.Config) (map[string]*metadata.Metadata, error) {
		return map[string]*metadata.Metadata{
			"provisioner": {IP: "192.168.100.1"},
		}, nil
	}
	defer func() { metadata.GetAll = originalGetAll }()

	originalGenerateKey := ssh.GenerateKey
	ssh.GenerateKey = func(privateKeyPath string) error { return nil }
	defer func() { ssh.GenerateKey = originalGenerateKey }()

	originalReadFile := readFile
	readFile = func(name string) ([]byte, error) {
		if strings.HasSuffix(name, ".pub") {
			return []byte("ssh-rsa AAAA..."), nil
		}
		return originalReadFile(name)
	}
	defer func() { readFile = originalReadFile }()

	originalConfigGetDistro := config.GetDistro
	config.GetDistro = func(distroName, arch string) (*config.ArchInfo, error) {
		return &config.ArchInfo{
			Qcow2URL:   "http://example.com/image.qcow2",
			KernelPath: "/boot/vmlinuz",
			InitrdPath: "/boot/initrd",
		}, nil
	}
	defer func() { config.GetDistro = originalConfigGetDistro }()

	originalDownloadImageIfNotExists := downloader.DownloadImageIfNotExists
	downloader.DownloadImageIfNotExists = func(ctx context.Context, imagePath, imageUrl string) error { return nil }
	defer func() { downloader.DownloadImageIfNotExists = originalDownloadImageIfNotExists }()

	originalCreateDisk := createDisk
	createDisk = func(ctx context.Context, imagePath, vmDiskPath, diskSize string) error { return nil }
	defer func() { createDisk = originalCreateDisk }()

	originalCreateBlankDisk := createBlankDisk
	createBlankDisk = func(ctx context.Context, vmDiskPath, diskSize string) error { return nil }
	defer func() { createBlankDisk = originalCreateBlankDisk }()

	originalCloudinitCreateISO := cloudinit.CreateISO
	cloudinit.CreateISO = func(ctx context.Context, vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image string) error { return nil }
	defer func() { cloudinit.CreateISO = originalCloudinitCreateISO }()

	originalQemuGetImageVirtualSize := qemu.GetImageVirtualSize
	qemu.GetImageVirtualSize = func(imagePath string) (int64, error) { return 1024 * 1024 * 1024, nil }
	defer func() { qemu.GetImageVirtualSize = originalQemuGetImageVirtualSize }()

	originalUtilParseSize := util.ParseSize
	util.ParseSize = func(sizeStr string) (int64, error) { return 2 * 1024 * 1024 * 1024, nil }
	defer func() { util.ParseSize = originalUtilParseSize }()

	originalCreateDirectories := createDirectories
	createDirectories = func(appDir string) error { return nil }
	defer func() { createDirectories = originalCreateDirectories }()

	originalCheckForDuplicateIPs := metadata.CheckForDuplicateIPs
	metadata.CheckForDuplicateIPs = func(cfg *config.Config, ip, ipv6 string) error { return nil }
	defer func() { metadata.CheckForDuplicateIPs = originalCheckForDuplicateIPs }()

	originalCheckExistingVMs := checkExistingVMs
	checkExistingVMs = func(cfg *config.Config, vmName, role string) error { return nil }
	defer func() { checkExistingVMs = originalCheckExistingVMs }()

	originalGetMac := getMac
	getMac = func(mac string) (string, error) { return "02:00:00:00:00:01", nil }
	defer func() { getMac = originalGetMac }()

	// Set flags for the command
	vmCreateCmd.Flags().Set("distro", "ubuntu-24.04")
	vmCreateCmd.Flags().Set("arch", "x86_64")
	vmCreateCmd.Flags().Set("ip", "192.168.100.2/24")
	vmCreateCmd.Flags().Set("disk-size", "20G")

	// Execute the command
	err := vmCreateCmd.RunE(vmCreateCmd, []string{"test-vm"})

	// Assert no error
	assert.NoError(t, err, "vmCreateCmd.RunE should not return an error")
}
