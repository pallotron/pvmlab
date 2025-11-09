package cmd

import (
	"errors"
	"pvmlab/internal/cloudinit"
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
			            args: []string{"vm", "create", "test-vm"},			setupMocks: func() {
				metadata.FindVM = func(c *config.Config, name string) (string, error) {
					return "test-vm", nil
				}
			},
			expectedError: "a VM named 'test-vm' already exists",
		},
		{
			            args: []string{"vm", "create", "test-vm"},			setupMocks: func() {
				createDisk = func(string, string, string) error {
					return errors.New("qemu-img failed")
				}
			},
			expectedError: "qemu-img failed",
		},
		{
			name: "iso create failure",
			args: []string{"vm", "create", "test-vm"},
			setupMocks: func() {
				cloudinit.CreateISO = func(string, string, string, string, string, string, string, string, string) error {
					return errors.New("iso creation failed")
				}
			},
			expectedError: "iso creation failed",
		},
		{
			name: "metadata save failure (warning)",
			args: []string{"vm", "create", "test-vm"},
			setupMocks: func() {
				metadata.Save = func(c *config.Config, _, _, _, _, _, _, _, _, _, _, _, sshKey string, i int, _ bool, _ string) error {
					return errors.New("metadata save failed")
				}
			},
			expectedError: "", // Should not error out, just warn
			expectedOut:   "Warning: failed to save VM metadata",
		},
		{
			name: "create target success",
			args: []string{"vm", "create", "my-target"},
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
