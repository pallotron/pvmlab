package cmd

import (
	"os"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"strings"
	"testing"
)

func TestProvisionerCreateCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func(t *testing.T)
		expectedError string
		expectedOut   string
	}{
		{
			name:          "missing vm name",
			args:          []string{"provisioner", "create"},
			setupMocks:    func(t *testing.T) {},
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name: "invalid architecture",
			args: []string{"provisioner", "create", "test-provisioner", "--ip", "192.168.1.1/24", "--arch", "invalid"},
			setupMocks: func(t *testing.T) {
				// Reset global flags
				provArch = "invalid"
			},
			expectedError: "--arch must be either 'aarch64' or 'x86_64'",
		},
		{
			name: "missing required IP flag",
			args: []string{"provisioner", "create", "test-provisioner", "--arch", "aarch64"},
			setupMocks: func(t *testing.T) {
				provArch = "aarch64"
				provIP = ""
			},
			expectedError: "--ip must be specified for the provisioner VM",
		},
		{
			name: "invalid IP format",
			args: []string{"provisioner", "create", "test-provisioner", "--ip", "invalid-ip", "--arch", "aarch64"},
			setupMocks: func(t *testing.T) {
				provArch = "aarch64"
				provIP = "invalid-ip"
			},
			expectedError: "invalid IP/CIDR address",
		},
		{
			name: "invalid IPv6 format",
			args: []string{"provisioner", "create", "test-provisioner", "--ip", "192.168.1.1/24", "--ipv6", "invalid-ipv6", "--arch", "aarch64"},
			setupMocks: func(t *testing.T) {
				provArch = "aarch64"
				provIP = "192.168.1.1/24"
				provIPv6 = "invalid-ipv6"
			},
			expectedError: "invalid IPv6/CIDR address",
		},
		{
			name: "duplicate IP error",
			args: []string{"provisioner", "create", "test-provisioner", "--ip", "192.168.1.1/24", "--arch", "aarch64"},
			setupMocks: func(t *testing.T) {
				provArch = "aarch64"
				provIP = "192.168.1.1/24"
				provIPv6 = ""
				// Mock GetAll to return an existing VM with the same IP
				metadata.GetAll = func(cfg *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"existing-vm": {
							IP: "192.168.1.1",
						},
					}, nil
				}
			},
			expectedError: "IP address from 192.168.1.1/24 is already in use",
		},
		{
			name: "existing provisioner VM",
			args: []string{"provisioner", "create", "test-provisioner", "--ip", "192.168.1.1/24", "--arch", "aarch64"},
			setupMocks: func(t *testing.T) {
				provArch = "aarch64"
				provIP = "192.168.1.1/24"
				provIPv6 = ""
				metadata.FindProvisioner = func(c *config.Config) (string, error) {
					return "test-provisioner", nil
				}
			},
			expectedError: "a provisioner VM named 'test-provisioner' already exists",
		},
		{
			name: "tar file specified but not found",
			args: []string{"provisioner", "create", "test-provisioner", "--ip", "192.168.1.1/24", "--arch", "aarch64", "--docker-pxeboot-stack-tar", "/nonexistent/file.tar"},
			setupMocks: func(t *testing.T) {
				provArch = "aarch64"
				provIP = "192.168.1.1/24"
				provIPv6 = ""
				provPxebootStackTar = "/nonexistent/file.tar"
			},
			expectedError: "specified --docker-pxeboot-stack-tar not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags to their default values before each test
			provIP = ""
			provIPv6 = ""
			provMAC = ""
			provDiskSize = "15G"
			provArch = "aarch64"
			provPxebootStackTar = "pxeboot_stack.tar"
			provDockerImagesPath = ""
			provVMsPath = ""

			// Reset mocks to default success behavior before each test
			setupMocks(t)

			// Apply test-specific mock setup
			tt.setupMocks(t)

			output, _, err := executeCommand(rootCmd, tt.args...)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', but got none. Output: %s", tt.expectedError, output)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', but got '%v'. Output: %s", tt.expectedError, err, output)
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

func TestProvisionerCreateWithLocalTarFile(t *testing.T) {
	// Create a temporary tar file for testing
	tmpFile, err := os.CreateTemp("", "test-pxeboot-*.tar")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Reset global flags
	provIP = "192.168.1.1/24"
	provIPv6 = ""
	provArch = "aarch64"
	provPxebootStackTar = tmpFile.Name()

	setupMocks(t)

	args := []string{"provisioner", "create", "test-provisioner", "--ip", "192.168.1.1/24", "--arch", "aarch64", "--docker-pxeboot-stack-tar", tmpFile.Name()}
	output, _, err := executeCommand(rootCmd, args...)

	if err != nil {
		t.Fatalf("expected no error, but got: %v. Output: %s", err, output)
	}

	if !strings.Contains(output, "Provisioner VM 'test-provisioner' created successfully") {
		t.Errorf("expected success message, but got: %s", output)
	}

	if !strings.Contains(output, "Using local docker tarball") {
		t.Errorf("expected local tarball message, but got: %s", output)
	}
}
