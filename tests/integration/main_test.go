package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	// pathToCLI will be the path to the pvmlab binary.
	// In CI, it's expected to be in the PATH.
	// For local runs, it's compiled.
	pathToCLI = "pvmlab"
)

// TestMain handles setup and teardown for the integration tests.
func TestMain(m *testing.M) {
	// Create a temporary directory in a known short path to avoid issues
	// with UNIX socket path length limits in QEMU.
	tempHomeDir, err := os.MkdirTemp("/tmp", "pvmlab_test_*")
	if err != nil {
		log.Fatalf("failed to create temp dir for integration test: %v", err)
	}
	// Defer cleanup of the temporary directory
	defer os.RemoveAll(tempHomeDir)

	// Set an environment variable to make the CLI use this temp directory.
	os.Setenv("PVMLAB_HOME", tempHomeDir)
	// Set a longer timeout for cloud-init in tests.
	os.Setenv("PVMLAB_WAIT_TIMEOUT", "900")
	// Enable debug output from the CLI.
	os.Setenv("PVMLAB_DEBUG", "true")

	// Add debug logging to verify the environment
	log.Printf("Using temporary home for tests: %s", tempHomeDir)
	log.Printf("PVMLAB_HOME set to: %s", os.Getenv("PVMLAB_HOME"))

	// For local runs, we always build a temporary binary to ensure our
	// PVMLAB_HOME logic is included. In a real CI environment, the
	// binary would be built and placed in the PATH by a previous step.
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current working directory for local test run: %v", err)
	}
	// Check for go.mod to confirm we are in the project root.
	if _, statErr := os.Stat(filepath.Join(projectRoot, "go.mod")); os.IsNotExist(statErr) {
		// If not in root, assume we are in tests/integration and go up two levels
		projectRoot = filepath.Join(projectRoot, "..", "..")
	}

	binDir := filepath.Join(projectRoot, "build")
	_ = os.MkdirAll(binDir, 0755)
	tempCLIPath := filepath.Join(binDir, "pvmlab_test")

	buildCmd := exec.Command("go", "build", "-o", tempCLIPath, "./pvmlab")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to build pvmlab binary for local test run: %v\n%s", err, string(output))
	}
	pathToCLI = tempCLIPath
	log.Printf("Using CLI path for tests: %s", pathToCLI)

	// Run the tests
	exitCode := m.Run()

	// Cleanup before exiting.
	if err := os.Remove(tempCLIPath); err != nil {
		log.Printf("Warning: failed to remove temporary CLI binary: %v", err)
	}
	if err := os.RemoveAll(tempHomeDir); err != nil {
		log.Printf("Warning: failed to remove temporary home directory: %v", err)
	}

	os.Exit(exitCode)
}

// runCmdWithLiveOutput executes a command, streams its output to the console in real-time,
// and also returns the captured output for validation.
func runCmdWithLiveOutput(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)

	err := cmd.Run()
	fullOutput := outBuf.String() + errBuf.String()

	if err != nil {
		// The error from cmd.Run() doesn't include the output, so we wrap it for better context.
		return fullOutput, fmt.Errorf("command failed with error: %w\n--- Output ---\n%s", err, fullOutput)
	}

	return fullOutput, nil
}

