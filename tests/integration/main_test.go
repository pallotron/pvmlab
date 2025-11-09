package integration

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	pathToCLI = "pvmlab"
)

// TestMain handles setup and teardown for the integration tests.
func TestMain(m *testing.M) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		log.Println("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
		os.Exit(0)
	}

	var tempHomeDir string
	var err error
	pvmlabHome := os.Getenv("PVMLAB_HOME")
	if pvmlabHome != "" {
		log.Printf("Using existing PVMLAB_HOME: %s", pvmlabHome)
		tempHomeDir = pvmlabHome
	} else {
		tempHomeDir, err = os.MkdirTemp("/tmp", "pvmlab_test_*")
		if err != nil {
			log.Fatalf("failed to create temp dir for integration test: %v", err)
		}
		os.Setenv("PVMLAB_HOME", tempHomeDir)
		log.Printf("Using temporary home for tests: %s", tempHomeDir)
	}

	// This because github actions macos workers do not have hvf accelleration
	os.Setenv("PVMLAB_WAIT_TIMEOUT", "2400")

	log.Printf("Using temporary home for tests: %s", tempHomeDir)

	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current working directory: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(projectRoot, "go.mod")); os.IsNotExist(statErr) {
		projectRoot = filepath.Join(projectRoot, "..", "..")
	}

	// Build the pxeboot stack tar for local runs
	if os.Getenv("CI") != "true" {
		log.Println("Building pxeboot_stack.tar for local integration tests...")
		makeCmd := exec.Command("make", "tar")
		makeCmd.Dir = filepath.Join(projectRoot, "pxeboot_stack")
		if output, err := makeCmd.CombinedOutput(); err != nil {
			log.Fatalf("failed to build pxeboot_stack.tar for local test run: %v\n%s", err, string(output))
		}
	}

	// --- Self-contained socket_vmnet setup ---
	cleanupSocketVMNet := setupSocketVMNet(tempHomeDir, projectRoot)
	defer cleanupSocketVMNet()
	// --- End of self-contained setup ---

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

	exitCode := m.Run()

	if os.Getenv("PVMLAB_DEBUG") != "true" {
		if err := os.Remove(tempCLIPath); err != nil {
			log.Printf("Warning: failed to remove temporary CLI binary: %v", err)
		}
	} else {
		log.Printf("PVMLAB_DEBUG is true, leaving test binary for inspection: %s", tempCLIPath)
	}

	// Only remove the directory if the test created it and not in CI
	if pvmlabHome == "" && os.Getenv("PVMLAB_DEBUG") != "true" && os.Getenv("CI") != "true" {
		os.RemoveAll(tempHomeDir)
	} else if os.Getenv("PVMLAB_DEBUG") == "true" {
		log.Printf("PVMLAB_DEBUG is true, leaving temp home dir for inspection: %s", tempHomeDir)
	}

	os.Exit(exitCode)
}

