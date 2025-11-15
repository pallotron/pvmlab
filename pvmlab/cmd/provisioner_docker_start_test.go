package cmd

import (
	"os"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"strings"
	"testing"
)

func TestProvisionerDockerStartCommand(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(t *testing.T) []string
		expectedError string
		expectedOut   string
	}{
		{
			name: "missing required docker-tar flag",
			setupMocks: func(t *testing.T) []string {
				return []string{"provisioner", "docker", "start"}
			},
			expectedError: "required flag(s) \"docker-tar\" not set",
		},
		{
			name: "no provisioner found",
			setupMocks: func(t *testing.T) []string {
				metadata.FindProvisioner = func(*config.Config) (string, error) {
					return "", nil
				}
				return []string{"provisioner", "docker", "start", "--docker-tar", "/tmp/test.tar"}
			},
			expectedError: "no provisioner found",
		},
		{
			name: "provisioner metadata load error",
			setupMocks: func(t *testing.T) []string {
				metadata.FindProvisioner = func(*config.Config) (string, error) {
					return "test-provisioner", nil
				}
				metadata.Load = func(*config.Config, string) (*metadata.Metadata, error) {
					return nil, &os.PathError{Op: "load", Path: "metadata", Err: os.ErrNotExist}
				}
				return []string{"provisioner", "docker", "start", "--docker-tar", "/tmp/test.tar"}
			},
			expectedError: "error loading VM metadata",
		},
		{
			name: "provisioner not running (SSH port is 0)",
			setupMocks: func(t *testing.T) []string {
				// Create a temporary tar file for this test
				tarFile, err := os.CreateTemp("", "test-*.tar")
				if err != nil {
					t.Fatalf("failed to create temp tar file: %v", err)
				}
				tarFile.Close()
				t.Cleanup(func() { os.Remove(tarFile.Name()) })

				metadata.FindProvisioner = func(*config.Config) (string, error) {
					return "test-provisioner", nil
				}
				metadata.Load = func(*config.Config, string) (*metadata.Metadata, error) {
					return &metadata.Metadata{
						SSHPort: 0,
					}, nil
				}

				return []string{"provisioner", "docker", "start", "--docker-tar", tarFile.Name()}
			},
			expectedError: "SSH port not found in metadata",
		},
		{
			name: "tarball file does not exist",
			setupMocks: func(t *testing.T) []string {
				metadata.FindProvisioner = func(*config.Config) (string, error) {
					return "test-provisioner", nil
				}
				metadata.Load = func(*config.Config, string) (*metadata.Metadata, error) {
					return &metadata.Metadata{
						SSHPort: 2222,
					}, nil
				}
				return []string{"provisioner", "docker", "start", "--docker-tar", "/nonexistent/test.tar"}
			},
			expectedError: "error opening source tarball",
		},
		// Note: Testing successful execution would require mocking exec.Command
		// and file I/O operations, which is more complex. The above tests cover
		// the error paths, validation logic, and flag parsing in the command.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original metadata.Load
			originalMetadataLoad := metadata.Load
			defer func() { metadata.Load = originalMetadataLoad }()

			setupMocks(t)
			args := tt.setupMocks(t)

			output, _, err := executeCommand(rootCmd, args...)

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
