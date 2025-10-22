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
func setupSocketVMNet(tempHomeDir, projectRoot string) (cleanupFunc func()) {
	socketVMNetDir := filepath.Join(projectRoot, "socket_vmnet")
	tempBinDir := filepath.Join(tempHomeDir, "bin")
	if err := os.MkdirAll(tempBinDir, 0755); err != nil {
		log.Fatalf("failed to create temp bin dir: %v", err)
	}

	// Build socket_vmnet and its client
	makeCmd := exec.Command("make", "all")
	makeCmd.Dir = socketVMNetDir
	if output, err := makeCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to build socket_vmnet: %v\n%s", err, string(output))
	}

	// Copy binaries to temp dir
	destSocketVMNetPath := filepath.Join(tempBinDir, "socket_vmnet")
	if err := copyAndMakeExecutable(filepath.Join(socketVMNetDir, "socket_vmnet"), destSocketVMNetPath); err != nil {
		log.Fatalf("failed to copy socket_vmnet binary: %v", err)
	}
	destSocketVMNetClientPath := filepath.Join(tempBinDir, "socket_vmnet_client")
	if err := copyAndMakeExecutable(filepath.Join(socketVMNetDir, "socket_vmnet_client"), destSocketVMNetClientPath); err != nil {
		log.Fatalf("failed to copy socket_vmnet_client binary: %v", err)
	}

	// Start local socket_vmnet process
	socketPath := filepath.Join(tempHomeDir, "vmlab.socket_vmnet")
	os.Setenv("PVMLAB_SOCKET_VMNET_PATH", socketPath)
	os.Setenv("PVMLAB_SOCKET_VMNET_CLIENT", destSocketVMNetClientPath)
	log.Printf("Using local socket_vmnet path: %s", socketPath)

	networkID := uuid.New().String()
	socketVMNetCmd := exec.Command("sudo", destSocketVMNetPath,
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
		pingCmd := exec.Command(destSocketVMNetClientPath, socketPath, "echo", "hello")
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
