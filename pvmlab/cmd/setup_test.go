package cmd

import (
	"fmt"
	"os/exec"
	"pvmlab/internal/socketvmnet"
	"pvmlab/internal/ssh"
	"strings"
	"testing"
)

func TestSetupCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name: "setup success",
			args: []string{"setup"},
			setupMocks: func() {
				ssh.GenerateKey = func(string) error { return nil }

			},
			expectedError: "",
			expectedOut:   "Setup completed successfully",
		},
		{
			name: "ssh key generation fails",
			args: []string{"setup"},
			setupMocks: func() {
				ssh.GenerateKey = func(string) error {
					return fmt.Errorf("ssh key failed")
				}
			},
			expectedError: "ssh key failed",
		},
		{
			name: "dependency check fails",
			args: []string{"setup"},
			setupMocks: func() {
				// Mock successful key generation
				ssh.GenerateKey = func(string) error { return nil }
				// Mock execCommand to fail on the first dependency check
				execCommand = func(name string, arg ...string) *exec.Cmd {
					cmd := exec.Command("false")
					return cmd
				}
			},
			expectedError: "brew not found",
		},
		{
			name: "socket_vmnet not running",
			args: []string{"setup"},
			setupMocks: func() {
				ssh.GenerateKey = func(string) error { return nil }
				socketvmnet.IsSocketVmnetRunning = func() (bool, error) { return false, nil }
			},
			expectedError: "",
			expectedOut:   "Setup completed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks to default success behavior
			ssh.GenerateKey = func(string) error { return nil }
			socketvmnet.IsSocketVmnetRunning = func() (bool, error) { return true, nil }
			execCommand = func(_ string, _ ...string) *exec.Cmd {
				cmd := exec.Command("true")
				return cmd
			}
			// Apply test-specific mock setup
			tt.setupMocks()

			// The --assets-only flag is used in the real setup command for integration tests,
			// but we don't need it for unit tests as we mock the dependency checks.
			// However, we must reset the flag variable itself.
			assetsOnly = false

			output, _, err := executeCommand(rootCmd, tt.args...)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected an error containing '%s', but got none. output: %s", tt.expectedError, output)
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
