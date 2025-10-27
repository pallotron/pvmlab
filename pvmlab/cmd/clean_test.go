package cmd

import (
	"errors"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/pidfile"
	"pvmlab/internal/socketvmnet"
	"strings"
	"testing"
)

func TestCleanCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name: "clean with no vms",
			args: []string{"clean"},
			setupMocks: func() {
				// Default mock for GetAll returns an empty map, so no setup needed.
			},
			expectedError: "",
			expectedOut:   "No VMs found to clean.",
		},
		{
			name: "clean with one vm",
			args: []string{"clean"},
			setupMocks: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"test-vm": {Name: "test-vm"},
					}, nil
				}
				// Mock that the VM is not running, so stop is skipped.
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return false, nil
				}
				socketvmnet.StopSocketVmnet = func() error { return nil }
			},
			expectedError: "",
			expectedOut:   "Cleaning directory:",
		},
		{
			name: "clean with purge flag",
			args: []string{"clean", "--purge"},
			setupMocks: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"test-vm": {Name: "test-vm"},
					}, nil
				}
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return false, nil
				}
				socketvmnet.StopSocketVmnet = func() error { return nil }
			},
			expectedError: "",
			expectedOut:   "Purging entire pvmlab directory",
		},
		{
			name: "getall returns error",
			args: []string{"clean"},
			setupMocks: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return nil, errors.New("metadata read error")
				}
			},
			expectedError: "metadata read error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.setupMocks()

			// Reset flag
			purge = false

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
