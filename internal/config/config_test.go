package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAppDir(t *testing.T) {
	cfg := &Config{homeDir: "/tmp"}
	expected := "/tmp/.provisioning-vm-lab"
	if got := cfg.GetAppDir(); got != expected {
		t.Errorf("GetAppDir() = %v, want %v", got, expected)
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
}
