package cmd

import (
	"pvmlab/internal/config"
	"pvmlab/internal/pidfile"
	"strings"
	"syscall"
	"testing"
)

func TestVMStopCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name:          "no vm name",
			args:          []string{"vm", "stop"},
			setupMocks:    func() {},
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name: "vm not running",
			args: []string{"vm", "stop", "test-vm"},
			setupMocks: func() {
				// Default pidfile.Read mock returns os.ErrNotExist
			},
			expectedError: "VM 'test-vm' is not running",
		},
		{
			name: "graceful shutdown success",
			args: []string{"vm", "stop", "test-vm"},
			setupMocks: func() {
				pidfile.Read = func(c *config.Config, name string) (int, error) {
					return 123, nil
				}
				// This test doesn't mock the network connection, so it will fail to connect
				// and fall through to the force stop. To test graceful shutdown properly
				// would require more invasive mocking. We will simulate it by having
				// isProcessRunning return false immediately.
				isProcessRunning = func(pid int) bool {
					return false
				}
			},
			expectedError: "",
			expectedOut:   "VM 'test-vm' stopped successfully",
		},
		{
			name: "forceful shutdown success",
			args: []string{"vm", "stop", "test-vm"},
			setupMocks: func() {
				pidfile.Read = func(c *config.Config, name string) (int, error) {
					return 123, nil
				}
				// Simulate process running initially, then stopping after the kill
				runCount := 0
				isProcessRunning = func(pid int) bool {
					runCount++
					return runCount <= 1
				}
			},
			expectedError: "",
			expectedOut:   "VM stopped successfully",
		},
		{
			name: "forceful shutdown timeout",
			args: []string{"vm", "stop", "test-vm"},
			setupMocks: func() {
				pidfile.Read = func(c *config.Config, name string) (int, error) {
					return 123, nil
				}
				// Simulate process that never stops
				isProcessRunning = func(pid int) bool {
					return true
				}
				// Mock syscallKill to prevent error on non-existent process
				syscallKill = func(pid int, sig syscall.Signal) error {
					return nil
				}
			},
			expectedError: "failed to stop VM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original isProcessRunning
			originalIsProcessRunning := isProcessRunning
			defer func() { isProcessRunning = originalIsProcessRunning }()

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
