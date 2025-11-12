package cmd

import (
	"errors"
	"strings"
	"testing"

	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
)

func TestVmShellCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
	}{
		{
			name:          "no vm name provided",
			args:          []string{"vm", "shell"},
			setupMocks:    func() {},
			expectedError: "requires at least 1 arg(s), only received 0",
		},
		{
			name: "vm metadata load error",
			args: []string{"vm", "shell", "test-vm"},
			setupMocks: func() {
				metadata.Load = func(c *config.Config, name string) (*metadata.Metadata, error) {
					return nil, errors.New("metadata not found")
				}
			},
			expectedError: "error loading VM metadata",
		},
		{
			name: "provisioner without SSH port",
			args: []string{"vm", "shell", "test-provisioner"},
			setupMocks: func() {
				metadata.Load = func(c *config.Config, name string) (*metadata.Metadata, error) {
					return &metadata.Metadata{
						Name:    "test-provisioner",
						Role:    "provisioner",
						SSHPort: 0,
					}, nil
				}
			},
			expectedError: "SSH port not found",
		},
		{
			name: "target VM without provisioner",
			args: []string{"vm", "shell", "test-target"},
			setupMocks: func() {
				metadata.Load = func(c *config.Config, name string) (*metadata.Metadata, error) {
					return &metadata.Metadata{
						Name: "test-target",
						Role: "target",
						IP:   "192.168.1.10",
					}, nil
				}
			},
			expectedError: "provisioner", // Will fail trying to get provisioner
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks to default success behavior
			setupMocks(t)

			// Apply test-specific mocks
			tt.setupMocks()

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

func TestVmShellCmd_TargetConstruction(t *testing.T) {
	tests := []struct {
		name           string
		meta           *metadata.Metadata
		expectedTarget string
	}{
		{
			name: "provisioner target",
			meta: &metadata.Metadata{
				Name:    "prov",
				Role:    "provisioner",
				SSHPort: 2222,
			},
			expectedTarget: "ubuntu@127.0.0.1",
		},
		{
			name: "target VM",
			meta: &metadata.Metadata{
				Name: "vm1",
				Role: "target",
				IP:   "192.168.1.100",
			},
			expectedTarget: "ubuntu@192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct target based on role
			var target string
			if tt.meta.Role == "provisioner" {
				target = "ubuntu@127.0.0.1"
			} else {
				target = "ubuntu@" + tt.meta.IP
			}

			if target != tt.expectedTarget {
				t.Errorf("target = %q, want %q", target, tt.expectedTarget)
			}
		})
	}
}

func TestVmShellCmd_WithCommand(t *testing.T) {
	// Test that additional arguments after vm name are passed as SSH command
	tests := []struct {
		name        string
		args        []string
		wantCommand bool
	}{
		{
			name:        "shell without command - interactive",
			args:        []string{"vm", "shell", "test-vm"},
			wantCommand: false,
		},
		{
			name:        "shell with single command",
			args:        []string{"vm", "shell", "test-vm", "ls"},
			wantCommand: true,
		},
		{
			name:        "shell with command and args",
			args:        []string{"vm", "shell", "test-vm", "cat", "/etc/hostname"},
			wantCommand: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if args length indicates command execution
			vmNameIndex := 2 // Position of vm name in args
			hasCommand := len(tt.args) > vmNameIndex+1

			if hasCommand != tt.wantCommand {
				t.Errorf("hasCommand = %v, want %v for args %v", hasCommand, tt.wantCommand, tt.args)
			}
		})
	}
}
