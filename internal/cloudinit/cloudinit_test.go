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
		isoPath := filepath.Join(appDir, "provisioner.iso")
		err := CreateISO("test-vm", "provisioner", appDir, isoPath, "192.168.1.1/24", "", "", "pxe-stack.tar", "ghcr.io/user/repo:tag")
		if err != nil {
			t.Fatalf("CreateISO() for provisioner returned an error: %v", err)
		}
		// Check if config files were created
		configDir := filepath.Join(appDir, "configs", "cloud-init", "test-vm")
		if _, err := os.Stat(filepath.Join(configDir, "meta-data")); os.IsNotExist(err) {
			t.Error("meta-data file was not created for provisioner")
		}
	})

	// Test case for "target" role
	t.Run("target", func(t *testing.T) {
		isoPath := filepath.Join(appDir, "target.iso")
		err := CreateISO("test-vm-target", "target", appDir, isoPath, "", "", "52:54:00:12:34:56", "", "")
		if err != nil {
			t.Fatalf("CreateISO() for target returned an error: %v", err)
		}
		// Check if config files were created
		configDir := filepath.Join(appDir, "configs", "cloud-init", "test-vm-target")
		if _, err := os.Stat(filepath.Join(configDir, "user-data")); os.IsNotExist(err) {
			t.Error("user-data file was not created for target")
		}
	})
}
