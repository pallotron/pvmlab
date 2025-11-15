package distro

import (
	"context"
	"os"
	"path/filepath"
	"pvmlab/internal/config"
	"testing"
)

func TestPull_Missing7z(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty to simulate missing 7z
	os.Setenv("PATH", "")

	cfg := &config.Config{}
	tempDir := t.TempDir()
	cfg.SetHomeDir(tempDir)

	err := Pull(context.Background(), cfg, "ubuntu-24.04", "aarch64")

	if err == nil {
		t.Error("expected error for missing 7z, got nil")
	}

	if err != nil && err.Error() != "7z is not installed. Please install it to extract PXE boot assets" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPull_MissingDocker(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Create a temp dir with only a fake 7z binary
	tempBinDir := t.TempDir()
	fake7z := filepath.Join(tempBinDir, "7z")
	if err := os.WriteFile(fake7z, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatalf("failed to create fake 7z: %v", err)
	}

	// Set PATH to only include our temp dir
	os.Setenv("PATH", tempBinDir)

	cfg := &config.Config{}
	tempDir := t.TempDir()
	cfg.SetHomeDir(tempDir)

	err := Pull(context.Background(), cfg, "ubuntu-24.04", "aarch64")

	if err == nil {
		t.Error("expected error for missing docker, got nil")
	}

	if err != nil && err.Error() != "docker is not installed. Please install it to create rootfs tarballs" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPull_UnsupportedDistro(t *testing.T) {
	cfg := &config.Config{}
	tempDir := t.TempDir()
	cfg.SetHomeDir(tempDir)

	// Use a distro that doesn't exist in config.Distros
	err := Pull(context.Background(), cfg, "nonexistent-distro", "aarch64")

	if err == nil {
		t.Error("expected error for unsupported distro, got nil")
	}

	if err != nil && err.Error() != "distro configuration not found for: nonexistent-distro" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPull_UnsupportedArch(t *testing.T) {
	cfg := &config.Config{}
	tempDir := t.TempDir()
	cfg.SetHomeDir(tempDir)

	// Set up a minimal distro configuration
	config.Distros = map[string]config.Distro{
		"test-distro": {
			Name:       "test",
			Version:    "1.0",
			DistroName: "ubuntu",
			Arch: map[string]config.ArchInfo{
				"aarch64": {
					Qcow2URL:   "http://example.com/test.qcow2",
					KernelPath: "boot/vmlinuz",
					InitrdPath: "boot/initrd",
				},
			},
		},
	}

	err := Pull(context.Background(), cfg, "test-distro", "unsupported-arch")

	if err == nil {
		t.Error("expected error for unsupported arch, got nil")
	}

	expectedMsg := "architecture 'unsupported-arch' not found for distro 'test-distro'"
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%v'", expectedMsg, err)
	}
}

func TestPull_InvalidExtractor(t *testing.T) {
	// This test verifies that NewExtractor is called with the right distro name
	cfg := &config.Config{}
	tempDir := t.TempDir()
	cfg.SetHomeDir(tempDir)

	// Set up a test distro configuration with invalid extractor
	// to ensure we fail at NewExtractor
	config.Distros = map[string]config.Distro{
		"test-distro": {
			Name:       "test",
			Version:    "1.0",
			DistroName: "invalid-distro-name", // This will cause NewExtractor to fail
			Arch: map[string]config.ArchInfo{
				"aarch64": {
					Qcow2URL:   "http://example.com/test.qcow2",
					KernelPath: "boot/vmlinuz",
					InitrdPath: "boot/initrd",
				},
			},
		},
	}

	// Save original PATH and set to include common binary paths
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	// Ensure we have a valid PATH that might contain 7z and docker
	if originalPath == "" {
		os.Setenv("PATH", "/usr/bin:/usr/local/bin:/bin")
	}

	err := Pull(context.Background(), cfg, "test-distro", "aarch64")

	// Should fail at NewExtractor
	if err == nil {
		t.Error("expected error, got nil")
	}

	expectedMsg := "no extractor available for distribution: invalid-distro-name"
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%v'", expectedMsg, err)
	}
}
