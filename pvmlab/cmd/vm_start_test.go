package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/pidfile"
	"strings"
	"testing"
)

func TestVMStartCommand(t *testing.T) {
	// This test focuses on the pre-flight checks of the 'vm start' command.
	// The more complex logic of building QEMU arguments is tested separately
	// in TestBuildQEMUArgs.
	tests := []struct {
		name          string
		args          []string
		setupMocks    func()
		expectedError string
	}{
		{
			name:          "no vm name",
			args:          []string{"vm", "start"},
			setupMocks:    func() {},
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name:          "wait and interactive flags are mutually exclusive",
			args:          []string{"vm", "start", "test-vm", "--wait", "--interactive"},
			setupMocks:    func() {},
			expectedError: "the --wait and --interactive flags are mutually exclusive",
		},
		{
			name: "vm is already running",
			args: []string{"vm", "start", "test-vm"},
			setupMocks: func() {
				pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
					return true, nil
				}
			},
			expectedError: "VM 'test-vm' is already running",
		},
		{
			name: "metadata not found",
			args: []string{"vm", "start", "test-vm"},
			setupMocks: func() {
				metadata.Load = func(c *config.Config, name string) (*metadata.Metadata, error) {
					return nil, errors.New("not found")
				}
			},
			expectedError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)
			tt.setupMocks()

			// Reset flags
			wait = false
			interactive = false
			bootOverride = ""

			_, _, err := executeCommand(rootCmd, tt.args...)

			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', but got '%v'", tt.expectedError, err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}
		})
	}
}

func TestBuildQEMUArgs(t *testing.T) {
	// This test focuses specifically on the logic of building the QEMU command line.
	tests := []struct {
		name           string
		opts           *vmStartOptions
		expectedArgs   []string
		unexpectedArgs []string
		expectedError  string
	}{
		{
			name: "basic target vm",
			opts: &vmStartOptions{
				vmName: "test-target",
				meta:   &metadata.Metadata{Role: "target", Arch: "aarch64", MAC: "aa:bb:cc"},
			},
			expectedArgs: []string{
				"qemu-system-aarch64",
				"-M", "virt,gic-version=3",
				"-drive", "file=/vms/test-target.qcow2,format=qcow2,if=virtio",
				"-drive", "file=/configs/cloud-init/test-target.iso,format=raw,if=virtio",
				"-device", "virtio-net-pci,netdev=net0,mac=aa:bb:cc",
				"-netdev", "socket,id=net0,fd=3",
				"-cpu", "host",
				"-accel", "hvf",
			},
		},
		{
			name: "provisioner vm",
			opts: &vmStartOptions{
				vmName: "test-prov",
				meta:   &metadata.Metadata{Role: "provisioner", Arch: "aarch64", MAC: "dd:ee:ff"},
			},
			expectedArgs: []string{
				"qemu-system-aarch64",
				"-m", "4096",
				"-device", "virtio-net-pci,netdev=net0",
				"-netdev", "user,id=net0,hostfwd=tcp::12345-:22",
				"-device", "virtio-net-pci,netdev=net1,mac=dd:ee:ff",
				"-netdev", "socket,id=net1,fd=3",
				"-virtfs", "local,path=/docker_images,mount_tag=host_share_docker_images",
			},
		},
		{
			name: "pxeboot target vm",
			opts: &vmStartOptions{
				vmName: "pxe-target",
				meta:   &metadata.Metadata{Role: "target", Arch: "aarch64", MAC: "aa:bb:cc", PxeBoot: true},
			},
			expectedArgs: []string{
				"-boot", "n",
				"-device", "e1000,netdev=net0,mac=aa:bb:cc", // e1000 for PXE
			},
			unexpectedArgs: []string{
				"cloud-init", // No ISO for PXE boot
			},
		},
		{
			name: "x86_64 vm",
			opts: &vmStartOptions{
				vmName: "x86-vm",
				meta:   &metadata.Metadata{Role: "target", Arch: "x86_64", MAC: "aa:bb:cc"},
			},
			expectedArgs: []string{
				"qemu-system-x86_64",
				"-M", "q35",
				"-cpu", "max",
			},
			unexpectedArgs: []string{
				"gic-version=3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMocks(t)

			// Create a temporary directory for app files
			tempDir := t.TempDir()
			tt.opts.appDir = tempDir
			if err := os.MkdirAll(filepath.Join(tempDir, "vms"), 0755); err != nil {
				t.Fatalf("failed to create temp vms dir: %v", err)
			}

			// Create a dummy UEFI vars file to avoid dependency on host system
			dummyUefiVarsPath := filepath.Join(tempDir, "dummy-uefi-vars.fd")
			if err := os.WriteFile(dummyUefiVarsPath, []byte(""), 0644); err != nil {
				t.Fatalf("failed to create dummy uefi vars file: %v", err)
			}
			originalUefiVarsPath := uefiVarsTemplatePath
			uefiVarsTemplatePath = dummyUefiVarsPath
			defer func() { uefiVarsTemplatePath = originalUefiVarsPath }()

			// The buildQEMUArgs function relies on info gathered in gatherVMInfo.
			// We can simulate that the necessary files exist by not returning an error
			// and ensuring the directories for file creation exist.

			args, err := buildQEMUArgs(tt.opts)

			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', but got '%v'", tt.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			argString := strings.Join(args, " ")
			for _, expected := range tt.expectedArgs {
				// In the test, the temp dir path will be long, so we just check for the relative part.
				// We replace the tempDir part of the path in the expected string to match the output.
				expected = strings.Replace(expected, "/vms", filepath.Join(tempDir, "vms"), 1)
				expected = strings.Replace(expected, "/configs", filepath.Join(tempDir, "configs"), 1)
				expected = strings.Replace(expected, "/docker_images", filepath.Join(tempDir, "docker_images"), 1)

				if !strings.Contains(argString, expected) {
					t.Errorf("expected QEMU args to contain '%s', but they did not. Got: %s", expected, argString)
				}
			}
			for _, unexpected := range tt.unexpectedArgs {
				if strings.Contains(argString, unexpected) {
					t.Errorf("expected QEMU args to NOT contain '%s', but they did. Got: %s", unexpected, argString)
				}
			}
		})
	}
}
