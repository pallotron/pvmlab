package logwatcher

import (
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"testing"
	"time"
)

// setup creates a temporary directory for logs and returns a config object.
func setup(t *testing.T) (*config.Config, func()) {
	tempDir, err := os.MkdirTemp("", "logwatcher-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	cfg.SetHomeDir(tempDir)

	// Create the logs subdirectory
	if err := os.MkdirAll(filepath.Join(cfg.GetAppDir(), "logs"), 0755); err != nil {
		t.Fatalf("Failed to create logs subdirectory: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return cfg, cleanup
}

func TestWaitForMessage_Success(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm-success"
	message := "cloud-init finished"
	logPath := filepath.Join(cfg.GetAppDir(), "logs", vmName+".log")
	timeout := 2 * time.Second

	// Create the log file
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- WaitForMessage(cfg, vmName, message, timeout)
	}()

	// Give the watcher a moment to start
	time.Sleep(100 * time.Millisecond)

	// Write the message to the log file
	_, err = logFile.WriteString(message + "\n")
	if err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	logFile.Close()

	// Check the result
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("WaitForMessage() returned an error: %v", err)
		}
	case <-time.After(timeout + 500*time.Millisecond):
		t.Fatal("WaitForMessage() did not return within the expected time")
	}
}

func TestWaitForMessage_Timeout(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm-timeout"
	message := "this will not appear"
	timeout := 100 * time.Millisecond // Use a short timeout for the test

	err := WaitForMessage(cfg, vmName, message, timeout)

	if err == nil {
		t.Fatal("WaitForMessage() did not return an error on timeout")
	}

	if !os.IsTimeout(err) && err.Error() != "timed out waiting for message in log file" {
		t.Errorf("WaitForMessage() returned wrong error type on timeout: %v", err)
	}
}
