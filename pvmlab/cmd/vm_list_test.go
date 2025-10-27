package cmd

import (
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/pidfile"
	"strings"
	"testing"
)

func TestVMListCommand(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func()
		expectedOut []string
	}{
		{
			name: "no vms",
			setupMocks: func() {
				// Default GetAll mock returns empty map
			},
			expectedOut: []string{"No VMs have been created yet."},
		},
		{
			name: "one stopped vm",
			setupMocks: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"test-vm": {Name: "test-vm", Role: "target", IP: "1.1.1.1", MAC: "aa:bb:cc"},
					}, nil
				}
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return false, nil
				}
			},
			expectedOut: []string{"test-vm", "1.1.1.1", "aa:bb:cc", "Stopped"},
		},
		{
			name: "one running provisioner",
			setupMocks: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"prov-vm": {Name: "prov-vm", Role: "provisioner", IP: "2.2.2.2", MAC: "dd:ee:ff"},
					}, nil
				}
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return true, nil
				}
			},
			expectedOut: []string{"prov-vm", "2.2.2.2", "dd:ee:ff", "Running"},
		},
		{
			name: "multiple vms with different states",
			setupMocks: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"vm1": {Name: "vm1", Role: "target", IP: "1.1.1.1", MAC: "aa:bb:cc"},
						"vm2": {Name: "vm2", Role: "provisioner", IP: "2.2.2.2", MAC: "dd:ee:ff"},
					}, nil
				}
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return name == "vm2", nil // Only vm2 is running
				}
			},
			expectedOut: []string{"vm1", "Stopped", "vm2", "Running"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.setupMocks()

			output, _, err := executeCommand(rootCmd, "vm", "list")
			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			for _, expected := range tt.expectedOut {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain '%s', but got '%s'", expected, output)
				}
			}
		})
	}
}
