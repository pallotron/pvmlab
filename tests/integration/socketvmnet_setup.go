package integration

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// setupSocketVMNet handles the compilation, setup, and execution of a self-contained
// socket_vmnet process for integration tests. It returns a cleanup function
// that should be deferred by the caller to ensure the process is terminated.
// On non-macOS systems, it returns a no-op cleanup function.
func setupSocketVMNet(tempHomeDir, projectRoot string) (cleanupFunc func()) {
	// socket_vmnet is macOS-only, skip on other platforms
	if os.Getenv("GOOS") == "linux" || os.Getenv("CI") == "true" {
		// Check if we're actually on Linux (not just cross-compiling)
		if _, err := os.Stat("/proc/version"); err == nil {
			log.Println("Skipping socket_vmnet setup on Linux")
			return func() {}
		}
	}

	// Use Homebrew-installed socket_vmnet for tests
	socketVMNetPath := "/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet"
	socketVMNetClientPath := "/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet_client"

	// Start local socket_vmnet process
	socketPath := filepath.Join(tempHomeDir, "vmlab.socket_vmnet")
	os.Setenv("PVMLAB_SOCKET_VMNET_PATH", socketPath)
	os.Setenv("PVMLAB_SOCKET_VMNET_CLIENT", socketVMNetClientPath)
	log.Printf("Using local socket_vmnet path: %s", socketPath)

	networkID := uuid.New().String()
	socketVMNetCmd := exec.Command("sudo", socketVMNetPath,
		"--vmnet-mode=host",
		"--vmnet-network-identifier="+networkID,
		socketPath,
	)
	var socketVMLog bytes.Buffer
	socketVMNetCmd.Stdout = &socketVMLog
	socketVMNetCmd.Stderr = &socketVMLog

	if err := socketVMNetCmd.Start(); err != nil {
		log.Fatalf("failed to start local socket_vmnet: %v", err)
	}
	log.Printf("Started local socket_vmnet with PID: %d", socketVMNetCmd.Process.Pid)

	// Wait for the socket service to be ready by pinging it with the client
	readinessTimeout := 10 * time.Second
	readinessPollInterval := 200 * time.Millisecond
	startTime := time.Now()
	for {
		pingCmd := exec.Command(socketVMNetClientPath, socketPath, "echo", "hello")
		if err := pingCmd.Run(); err == nil {
			log.Printf("socket_vmnet service is ready after %v", time.Since(startTime))
			break
		}
		if time.Since(startTime) > readinessTimeout {
			log.Printf("---\n%s", socketVMLog.String())
			log.Fatalf("timed out waiting for socket_vmnet service to become ready at %s", socketPath)
		}
		time.Sleep(readinessPollInterval)
	}

	cleanupFunc = func() {
		log.Println("Stopping local socket_vmnet process...")
		if err := socketVMNetCmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("failed to send SIGTERM to socket_vmnet: %v", err)
			_ = socketVMNetCmd.Process.Kill()
		}
		_ = socketVMNetCmd.Wait()
		log.Println("Local socket_vmnet process stopped.")
	}

	return cleanupFunc
}
