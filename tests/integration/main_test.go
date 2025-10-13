package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pvmlab/internal/waiter"
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
		_, _ = runCmdWithLiveOutput(pathToCLI, "vm", "clean", vmName) // Ignore error, just a best effort
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
		metaPath := filepath.Join(os.Getenv("PVMLAB_HOME"), ".pvmlab", "vms", vmName+".json")
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

		// Wait for the SSH port to become available. This is the true test of readiness.
		timeout := 15 * time.Minute
		err = waiter.ForPort("localhost", meta.SSHPort, timeout)
		if err != nil {
			t.Fatalf("waiting for SSH port %d failed: %v", meta.SSHPort, err)
		}

		// On local runs, perform the stricter check to ensure cloud-init completes fully.
		// In CI, just checking for the SSH port is a sufficient smoke test.
		if os.Getenv("CI") != "true" {
			sshKeyPath := filepath.Join(os.Getenv("PVMLAB_HOME"), ".pvmlab", "ssh", "vm_rsa")
			err = waiter.ForCloudInitProvisioner(meta.SSHPort, sshKeyPath, timeout)
			if err != nil {
				// If waiting fails, print a recursive listing of the app dir for debugging.
				debugDir := filepath.Join(os.Getenv("PVMLAB_HOME"), ".pvmlab")
				log.Printf("--- Debugging directory structure for: %s ---", debugDir)
				debugCmd := exec.Command("ls", "-lR", debugDir)
				debugOutput, _ := debugCmd.CombinedOutput() // Ignore error, this is best-effort
				log.Println(string(debugOutput))
				log.Println("--- End debugging ---")
				t.Fatalf("waiting for cloud-init target failed: %v", err)
			}
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
