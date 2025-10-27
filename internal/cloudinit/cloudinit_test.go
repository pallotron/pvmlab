package cloudinit

import (
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/runner"
	"strings"
	"testing"
)

func TestExecuteTemplate(t *testing.T) {
	tmplStr := "Hello, [[ .Name ]]!"
	data := struct{ Name string }{Name: "World"}
	expected := "Hello, World!"

	result, err := executeTemplate("testTemplate", tmplStr, data)
	if err != nil {
		t.Fatalf("executeTemplate() returned an error: %v", err)
	}

	if result != expected {
		t.Errorf("executeTemplate() = %q, want %q", result, expected)
	}
}

func TestCreateISO(t *testing.T) {
	// Setup temporary directory
	appDir, err := os.MkdirTemp("", "test-app")
	if err != nil {
		t.Fatalf("Failed to create temp app dir: %v", err)
	}
	defer os.RemoveAll(appDir)

	// Setup dummy SSH key
	sshDir := filepath.Join(appDir, "ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatalf("Failed to create ssh dir: %v", err)
	}
	sshKeyPath := filepath.Join(sshDir, "vm_rsa.pub")
	dummyKey := "ssh-rsa AAAA..."
	if err := os.WriteFile(sshKeyPath, []byte(dummyKey), 0644); err != nil {
		t.Fatalf("Failed to write dummy ssh key: %v", err)
	}

	// Setup mock runner
	originalRun := runner.Run
	t.Cleanup(func() {
		runner.Run = originalRun
	})

	runner.Run = func(cmd *exec.Cmd) error {
		// Check if the command is mkisofs
		if !strings.HasSuffix(cmd.Path, "mkisofs") {
			t.Errorf("Expected command to be 'mkisofs', got '%s'", cmd.Path)
		}
		// You can add more checks here for the arguments if needed
		return nil
	}

	// Test case for "provisioner" role
	t.Run("provisioner", func(t *testing.T) {
		vmName := "test-vm"
		isoPath := filepath.Join(appDir, "provisioner.iso")
		err := CreateISO(vmName, "provisioner", appDir, isoPath, "192.168.1.1/24", "", "", "pxe-stack.tar", "ghcr.io/user/repo:tag")
		if err != nil {
			t.Fatalf("CreateISO() for provisioner returned an error: %v", err)
		}

		configDir := filepath.Join(appDir, "configs", "cloud-init", vmName)
		checkFileContent(t, filepath.Join(configDir, "meta-data"), `instance-id: iid-cloudimg-provisioner
local-hostname: provisioner
public-keys:
  - "ssh-rsa AAAA..."
pxe_boot_stack_tar: "pxe-stack.tar"
pxe_boot_stack_name: "pxe-stack"
pxe_boot_stack_image: "ghcr.io/user/repo:tag"
provisioner_ip: "192.168.1.1"
dhcp_range_start: "192.168.1.100"
dhcp_range_end: "192.168.1.200"
`)
		checkFileContent(t, filepath.Join(configDir, "user-data"), provisionerUserDataTemplate)
		checkFileContent(t, filepath.Join(configDir, "network-config"), `version: 2
ethernets:
  enp0s1:
    dhcp4: true
    dhcp6: true
  enp0s2:
    dhcp4: false
    addresses:
      - "192.168.1.1/24"
      `)
	})

	// Test case for "target" role
	t.Run("target", func(t *testing.T) {
		vmName := "test-vm-target"
		isoPath := filepath.Join(appDir, "target.iso")
		err := CreateISO(vmName, "target", appDir, isoPath, "", "", "52:54:00:12:34:56", "", "")
		if err != nil {
			t.Fatalf("CreateISO() for target returned an error: %v", err)
		}

		configDir := filepath.Join(appDir, "configs", "cloud-init", vmName)
		checkFileContent(t, filepath.Join(configDir, "meta-data"), `instance-id: iid-cloudimg-test-vm-target
local-hostname: test-vm-target
public-keys:
  - "ssh-rsa AAAA..."
`)
		checkFileContent(t, filepath.Join(configDir, "user-data"), targetUserData)
		checkFileContent(t, filepath.Join(configDir, "network-config"), `network:
  version: 2
  ethernets:
    static-interface:
      match:
        macaddress: "52:54:00:12:34:56"
      dhcp4: true
      dhcp6: true
`)
	})
}

func checkFileContent(t *testing.T, path, expectedContent string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	if string(content) != expectedContent {
		t.Errorf("File content for %s does not match expected content.\nGot:\n%s\nWant:\n%s", path, string(content), expectedContent)
	}
}
