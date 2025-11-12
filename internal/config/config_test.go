package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAppDir(t *testing.T) {
	cfg := &Config{homeDir: "/tmp"}
	expected := "/tmp/.pvmlab"
	if got := cfg.GetAppDir(); got != expected {
		t.Errorf("GetAppDir() = %v, want %v", got, expected)
	}
}

func TestSetHomeDir(t *testing.T) {
	cfg := &Config{}
	cfg.SetHomeDir("/tmp")
	if cfg.homeDir != "/tmp" {
		t.Errorf("SetHomeDir() did not set homeDir correctly, got %v, want %v", cfg.homeDir, "/tmp")
	}
}

func TestGetProjectRootDir(t *testing.T) {
	// Create a temporary directory structure to simulate a project
	tmpDir, err := os.MkdirTemp("", "pvmlab-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a go.mod file in the temporary directory
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module pvmlab"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{}
	got, err := cfg.GetProjectRootDir(tmpDir)
	if err != nil {
		t.Errorf("GetProjectRootDir() error = %v, wantErr %v", err, false)
	}
	if got != tmpDir {
		t.Errorf("GetProjectRootDir() = %v, want %v", got, tmpDir)
	}

	// Test case where go.mod is not found
	tmpDirNoMod, err := os.MkdirTemp("", "pvmlab-test-no-mod")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirNoMod)

	_, err = cfg.GetProjectRootDir(tmpDirNoMod)
	if err == nil {
		t.Errorf("GetProjectRootDir() error = %v, wantErr %v", err, true)
	}
}

func TestGetProvisionerImageURL(t *testing.T) {
	Version = "devel"
	url, name := GetProvisionerImageURL("aarch64")
	if url != "https://github.com/pallotron/pvmlab/releases/latest/download/provisioner-custom.arm64.qcow2" {
		t.Errorf("GetProvisionerImageURL() url = %v, want %v", url, "https://github.com/pallotron/pvmlab/releases/latest/download/provisioner-custom.arm64.qcow2")
	}
	if name != "provisioner-custom.arm64.qcow2" {
		t.Errorf("GetProvisionerImageURL() name = %v, want %v", name, "provisioner-custom.arm64.qcow2")
	}

	url, name = GetProvisionerImageURL("amd64")
	if url != "https://github.com/pallotron/pvmlab/releases/latest/download/provisioner-custom.amd64.qcow2" {
		t.Errorf("GetProvisionerImageURL() url = %v, want %v", url, "https://github.com/pallotron/pvmlab/releases/latest/download/provisioner-custom.amd64.qcow2")
	}
	if name != "provisioner-custom.amd64.qcow2" {
		t.Errorf("GetProvisionerImageURL() name = %v, want %v", name, "provisioner-custom.amd64.qcow2")
	}

	Version = "v0.1.0"
	url, name = GetProvisionerImageURL("aarch64")
	if url != "https://github.com/pallotron/pvmlab/releases/download/v0.1.0/provisioner-custom.arm64.qcow2" {
		t.Errorf("GetProvisionerImageURL() url = %v, want %v", url, "https://github.com/pallotron/pvmlab/releases/download/v0.1.0/provisioner-custom.arm64.qcow2")
	}
	if name != "provisioner-custom.arm64.qcow2" {
		t.Errorf("GetProvisionerImageURL() name = %v, want %v", name, "provisioner-custom.arm64.qcow2")
	}
}

func TestGetPxeBootStackImageURL(t *testing.T) {
	Version = "devel"
	url := GetPxeBootStackImageURL()
	if url != "ghcr.io/pallotron/pvmlab/pxeboot_stack:latest" {
		t.Errorf("GetPxeBootStackImageURL() = %v, want %v", url, "ghcr.io/pallotron/pvmlab/pxeboot_stack:latest")
	}

	Version = "v0.1.0"
	url = GetPxeBootStackImageURL()
	if url != "ghcr.io/pallotron/pvmlab/pxeboot_stack:v0.1.0" {
		t.Errorf("GetPxeBootStackImageURL() = %v, want %v", url, "ghcr.io/pallotron/pvmlab/pxeboot_stack:v0.1.0")
	}
}

func TestGetPxeBootStackImageName(t *testing.T) {
	Version = "devel"
	name, tag := GetPxeBootStackImageName()
	if name != "pxeboot_stack" {
		t.Errorf("GetPxeBootStackImageName() = %v, want %v", name, "pxeboot_stack")
	}
	if tag != "latest" {
		t.Errorf("GetPxeBootStackImageName() = %v, want %v", tag, "latest")
	}

	Version = "v0.1.0"
	name, tag = GetPxeBootStackImageName()
	if name != "pxeboot_stack" {
		t.Errorf("GetPxeBootStackImageName() = %v, want %v", name, "pxeboot_stack")
	}
	if tag != Version {
		t.Errorf("GetPxeBootStackImageName() = %v, want %v", tag, Version)
	}
}

func TestNew(t *testing.T) {
	// Test with PVMLAB_HOME set
	os.Setenv("PVMLAB_HOME", "/tmp/pvmlab-home")
	cfg, err := New()
	if err != nil {
		t.Fatalf("New() with PVMLAB_HOME failed: %v", err)
	}
	if cfg.homeDir != "/tmp/pvmlab-home" {
		t.Errorf("New() with PVMLAB_HOME: got %s, want /tmp/pvmlab-home", cfg.homeDir)
	}
	os.Unsetenv("PVMLAB_HOME")

	// Test without PVMLAB_HOME set
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}
	cfg, err = New()
	if err != nil {
		t.Fatalf("New() without PVMLAB_HOME failed: %v", err)
	}
	if cfg.homeDir != userHome {
		t.Errorf("New() without PVMLAB_HOME: got %s, want %s", cfg.homeDir, userHome)
	}
}
