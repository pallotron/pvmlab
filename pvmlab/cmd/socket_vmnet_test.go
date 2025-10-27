package cmd

import (
	"errors"
	"pvmlab/internal/socketvmnet"
	"strings"
	"testing"
)

func TestSocketVmnetStatusCommand(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name: "status when running",
			setupMocks: func() {
				socketvmnet.IsSocketVmnetRunning = func() (bool, error) { return true, nil }
			},
			expectedError: "",
			expectedOut:   "service is running",
		},
		{
			name: "status when stopped",
			setupMocks: func() {
				socketvmnet.IsSocketVmnetRunning = func() (bool, error) { return false, nil }
			},
			expectedError: "",
			expectedOut:   "service is stopped",
		},
		{
			name: "status returns error",
			setupMocks: func() {
				socketvmnet.IsSocketVmnetRunning = func() (bool, error) { return false, errors.New("launchctl error") }
			},
			expectedError: "launchctl error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.setupMocks()

			output, _, err := executeCommand(rootCmd, "socket_vmnet", "status")

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

func TestSocketVmnetStartCommand(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name: "start success",
			setupMocks: func() {
				socketvmnet.StartSocketVmnet = func() error { return nil }
			},
			expectedError: "",
			expectedOut:   "service started successfully",
		},
		{
			name: "start fails",
			setupMocks: func() {
				socketvmnet.StartSocketVmnet = func() error { return errors.New("launchctl start failed") }
			},
			expectedError: "launchctl start failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.setupMocks()

			output, _, err := executeCommand(rootCmd, "socket_vmnet", "start")

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

func TestSocketVmnetStopCommand(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name: "stop success",
			setupMocks: func() {
				socketvmnet.StopSocketVmnet = func() error { return nil }
			},
			expectedError: "",
			expectedOut:   "service stopped successfully",
		},
		{
			name: "stop fails",
			setupMocks: func() {
				socketvmnet.StopSocketVmnet = func() error { return errors.New("launchctl stop failed") }
			},
			expectedError: "launchctl stop failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.setupMocks()

			output, _, err := executeCommand(rootCmd, "socket_vmnet", "stop")

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
