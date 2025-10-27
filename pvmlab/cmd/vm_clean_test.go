package cmd

import (
	"errors"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/pidfile"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVMCleanCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name:          "no vm name without --all",
			args:          []string{"vm", "clean"},
			setupMocks:    func() {},
			expectedError: "a vm-name is required when --all is not specified",
		},
		{
			name: "clean single vm success",
			args: []string{"vm", "clean", "test-vm"},
			setupMocks: func() {
				// Mock that VM is not running, so stop is skipped
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return false, nil
				}
			},
			expectedError: "",
			expectedOut:   "VM 'test-vm' files cleaned successfully",
		},
		{
			name: "clean single vm that is running",
			args: []string{"vm", "clean", "test-vm"},
			setupMocks: func() {
				// Mock that VM is running initially, then not running after stop
				var isRunning = true
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					if isRunning {
						isRunning = false
						return true, nil
					}
					return false, nil
				}
				// Mock a successful stop command
				vmStopCmd.RunE = func(cmd *cobra.Command, args []string) error {
					return nil
				}
			},
			expectedError: "",
			expectedOut:   "VM 'test-vm' files cleaned successfully",
		},
		{
			name: "clean --all with no vms",
			args: []string{"vm", "clean", "--all"},
			setupMocks: func() {
				// Default GetAll mock returns empty map
			},
			expectedError: "",
			expectedOut:   "No VMs found to clean",
		},
		{
			name: "clean --all with multiple vms",
			args: []string{"vm", "clean", "--all"},
			setupMocks: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"vm1": {Name: "vm1"},
						"vm2": {Name: "vm2"},
					}, nil
				}
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return false, nil
				}
			},
			expectedError: "",
			expectedOut:   "All VMs cleaned",
		},
		{
			name: "metadata delete fails (warning)",
			args: []string{"vm", "clean", "test-vm"},
			setupMocks: func() {
				metadata.Delete = func(c *config.Config, name string) error {
					return errors.New("delete failed")
				}
			},
			expectedError: "",
			expectedOut:   "Warning: could not remove metadata file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original vmStopCmd.RunE
			originalStopRunE := vmStopCmd.RunE
			defer func() { vmStopCmd.RunE = originalStopRunE }()

			setupMocks(t)
			tt.setupMocks()

			// Reset flag
			cleanAll = false

			output, _, err := executeCommand(rootCmd, tt.args...)

			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', but got '%v'", tt.expectedError, err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			if tt.expectedOut != "" && !strings.Contains(output, tt.expectedOut) {
				t.Errorf("expected output to contain '%s', but got '%s'", tt.expectedOut, output)
			}
		})
	}
}
