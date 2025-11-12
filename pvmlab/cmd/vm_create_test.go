package cmd

import (
	"context"
	"errors"
	"os"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"strings"
	"testing"
)

func TestVMCreateCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
		expectedOut   string
	}{
		{
			name:          "missing vm name",
			args:          []string{"vm", "create"},
			setupMocks:    func() {},
			expectedError: "accepts 1 arg(s), received 0",
		},

		{
			name: "existing vm",
			args: []string{"vm", "create", "test-vm"},
			setupMocks: func() {
				metadata.FindVM = func(c *config.Config, name string) (string, error) {
					return "test-vm", nil
				}
			},
			expectedError: "a VM named 'test-vm' already exists",
		},
		{
			name: "disk creation fails",
			args: []string{"vm", "create", "test-vm", "--distro", "ubuntu-24.04"},
			setupMocks: func() {
				createDisk = func(ctx context.Context, imagePath, vmDiskPath, diskSize string) error {
					return errors.New("qemu-img failed")
				}
			},
			expectedError: "qemu-img failed",
		},
		{
			name: "iso create failure",
			args: []string{"vm", "create", "test-vm", "--distro", "ubuntu-24.04"},
			setupMocks: func() {
				cloudinit.CreateISO = func(ctx context.Context, vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image string) error {
					return errors.New("iso creation failed")
				}
			},
			expectedError: "iso creation failed",
		},
		{
			name: "metadata save failure (warning)",
			args: []string{"vm", "create", "test-vm", "--distro", "ubuntu-24.04"},
			setupMocks: func() {
				metadata.Save = func(c *config.Config, _, _, _, _, _, _, _, _, _, _, _, sshKey, _, _ string, i int, _ bool, _ string) error {
					return errors.New("metadata save failed")
				}
			},
			expectedError: "", // Should not error out, just warn
			expectedOut:   "Warning: failed to save VM metadata",
		},
		{
			name: "create target success",
			args: []string{"vm", "create", "my-target", "--distro", "ubuntu-24.04"},
			setupMocks: func() {
				// All mocks default to success
			},
			expectedError: "",
			expectedOut:   "created successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags to their default values before each test
			ip = ""
			diskSize = "10G"

			// Reset mocks to default success behavior before each test
			setupMocks(t)
			config.GetDistro = func(distroName, arch string) (*config.ArchInfo, error) {
				return &config.ArchInfo{
					Qcow2URL: "http://example.com/image.qcow2",
				}, nil
			}
			// Apply test-specific mock setup
			tt.setupMocks()

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

			if tt.expectedOut != "" {
				if !strings.Contains(output, tt.expectedOut) {
					t.Errorf("expected output to contain '%s', but got '%s'", tt.expectedOut, output)
				}
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name     string
		sizeStr  string
		expected int64
		wantErr  bool
	}{
		{
			name:     "bytes only",
			sizeStr:  "2048",
			expected: 2048,
			wantErr:  false,
		},
		{
			name:     "kilobytes with K",
			sizeStr:  "10K",
			expected: 10 * 1024,
			wantErr:  false,
		},
		{
			name:     "kilobytes with KB",
			sizeStr:  "10KB",
			expected: 10 * 1024,
			wantErr:  false,
		},
		{
			name:     "megabytes with M",
			sizeStr:  "512M",
			expected: 512 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "megabytes with MB",
			sizeStr:  "512MB",
			expected: 512 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "gigabytes with G",
			sizeStr:  "10G",
			expected: 10 * 1024 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "gigabytes with GB",
			sizeStr:  "10GB",
			expected: 10 * 1024 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "terabytes with T",
			sizeStr:  "2T",
			expected: 2 * 1024 * 1024 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "terabytes with TB",
			sizeStr:  "2TB",
			expected: 2 * 1024 * 1024 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "bytes with B suffix",
			sizeStr:  "2048B",
			expected: 2048,
			wantErr:  false,
		},
		{
			name:     "lowercase units",
			sizeStr:  "10g",
			expected: 10 * 1024 * 1024 * 1024,
			wantErr:  false,
		},
		{
			name:    "invalid format - no number",
			sizeStr: "abc",
			wantErr: true,
		},
		{
			name:    "invalid format - unknown unit",
			sizeStr: "10X",
			wantErr: true,
		},
		{
			name:    "empty string",
			sizeStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSize(tt.sizeStr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSize(%q) expected error, but got none", tt.sizeStr)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSize(%q) unexpected error: %v", tt.sizeStr, err)
				return
			}

			if result != tt.expected {
				t.Errorf("parseSize(%q) = %d, want %d", tt.sizeStr, result, tt.expected)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "empty IP (valid)",
			ip:      "",
			wantErr: false,
		},
		{
			name:    "valid IPv4 CIDR",
			ip:      "192.168.1.10/24",
			wantErr: false,
		},
		{
			name:    "valid IPv4 CIDR with /32",
			ip:      "10.0.0.1/32",
			wantErr: false,
		},
		{
			name:    "invalid - no CIDR notation",
			ip:      "192.168.1.10",
			wantErr: true,
		},
		{
			name:    "invalid - malformed IP",
			ip:      "999.999.999.999/24",
			wantErr: true,
		},
		{
			name:    "invalid - malformed CIDR",
			ip:      "192.168.1.10/abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIP(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIP(%q) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

func TestValidateIPv6(t *testing.T) {
	tests := []struct {
		name    string
		ipv6    string
		wantErr bool
	}{
		{
			name:    "empty IPv6 (valid)",
			ipv6:    "",
			wantErr: false,
		},
		{
			name:    "valid IPv6 CIDR",
			ipv6:    "fd00:cafe:babe::1/64",
			wantErr: false,
		},
		{
			name:    "valid IPv6 CIDR with /128",
			ipv6:    "2001:db8::1/128",
			wantErr: false,
		},
		{
			name:    "invalid - no CIDR notation",
			ipv6:    "fd00:cafe:babe::1",
			wantErr: true,
		},
		{
			name:    "invalid - malformed IPv6",
			ipv6:    "gggg::1/64",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPv6(tt.ipv6)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIPv6(%q) error = %v, wantErr %v", tt.ipv6, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMac(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
	}{
		{
			name:    "empty MAC (valid)",
			mac:     "",
			wantErr: false,
		},
		{
			name:    "valid MAC with colons",
			mac:     "00:11:22:33:44:55",
			wantErr: false,
		},
		{
			name:    "valid MAC with hyphens",
			mac:     "00-11-22-33-44-55",
			wantErr: false,
		},
		{
			name:    "valid MAC uppercase",
			mac:     "AA:BB:CC:DD:EE:FF",
			wantErr: false,
		},
		{
			name:    "valid MAC lowercase",
			mac:     "aa:bb:cc:dd:ee:ff",
			wantErr: false,
		},
		{
			name:    "invalid - no separators",
			mac:     "001122334455",
			wantErr: true,
		},
		{
			name:    "invalid - wrong length",
			mac:     "00:11:22:33:44",
			wantErr: true,
		},
		{
			name:    "invalid - invalid characters",
			mac:     "00:11:22:33:44:GG",
			wantErr: true,
		},
		{
			name:    "mixed separators - actually valid per regex",
			mac:     "00:11-22:33:44:55",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMac(tt.mac)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMac(%q) error = %v, wantErr %v", tt.mac, err, tt.wantErr)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		defaultPath string
		wantErr     bool
	}{
		{
			name:        "empty path - use default",
			path:        "",
			defaultPath: "/default/path",
			wantErr:     false,
		},
		{
			name:        "relative path",
			path:        "relative/path",
			defaultPath: "/default/path",
			wantErr:     false,
		},
		{
			name:        "absolute path",
			path:        "/absolute/path",
			defaultPath: "/default/path",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolvePath(tt.path, tt.defaultPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolvePath(%q, %q) error = %v, wantErr %v", tt.path, tt.defaultPath, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.path == "" && result != tt.defaultPath {
					t.Errorf("resolvePath(%q, %q) = %q, want %q", tt.path, tt.defaultPath, result, tt.defaultPath)
				}
			}
		})
	}
}

func TestGetImageVirtualSize(t *testing.T) {
	tests := []struct {
		name      string
		imagePath string
		wantErr   bool
	}{
		{
			name:      "non-existent image",
			imagePath: "/non/existent/path/image.qcow2",
			wantErr:   true,
		},
		{
			name:      "empty path",
			imagePath: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getImageVirtualSize(tt.imagePath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getImageVirtualSize(%q) expected error, but got none", tt.imagePath)
				}
			} else {
				if err != nil {
					t.Errorf("getImageVirtualSize(%q) unexpected error: %v", tt.imagePath, err)
				}
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary source file
	tmpDir := t.TempDir()
	srcFile := tmpDir + "/source.txt"
	dstFile := tmpDir + "/dest.txt"

	// Write some content to the source file
	content := []byte("test content for copy")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		src     string
		dst     string
		wantErr bool
	}{
		{
			name:    "successful copy",
			src:     srcFile,
			dst:     dstFile,
			wantErr: false,
		},
		{
			name:    "non-existent source",
			src:     "/non/existent/source.txt",
			dst:     dstFile,
			wantErr: true,
		},
		{
			name:    "invalid destination path",
			src:     srcFile,
			dst:     "/non/existent/dir/dest.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := copyFile(tt.src, tt.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyFile(%q, %q) error = %v, wantErr %v", tt.src, tt.dst, err, tt.wantErr)
			}
		})
	}
}

func TestCheckExistingVMs(t *testing.T) {
	tests := []struct {
		name    string
		vmName  string
		role    string
		setup   func()
		wantErr bool
		errMsg  string
	}{
		{
			name:   "target VM - no existing VM",
			vmName: "new-vm",
			role:   "target",
			setup: func() {
				metadata.FindVM = func(c *config.Config, name string) (string, error) {
					return "", nil
				}
			},
			wantErr: false,
		},
		{
			name:   "target VM - existing VM found",
			vmName: "existing-vm",
			role:   "target",
			setup: func() {
				metadata.FindVM = func(c *config.Config, name string) (string, error) {
					return "existing-vm", nil
				}
			},
			wantErr: true,
			errMsg:  "already exists",
		},
		{
			name:   "target VM - error checking",
			vmName: "test-vm",
			role:   "target",
			setup: func() {
				metadata.FindVM = func(c *config.Config, name string) (string, error) {
					return "", errors.New("database error")
				}
			},
			wantErr: true,
			errMsg:  "error checking for existing VM",
		},
		{
			name:   "provisioner - no existing provisioner",
			vmName: "new-provisioner",
			role:   "provisioner",
			setup: func() {
				metadata.FindProvisioner = func(c *config.Config) (string, error) {
					return "", nil
				}
			},
			wantErr: false,
		},
		{
			name:   "provisioner - existing provisioner found",
			vmName: "new-provisioner",
			role:   "provisioner",
			setup: func() {
				metadata.FindProvisioner = func(c *config.Config) (string, error) {
					return "existing-provisioner", nil
				}
			},
			wantErr: true,
			errMsg:  "already exists",
		},
		{
			name:   "provisioner - error checking",
			vmName: "test-provisioner",
			role:   "provisioner",
			setup: func() {
				metadata.FindProvisioner = func(c *config.Config) (string, error) {
					return "", errors.New("database error")
				}
			},
			wantErr: true,
			errMsg:  "error checking for existing provisioner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.setup()

			cfg, _ := config.New()
			err := checkExistingVMs(cfg, tt.vmName, tt.role)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkExistingVMs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !stringContains(err.Error(), tt.errMsg) {
					t.Errorf("checkExistingVMs() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestGetMac(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
	}{
		{
			name:    "valid MAC provided",
			mac:     "00:11:22:33:44:55",
			wantErr: false,
		},
		{
			name:    "empty MAC - should generate random",
			mac:     "",
			wantErr: false,
		},
		{
			name:    "invalid MAC",
			mac:     "invalid-mac",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getMac(tt.mac)

			if (err != nil) != tt.wantErr {
				t.Errorf("getMac(%q) error = %v, wantErr %v", tt.mac, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == "" {
					t.Errorf("getMac(%q) returned empty MAC", tt.mac)
				}
				// Validate the returned MAC is valid format
				if err := validateMac(result); err != nil {
					t.Errorf("getMac(%q) returned invalid MAC %q: %v", tt.mac, result, err)
				}
			}
		})
	}
}