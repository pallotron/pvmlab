package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"pvmlab/internal/config"
	"strconv"
	"testing"
)

// setup creates a temporary directory for pids and returns a config object.
func setup(t *testing.T) (*config.Config, func()) {
	tempDir, err := os.MkdirTemp("", "pidfile-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	cfg.SetHomeDir(tempDir)

	// Create the pids subdirectory
	if err := os.MkdirAll(filepath.Join(cfg.GetAppDir(), "pids"), 0755); err != nil {
		t.Fatalf("Failed to create pids subdirectory: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return cfg, cleanup
}

func TestRead(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm"
	pidPath := filepath.Join(cfg.GetAppDir(), "pids", vmName+".pid")

	t.Run("success", func(t *testing.T) {
		expectedPID := 12345
		if err := os.WriteFile(pidPath, []byte(strconv.Itoa(expectedPID)), 0644); err != nil {
			t.Fatalf("Failed to write pid file: %v", err)
		}
		defer os.Remove(pidPath)

		pid, err := Read(cfg, vmName)
		if err != nil {
			t.Fatalf("Read() returned an error: %v", err)
		}
		if pid != expectedPID {
			t.Errorf("Read() got = %d, want %d", pid, expectedPID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := Read(cfg, "non-existent-vm")
		if !os.IsNotExist(err) {
			t.Errorf("Read() with non-existent file should return os.IsNotExist, got: %v", err)
		}
	})

	t.Run("invalid content", func(t *testing.T) {
		if err := os.WriteFile(pidPath, []byte("not-a-pid"), 0644); err != nil {
			t.Fatalf("Failed to write pid file: %v", err)
		}
		defer os.Remove(pidPath)

		_, err := Read(cfg, vmName)
		if err == nil {
			t.Fatal("Read() with invalid content did not return an error")
		}
	})
}

func TestIsRunning(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm-running"
	pidPath := filepath.Join(cfg.GetAppDir(), "pids", vmName+".pid")

	t.Run("not running - no pid file", func(t *testing.T) {
		running, err := IsRunning(cfg, "non-existent-vm")
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

		running, err := IsRunning(cfg, vmName)
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

		running, err := IsRunning(cfg, vmName)
		if err != nil {
			t.Fatalf("IsRunning() returned an error: %v", err)
		}
		if !running {
			t.Error("IsRunning() should be true when process exists")
		}
	})
}