// TestVMLifecycle is a full integration test for the VM lifecycle.
func TestVMLifecycle(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	provisionerName := "test-provisioner"
	clientName := "test-client"
	provisionerIP := "192.168.254.1/24"
	provisionerIPv6 := "fd00:cafe:baba::1/64"
	clientIP := "192.168.254.2/24"
	clientIPv6 := "fd00:cafe:baba::2/64"

	defer func() {
		runCmdWithLiveOutput(pathToCLI, "clean", "--purge")
	}()

	defer func() {
		r := recover()
		if os.Getenv("PVMLAB_DEBUG") != "true" && os.Getenv("CI") != "true" {
			runCmdWithLiveOutput(pathToCLI, "vm", "clean", provisionerName)
			runCmdWithLiveOutput(pathToCLI, "vm", "clean", clientName)
		} else {
			log.Println("PVMLAB_DEBUG is true, leaving VMs for inspection.")
		}
		if r != nil {
			panic(r) // re-panic after cleanup
		}
	}()

	// Step 0: Setup assets only
	if !t.Run("0-SetupAssetsOnly", func(t *testing.T) {
		runCmdOrFail(t, pathToCLI, "setup", "--assets-only")
	}) {
		t.FailNow()
	}

	// Step 1: Create Provisioner
	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(projectRoot, "go.mod")); os.IsNotExist(statErr) {
		projectRoot = filepath.Join(projectRoot, "..", "..")
	}
	if !t.Run("1-VMCreateProvisioner", func(t *testing.T) {
		runCmdOrFail(
			t, pathToCLI,
			"provisioner", "create", provisionerName,
			"--ip", provisionerIP, "--ipv6", provisionerIPv6,
			"--docker-pxeboot-stack-tar", filepath.Join(projectRoot, "pxeboot_stack", "pxeboot_stack.tar"),
		)
	}) {
		t.FailNow()
	}

	if !t.Run("2-VMStartProvisioner", func(t *testing.T) {
		runCmdOrFail(t, pathToCLI, "vm", "start", provisionerName, "--wait=true")
	}) {
		t.FailNow()
	}

	if !t.Run("3-VerifyProvisionerNetwork", func(t *testing.T) {
		// We can't ping google.com anymore, but we can check if services are up.
		// This test is now more about internal health than external connectivity.
		output := runCmdOrFail(t, pathToCLI, "vm", "shell", provisionerName, "--", "ip", "addr")
		if !strings.Contains(output, "inet 192.168.254.1") {
			t.Errorf("provisioner does not have the expected IPv4 address. Output:\n%s", output)
		}
		if !strings.Contains(output, "inet6 fd00:cafe:baba::1") {
			t.Errorf("provisioner does not have the expected IPv6 address. Output:\n%s", output)
		}
		runCmdOrFail(t, pathToCLI, "vm", "shell", provisionerName, "--", "ping", "-c", "4", "google.com")
	}) {
		t.FailNow()
	}

	if !t.Run("4-VerifyProvisionerServices", func(t *testing.T) {
		output := runCmdOrFail(t, pathToCLI, "vm", "shell", provisionerName, "--", "systemctl", "is-active", "radvd")
		if !strings.Contains(strings.TrimSpace(output), "active") {
			t.Fatalf("radvd service is not active on provisioner. Initial output:\n%s", output)
		}

		output = runCmdOrFail(t, pathToCLI, "vm", "shell", provisionerName, "--", "sudo", "docker", "ps")
		if !strings.Contains(output, "pxeboot_stack") {
			pxeboot_output := runCmdOrFail(t,
				pathToCLI, "vm", "shell", provisionerName, "--",
				"sudo", "journalctl", "-xfeu", "pxeboot.service", "--no-pager",
			)
			t.Fatalf("pxeboot stack container is not running on provisioner. \npxeboot.service logs:\n%s", pxeboot_output)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("5-VMCreateClient", func(t *testing.T) {
		runCmdOrFail(t, pathToCLI, "vm", "create", clientName, "--ip", clientIP, "--ipv6", clientIPv6, "--distro", "ubuntu-24.04")
	}) {
		t.FailNow()
	}

	if !t.Run("6-VMStartClient", func(t *testing.T) {
		runCmdOrFail(t, pathToCLI, "vm", "start", clientName, "--wait=true")
	}) {
		t.FailNow()
	}

	if !t.Run("7-VerifyClientNetwork", func(t *testing.T) {
		output := runCmdOrFail(t, pathToCLI, "vm", "shell", clientName, "--", "ip", "addr")

		if !strings.Contains(output, "inet 192.168.254.2") {
			t.Errorf("client does not have the expected IPv4 address. Output:\n%s", output)
		}
		// if !strings.Contains(output, "inet6 fd00:cafe:baba::2") {
		// 	t.Errorf("client does not have the expected IPv6 address. Output:\n%s", output)
		// }
		runCmdOrFail(t, pathToCLI, "vm", "shell", clientName, "--", "ping", "-c", "4", "google.com")
	}) {
		t.FailNow()
	}

	if !t.Run("8-VMList", func(t *testing.T) {
		output := runCmdOrFail(t, pathToCLI, "vm", "list")
		if !strings.Contains(output, provisionerName) {
			t.Errorf("vm list output does not contain the provisioner name '%s'\nOutput:\n%s", provisionerName, output)
		}
		if !strings.Contains(output, clientName) {
			t.Errorf("vm list output does not contain the client name '%s'\nOutput:\n%s", clientName, output)
		}
		if strings.Count(output, "Running") != 2 {
			t.Errorf("expected 2 running VMs, but vm list shows something else.\nOutput:\n%s", output)
		}
	}) {
		t.FailNow()
	}

	if !t.Run("9-VMStopAll", func(t *testing.T) {
		if os.Getenv("PVMLAB_DEBUG") == "true" {
			t.Log("PVMLAB_DEBUG is true, skipping VM stop for inspection.")
			t.Skip()
		}
		runCmdOrFail(t, pathToCLI, "vm", "stop", provisionerName)
		runCmdOrFail(t, pathToCLI, "vm", "stop", clientName)
	}) {
		t.FailNow()
	}

	t.Run("10-VMCleanFinal", func(t *testing.T) {
		if os.Getenv("PVMLAB_DEBUG") == "true" || os.Getenv("CI") == "true" {
			t.Log("PVMLAB_DEBUG is true or running in CI, skipping VM clean for inspection.")
			t.Skip()
		}
		runCmdOrFail(t, pathToCLI, "vm", "clean", provisionerName)
		output := runCmdOrFail(t, pathToCLI, "vm", "list")
		if strings.Contains(output, provisionerName) {
			t.Fatalf("vm list output still contains the provisioner name '%s'\nOutput:\n%s", provisionerName, output)
		}

		runCmdOrFail(t, pathToCLI, "vm", "clean", clientName)
		output = runCmdOrFail(t, pathToCLI, "vm", "list")
		if strings.Contains(output, clientName) {
			t.Fatalf("vm list output still contains the client name '%s'\nOutput:\n%s", clientName, output)
		}

		runCmdOrFail(t, pathToCLI, "clean", "--purge")
	})
}
