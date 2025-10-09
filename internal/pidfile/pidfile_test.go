package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"strconv"
	"testing"
)

// setup creates a temporary directory for pids and mocks config.GetAppDirFunc.
func setup(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "pidfile-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Mock GetAppDirFunc to use the temp directory
	originalGetAppDir := config.GetAppDirFunc
	config.GetAppDirFunc = func() (string, error) {
		return tempDir, nil
	}

	// Create the pids subdirectory
	if err := os.MkdirAll(filepath.Join(tempDir, "pids"), 0755); err != nil {
		t.Fatalf("Failed to create pids subdirectory: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
		config.GetAppDirFunc = originalGetAppDir
	}

	return tempDir, cleanup
}

func TestRead(t *testing.T) {
	appDir, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm"
	pidPath := filepath.Join(appDir, "pids", vmName+".pid")

	t.Run("success", func(t *testing.T) {
		expectedPID := 12345
		if err := os.WriteFile(pidPath, []byte(strconv.Itoa(expectedPID)), 0644); err != nil {
			t.Fatalf("Failed to write pid file: %v", err)
		}
		defer os.Remove(pidPath)

		pid, err := Read(vmName)
		if err != nil {
			t.Fatalf("Read() returned an error: %v", err)
		}
		if pid != expectedPID {
			t.Errorf("Read() got = %d, want %d", pid, expectedPID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := Read("non-existent-vm")
		if !os.IsNotExist(err) {
			t.Errorf("Read() with non-existent file should return os.IsNotExist, got: %v", err)
		}
	})

	t.Run("invalid content", func(t *testing.T) {
		if err := os.WriteFile(pidPath, []byte("not-a-pid"), 0644); err != nil {
			t.Fatalf("Failed to write pid file: %v", err)
		}
		defer os.Remove(pidPath)

		_, err := Read(vmName)
		if err == nil {
			t.Fatal("Read() with invalid content did not return an error")
		}
	})
}

func TestIsRunning(t *testing.T) {
	appDir, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm-running"
	pidPath := filepath.Join(appDir, "pids", vmName+".pid")

	t.Run("not running - no pid file", func(t *testing.T) {
		running, err := IsRunning("non-existent-vm")
		if err != nil {
			t.Fatalf("IsRunning() returned an error: %v", err)
		}
		if running {
			t.Error("IsRunning() should be false when pid file does not exist")
		}
	})

	t.Run("not running - process does not exist", func(t *testing.T) {
		// Use a PID that is highly unlikely to exist
		if err := os.WriteFile(pidPath, []byte("999999"), 0644); err != nil {
			t.Fatalf("Failed to write pid file: %v", err)
		}
		defer os.Remove(pidPath)

		running, err := IsRunning(vmName)
		if err != nil {
			t.Fatalf("IsRunning() returned an error: %v", err)
		}
		if running {
			t.Error("IsRunning() should be false when process does not exist")
		}
	})

	t.Run("running - process exists", func(t *testing.T) {
		// Use the current process's PID for a guaranteed running process
		myPID := os.Getpid()
		if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", myPID)), 0644); err != nil {
			t.Fatalf("Failed to write pid file: %v", err)
		}
		defer os.Remove(pidPath)

		running, err := IsRunning(vmName)
		if err != nil {
			t.Fatalf("IsRunning() returned an error: %v", err)
		}
		if !running {
			t.Error("IsRunning() should be true when process exists")
		}
	})
}
