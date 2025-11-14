package cmd

import (
	"os"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"strings"
	"testing"
)

func TestProvisionerDockerStopCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name:          "no container name",
			args:          []string{"provisioner", "docker", "stop"},
			setupMocks:    func() {},
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name: "no provisioner found",
			args: []string{"provisioner", "docker", "stop", "test-container"},
			setupMocks: func() {
				metadata.FindProvisioner = func(*config.Config) (string, error) {
					return "", nil
				}
			},
			expectedError: "no provisioner found",
		},
		{
			name: "provisioner metadata load error",
			args: []string{"provisioner", "docker", "stop", "test-container"},
			setupMocks: func() {
				metadata.FindProvisioner = func(*config.Config) (string, error) {
					return "test-provisioner", nil
				}
				metadata.Load = func(*config.Config, string) (*metadata.Metadata, error) {
					return nil, &os.PathError{Op: "load", Path: "metadata", Err: os.ErrNotExist}
				}
			},
			expectedError: "error loading provisioner metadata",
		},
		{
			name: "provisioner not running (SSH port is 0)",
			args: []string{"provisioner", "docker", "stop", "test-container"},
			setupMocks: func() {
				metadata.FindProvisioner = func(*config.Config) (string, error) {
					return "test-provisioner", nil
				}
				metadata.Load = func(*config.Config, string) (*metadata.Metadata, error) {
					return &metadata.Metadata{
						SSHPort: 0,
					}, nil
				}
			},
			expectedError: "SSH port not found in metadata",
		},
		// Note: Testing successful execution would require mocking exec.Command
		// which is more complex. The above tests cover the error paths and
		// validation logic in the command.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original metadata.Load
			originalMetadataLoad := metadata.Load
			defer func() { metadata.Load = originalMetadataLoad }()

			setupMocks(t)
			tt.setupMocks()

			output, _, err := executeCommand(rootCmd, tt.args...)

			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', but got '%v'", tt.expectedError, err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, but got: %v. Output: %s", err, output)
			}

			if tt.expectedOut != "" && !strings.Contains(output, tt.expectedOut) {
				t.Errorf("expected output to contain '%s', but got '%s'", tt.expectedOut, output)
			}
		})
	}
}
