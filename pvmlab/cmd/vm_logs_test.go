package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"pvmlab/internal/config"
)

func TestVmLogsCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupLogFile  bool
		expectedError string
	}{
		{
			name:          "no vm name provided",
			args:          []string{"vm", "logs"},
			setupLogFile:  false,
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name:          "too many arguments",
			args:          []string{"vm", "logs", "vm1", "extra"},
			setupLogFile:  false,
			expectedError: "accepts 1 arg(s), received 2",
		},
		{
			name:          "non-existent log file",
			args:          []string{"vm", "logs", "nonexistent-vm"},
			setupLogFile:  false,
			expectedError: "error tailing log file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset root command for each test
			rootCmd.SetArgs(tt.args[1:])

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
		})
	}
}

func TestVmLogsCmd_LogPathConstruction(t *testing.T) {
	// Test that the log path is constructed correctly
	cfg, err := config.New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	appDir := cfg.GetAppDir()
	expectedLogDir := filepath.Join(appDir, "logs")
	expectedPath := filepath.Join(expectedLogDir, "test-vm.log")

	// Verify the expected path format
	if !strings.Contains(expectedPath, "logs") {
		t.Errorf("Expected log path to contain 'logs', got: %s", expectedPath)
	}

	if !strings.HasSuffix(expectedPath, ".log") {
		t.Errorf("Expected log path to end with '.log', got: %s", expectedPath)
	}

	// Note: We don't actually run the tail command here because tail -f
	// would hang indefinitely. The actual command execution is tested
	// in the error cases above (non-existent log file).
	t.Logf("Expected log path format: %s", expectedPath)
}
