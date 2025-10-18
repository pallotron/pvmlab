package metadata

import (
	"os"
	"pvmlab/internal/config"
	"testing"
)

func TestCheckForDuplicateIPs(t *testing.T) {
	tests := []struct {
		name        string
		vms         map[string]*Metadata
		newIP       string
		newIPv6     string
		expectError bool
	}{
		{
			name: "no existing VMs",
			vms:  map[string]*Metadata{},
			newIP:   "192.168.1.1/24",
			newIPv6: "fd00::1/64",
			expectError: false,
		},
		{
			name: "duplicate IPv4",
			vms: map[string]*Metadata{
				"vm1": {IP: "192.168.1.1"},
			},
			newIP:   "192.168.1.1/24",
			newIPv6: "fd00::1/64",
			expectError: true,
		},
		{
			name: "duplicate IPv6",
			vms: map[string]*Metadata{
				"vm1": {IPv6: "fd00::1"},
			},
			newIP:   "192.168.1.1/24",
			newIPv6: "fd00::1/64",
			expectError: true,
		},
		{
			name: "no duplicates",
			vms: map[string]*Metadata{
				"vm1": {IP: "192.168.1.2", IPv6: "fd00::2"},
			},
			newIP:   "192.168.1.1/24",
			newIPv6: "fd00::1/64",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the tests
			tmpDir, err := os.MkdirTemp("", "pvmlab-test")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create a mock config
			cfg := &config.Config{}
			cfg.SetHomeDir(tmpDir)

			// Create dummy VM metadata files
			for vmName, meta := range tt.vms {
				if err := Save(cfg, vmName, meta.Role, meta.Arch, meta.IP, meta.Subnet, meta.IPv6, meta.SubnetV6, meta.MAC, meta.PxeBootStackTar, meta.DockerImagesPath, meta.VMsPath, meta.SSHPort, meta.PxeBoot); err != nil {
					t.Fatalf("failed to save dummy metadata: %v", err)
				}
			}

			// Call the function
			err = CheckForDuplicateIPs(cfg, tt.newIP, tt.newIPv6)

			// Check the result
			if tt.expectError && err == nil {
				t.Errorf("expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}
