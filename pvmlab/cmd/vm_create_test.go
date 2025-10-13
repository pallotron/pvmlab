package cmd

import (
	"errors"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"strings"
	"testing"
)

func TestVMCreateCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name:          "missing vm name",
			args:          []string{"vm", "create"},
			setupMocks:    func() {},
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name:          "invalid role",
			args:          []string{"vm", "create", "test-vm", "--role", "invalid"},
			setupMocks:    func() {},
			expectedError: "--role must be either 'provisioner' or 'target'",
		},
		{
			name:          "provisioner missing ip",
			args:          []string{"vm", "create", "test-vm", "--role", "provisioner"},
			setupMocks:    func() {},
			expectedError: "--ip must be specified for provisioner VMs",
		},
		{
			name: "existing provisioner",
			args: []string{"vm", "create", "new-prov", "--role", "provisioner", "--ip", "1.2.3.4/24"},
			setupMocks: func() {
				metadata.FindProvisioner = func(c *config.Config) (string, error) {
					return "existing-prov", nil
				}
			},
			expectedError: "a provisioner VM named 'existing-prov' already exists",
		},
		{
			name: "existing target vm",
			args: []string{"vm", "create", "test-vm", "--role", "target"},
			setupMocks: func() {
				metadata.FindVM = func(c *config.Config, name string) (string, error) {
					return "test-vm", nil
				}
			},
			expectedError: "a VM named 'test-vm' already exists",
		},
		{
			name: "disk create failure",
			args: []string{"vm", "create", "test-vm", "--role", "target"},
			setupMocks: func() {
				createDisk = func(string, string, string) error {
					return errors.New("qemu-img failed")
				}
			},
			expectedError: "qemu-img failed",
		},
		{
			name: "iso create failure",
			args: []string{"vm", "create", "test-vm", "--role", "target"},
			setupMocks: func() {
				createISO = func(string, string, string, string, string, string, string) error {
					return errors.New("iso creation failed")
				}
			},
			expectedError: "iso creation failed",
		},
		{
			name: "metadata save failure (warning)",
			args: []string{"vm", "create", "test-vm", "--role", "target"},
			setupMocks: func() {
				metadata.Save = func(c *config.Config, _, _, _, _, _, _, _, _ string, i int) error {
					return errors.New("metadata save failed")
				}
			},
			expectedError: "", // Should not error out, just warn
			expectedOut:   "Warning: failed to save VM metadata",
		},
		{
			name: "create provisioner success",
			args: []string{"vm", "create", "my-prov", "--role", "provisioner", "--ip", "192.168.1.1/24"},
			setupMocks: func() {
				// All mocks default to success
			},
			expectedError: "",
			expectedOut:   "VM 'my-prov' created successfully",
		},
		{
			name: "create target success",
			args: []string{"vm", "create", "my-target", "--role", "target"},
			setupMocks: func() {
				// All mocks default to success
			},
			expectedError: "",
			expectedOut:   "VM 'my-target' created successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags to their default values before each test
			ip = ""
			role = "target"
			mac = ""
			pxebootStackTar = "pxeboot_stack.tar"
			dockerImagesPath = ""
			vmsPath = ""
			diskSize = "10G"

			// Reset mocks to default success behavior before each test
			setupMocks(t)
			// Apply test-specific mock setup
			tt.setupMocks()

			output, _, err := executeCommand(rootCmd, tt.args...)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected an error, but got none. output: %s", output)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', but got '%v'", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, but got: %v", err)
				}
			}

			if tt.expectedOut != "" {
				if !strings.Contains(output, tt.expectedOut) {
					t.Errorf("expected output to contain '%s', but got '%s'", tt.expectedOut, output)
				}
			}
		})
	}
}
