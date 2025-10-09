package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAppDir(t *testing.T) {
	// Get the original userHomeDir function
	originalUserHomeDir := userHomeDir
	// Defer the restoration of the original function
	t.Cleanup(func() {
		userHomeDir = originalUserHomeDir
	})

	// Create a temporary directory to act as the user's home
	tempHomeDir, err := os.MkdirTemp("", "test-home")
	if err != nil {
		t.Fatalf("failed to create temporary home directory: %v", err)
	}
	defer os.RemoveAll(tempHomeDir)

	// Set the userHomeDir function to return the temporary directory
	userHomeDir = func() (string, error) {
		return tempHomeDir, nil
	}

	// Call the function under test
	appDir, err := GetAppDir()
	if err != nil {
		t.Fatalf("GetAppDir() returned an error: %v", err)
	}

	// Define the expected application directory path
	expectedAppDir := filepath.Join(tempHomeDir, "."+AppName)

	// Check if the returned path matches the expected path
	if appDir != expectedAppDir {
		t.Errorf("GetAppDir() = %v, want %v", appDir, expectedAppDir)
	}
}

func TestGetProjectRootDir(t *testing.T) {
	// Create a temporary directory structure for the test
	rootDir, err := os.MkdirTemp("", "test-project")
	if err != nil {
		t.Fatalf("failed to create temporary project directory: %v", err)
	}
	defer os.RemoveAll(rootDir)

	// Create a go.mod file in the root directory
	if err := os.WriteFile(filepath.Join(rootDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatalf("failed to create go.mod file: %v", err)
	}

	// Create a subdirectory and change the working directory to it
	subDir := filepath.Join(rootDir, "internal", "test")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(originalWd)
	})

	// Call the function under test
	foundRootDir, err := GetProjectRootDir()
	if err != nil {
		t.Fatalf("GetProjectRootDir() returned an error: %v", err)
	}

	// Check if the returned path matches the expected path
	evalFoundRootDir, err := filepath.EvalSymlinks(foundRootDir)
	if err != nil {
		t.Fatalf("failed to evaluate symlinks for foundRootDir: %v", err)
	}
	evalRootDir, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		t.Fatalf("failed to evaluate symlinks for rootDir: %v", err)
	}
	if evalFoundRootDir != evalRootDir {
		t.Errorf("GetProjectRootDir() = %v, want %v", evalFoundRootDir, evalRootDir)
	}
}