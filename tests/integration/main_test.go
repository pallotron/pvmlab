package integration

import (
	"log"
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

	// Defer cleanup of the temporary binary
	defer os.Remove(tempCLIPath)

	// Run the tests
	exitCode := m.Run()
	os.Exit(exitCode)
}

// TestVMLifecycle is a full integration test for the VM lifecycle.
func TestVMLifecycle(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	vmName := "test-vm-lifecycle"
	// Ensure cleanup happens even if the test fails
	defer func() {
		cmd := exec.Command(pathToCLI, "vm", "clean", vmName)
		cmd.Env = os.Environ()
		cmd.Run() // Ignore error, just a best effort
	}()

	t.Run("0-Setup", func(t *testing.T) {
		cmd := exec.Command(pathToCLI, "setup")
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup failed: %v\n%s", err, string(output))
		}
	})

	t.Run("1-VMCleanInitial", func(t *testing.T) {
		cmd := exec.Command(pathToCLI, "vm", "clean", vmName)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		// This might fail if the VM doesn't exist, which is fine.
		// We are just ensuring a clean slate.
		t.Logf("Initial clean output: %s", string(output))
		if err != nil {
			t.Logf("Initial clean command finished with an error (this is often expected): %v", err)
		}
	})

	t.Run("2-VMCreateProvisioner", func(t *testing.T) {
		// Using a non-default IP to avoid conflicts with existing setups.
		cmd := exec.Command(pathToCLI, "vm", "create", vmName, "--role", "provisioner", "--ip", "192.168.254.1")
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("vm create failed: %v\n%s", err, string(output))
		}
	})

	t.Run("3-VMStart", func(t *testing.T) {
		cmd := exec.Command(pathToCLI, "vm", "start", vmName)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("vm start failed: %v\n%s", err, string(output))
		}
		// Give the VM a moment to boot. This might need adjustment.
		time.Sleep(20 * time.Second)
	})

	t.Run("4-VMList", func(t *testing.T) {
		cmd := exec.Command(pathToCLI, "vm", "list")
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("vm list failed: %v\n%s", err, string(output))
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
		cmd := exec.Command(pathToCLI, "vm", "stop", vmName)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("vm stop failed: %v\n%s", err, string(output))
		}
	})

	t.Run("6-VMCleanFinal", func(t *testing.T) {
		cmd := exec.Command(pathToCLI, "vm", "clean", vmName)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("final vm clean failed: %v\n%s", err, string(output))
		}
	})
}