// TestVMLifecycle is a full integration test for the VM lifecycle.
func TestVMLifecycle(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	vmName := "test-vm-lifecycle"
	// Ensure cleanup happens even if the test fails
	defer func() {
		runCmdWithLiveOutput(pathToCLI, "vm", "clean", vmName) // Ignore error, just a best effort
	}()

	t.Run("0-Setup", func(t *testing.T) {
		output, err := runCmdWithLiveOutput(pathToCLI, "setup")
		if err != nil {
			t.Fatalf("setup failed: %v\n%s", err, output)
		}
	})

	t.Run("1-VMCleanInitial", func(t *testing.T) {
		output, err := runCmdWithLiveOutput(pathToCLI, "vm", "clean", vmName)
		// This might fail if the VM doesn't exist, which is fine.
		// We are just ensuring a clean slate.
		t.Logf("Initial clean output: %s", output)
		if err != nil {
			t.Logf("Initial clean command finished with an error (this is often expected): %v", err)
		}
	})

	t.Run("2-VMCreateProvisioner", func(t *testing.T) {
		// Using a non-default IP to avoid conflicts with existing setups.
		output, err := runCmdWithLiveOutput(pathToCLI, "vm", "create", vmName, "--role", "provisioner", "--ip", "192.168.254.1")
		if err != nil {
			t.Fatalf("vm create failed: %v\n%s", err, output)
		}
	})

	t.Run("3-VMStart", func(t *testing.T) {
		// Start the VM but don't wait for cloud-init within the CLI.
		_, err := runCmdWithLiveOutput(pathToCLI, "vm", "start", vmName, "--wait=false")
		if err != nil {
			t.Fatalf("vm start --wait=false failed: %v", err)
		}

		// Load metadata to find the SSH port.
		metaPath := filepath.Join(os.Getenv("PVMLAB_HOME"), ".provisioning-vm-lab", "vms", vmName+".json")
		metaFile, err := os.ReadFile(metaPath)
		if err != nil {
			t.Fatalf("failed to read metadata file: %v", err)
		}
		var meta struct {
			SSHPort int `json:"ssh_port"`
		}
		if err := json.Unmarshal(metaFile, &meta); err != nil {
			t.Fatalf("failed to unmarshal metadata: %v", err)
		}
		if meta.SSHPort == 0 {
			t.Fatal("SSH port is 0 in metadata")
		}

		// Tail the log file in the background for debugging, but don't rely on it for success.
		logPath := filepath.Join(os.Getenv("PVMLAB_HOME"), ".provisioning-vm-lab", "logs", vmName+".log")
		done := make(chan struct{})
		go tailFile(t, logPath, os.Stdout, done)
		defer close(done)

		// Wait for the SSH port to become available. This is the true test of readiness.
		timeout := 15 * time.Minute
		err = waitForPort(t, "localhost", meta.SSHPort, timeout)

		if err != nil {
			// If waiting fails, print a recursive listing of the app dir for debugging.
			debugDir := filepath.Join(os.Getenv("PVMLAB_HOME"), ".provisioning-vm-lab")
			log.Printf("--- Debugging directory structure for: %s ---", debugDir)
			debugCmd := exec.Command("ls", "-lR", debugDir)
			debugOutput, _ := debugCmd.CombinedOutput() // Ignore error, this is best-effort
			log.Println(string(debugOutput))
			log.Println("--- End debugging ---")
			t.Fatalf("waiting for SSH port %d failed: %v", meta.SSHPort, err)
		}
	})

	t.Run("4-VMList", func(t *testing.T) {
		output, err := runCmdWithLiveOutput(pathToCLI, "vm", "list")
		if err != nil {
			t.Fatalf("vm list failed: %v\n%s", err, output)
		}
		outStr := string(output)
		if !strings.Contains(outStr, vmName) {
			t.Errorf("vm list output does not contain the new VM name '%s'\nOutput:\n%s", vmName, outStr)
		}
		// The README shows "Running"
		if !strings.Contains(outStr, "Running") {
			t.Errorf("vm list output does not show the VM as 'Running'\nOutput:\n%s", outStr)
		}
	})

	t.Run("5-VMStop", func(t *testing.T) {
		output, err := runCmdWithLiveOutput(pathToCLI, "vm", "stop", vmName)
		if err != nil {
			t.Fatalf("vm stop failed: %v\n%s", err, output)
		}
	})

	t.Run("6-VMCleanFinal", func(t *testing.T) {
		output, err := runCmdWithLiveOutput(pathToCLI, "vm", "clean", vmName)
		if err != nil {
			t.Fatalf("final vm clean failed: %v\n%s", err, output)
		}
	})
}

// waitForPort polls a TCP port until it becomes available or a timeout is reached.
func waitForPort(t *testing.T, host string, port int, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)
	t.Logf("Waiting for port %s to become available...", address)

	timeoutChan := time.After(timeout)
	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("timed out waiting for port %s", address)
		default:
			conn, err := net.DialTimeout("tcp", address, 1*time.Second)
			if err == nil {
				conn.Close()
				t.Logf("Port %s is now available.", address)
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// tailFile tails a file and sends its content to the writer.
// It stops when the done channel is closed.
func tailFile(t *testing.T, filePath string, writer io.Writer, done chan struct{}) {
	var file *os.File
	var err error

	// Wait for the file to be created, checking for the done signal periodically.
	for {
		file, err = os.Open(filePath)
		if err == nil {
			break // File found
		}
		if !os.IsNotExist(err) {
			t.Logf("Error opening file %s, not a 'not exist' error: %v", filePath, err)
			return
		}

		// Wait a bit before retrying.
		select {
		case <-done:
			t.Logf("Stopped waiting for file %s because test is done.", filePath)
			return
		case <-time.After(100 * time.Millisecond):
			// continue to next attempt
		}
	}

	defer file.Close()
	t.Logf("Tailing file: %s", filePath)

	reader := bufio.NewReader(file)
	for {
		select {
		case <-done:
			t.Logf("Stopping tail of file: %s", filePath)
			return
		default:
			line, err := reader.ReadBytes('\n')
			if err == io.EOF {
				// If we hit the end of the file, wait a bit for more content,
				// but stop if the done signal is received.
				select {
				case <-done:
					t.Logf("Stopping tail of file: %s", filePath)
					return
				case <-time.After(100 * time.Millisecond):
					// continue reading
				}
				continue
			} else if err != nil {
				t.Logf("Error tailing file %s: %v", filePath, err)
				return
			}
			_, _ = writer.Write(line)
		}
	}
}
