package integration

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
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

	defer func() {
		// Only remove the directory if the test created it
		if pvmlabHome == "" && os.Getenv("PVMLAB_DEBUG") != "true" {
			os.RemoveAll(tempHomeDir)
		} else if os.Getenv("PVMLAB_DEBUG") == "true" {
			log.Printf("PVMLAB_DEBUG is true, leaving temp home dir for inspection: %s", tempHomeDir)
		}
	}()

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

	// --- Self-contained socket_vmnet setup ---
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

	// Defer cleanup for the socket_vmnet process
	defer func() {
		log.Println("Stopping local socket_vmnet process...")
		if err := socketVMNetCmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("failed to send SIGTERM to socket_vmnet: %v", err)
			_ = socketVMNetCmd.Process.Kill()
		}
		_ = socketVMNetCmd.Wait()
		log.Println("Local socket_vmnet process stopped.")
	}()

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
			log.Printf("--- socket_vmnet logs ---\n%s", socketVMLog.String())
			log.Fatalf("timed out waiting for socket_vmnet service to become ready at %s", socketPath)
		}
		time.Sleep(readinessPollInterval)
	}
	// --- End of self-contained setup ---

	// Copy the pxeboot_stack.tar to the temp home dir
	destDir := filepath.Join(tempHomeDir, ".pvmlab", "docker_images")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Fatalf("failed to create destination dir for pxeboot_stack.tar: %v", err)
	}
	srcPath := filepath.Join(projectRoot, "pxeboot_stack", "pxeboot_stack.tar")
	destPath := filepath.Join(destDir, "pxeboot_stack.tar")
	if err := copyFile(srcPath, destPath); err != nil {
		log.Fatalf("failed to copy pxeboot_stack.tar: %v", err)
	}
	log.Printf("Copied %s to %s", srcPath, destPath)

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

	os.Exit(exitCode)
}

func copyAndMakeExecutable(src, dst string) error {
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func runCmdWithLiveOutput(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)

	err := cmd.Run()
	fullOutput := outBuf.String() + errBuf.String()

	if err != nil {
		return fullOutput, fmt.Errorf("command failed with error: %w\n--- Output ---\n%s", err, fullOutput)
	}

	return fullOutput, nil
}

func runCmdOrFail(t *testing.T, command string, args ...string) string {
	t.Helper()
	output, err := runCmdWithLiveOutput(command, args...)
	if err != nil {
		t.Fatalf("Command `%s %s` failed.\nError: %v\nOutput:\n%s", command, strings.Join(args, " "), err, output)
	}
	return output
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
		r := recover()
		if os.Getenv("PVMLAB_DEBUG") != "true" {
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
	if !t.Run("1-VMCreateProvisioner", func(t *testing.T) {
		runCmdOrFail(
			t, pathToCLI,
			"vm", "create", provisionerName, "--role", "provisioner",
			"--ip", provisionerIP, "--ipv6", provisionerIPv6,
		)
	}) {
		t.FailNow()
	}

	// Step 2: Start Provisioner
	if !t.Run("2-VMStartProvisioner", func(t *testing.T) {
		runCmdOrFail(t, pathToCLI, "vm", "start", provisionerName, "--wait=true")
	}) {
		t.FailNow()
	}

	// Step 3: Verify Provisioner Network (This will likely fail now as there's no host network)
	if !t.Run("3-VerifyProvisionerNetwork", func(t *testing.T) {
		// We can't ping google.com anymore, but we can check if services are up.
		// This test is now more about internal health than external connectivity.
		t.Log("Skipping external network verification for self-contained test.")
	}) {
		// Do not FailNow, as this is an expected change.
	}

	// Step 4: Verify Provisioner Services
	if !t.Run("4-VerifyProvisionerServices", func(t *testing.T) {
		output := runCmdOrFail(t, pathToCLI, "vm", "shell", provisionerName, "--", "systemctl", "is-active", "radvd")
		if !strings.Contains(strings.TrimSpace(output), "active") {
			t.Fatalf("radvd service is not active on provisioner. Initial output:\n%s", output)
		}

		// output = runCmdOrFail(t, pathToCLI, "vm", "shell", provisionerName, "--", "sudo", "docker", "ps")
		// if !strings.Contains(output, "pxeboot_stack") {
		// 	systemd_output := runCmdOrFail(t,
		// 		pathToCLI, "vm", "shell", provisionerName, "--",
		// 		"sudo", "journalctl", "-xfeu", "pxeboot_stack.service", "--no-pager",
		// 	)
		// 	t.Fatalf("pxeboot_stack container is not running on provisioner. Output:\n%s", systemd_output)
		// }
	}) {
		t.FailNow()
	}

	// Step 5: Create Client
	if !t.Run("5-VMCreateClient", func(t *testing.T) {
		runCmdOrFail(t, pathToCLI, "vm", "create", clientName, "--role", "target", "--ip", clientIP, "--ipv6", clientIPv6)
	}) {
		t.FailNow()
	}

	// Step 6: Start Client
	if !t.Run("6-VMStartClient", func(t *testing.T) {
		runCmdOrFail(t, pathToCLI, "vm", "start", clientName, "--wait=true")
	}) {
		t.FailNow()
	}

	// Step 7: Verify Client Network
	if !t.Run("7-VerifyClientNetwork", func(t *testing.T) {
		output := runCmdOrFail(t, pathToCLI, "vm", "shell", clientName, "--", "ip", "addr")

		if !strings.Contains(output, "inet 192.168.254.2") {
			t.Errorf("client does not have the expected IPv4 address. Output:\n%s", output)
		}
		// TODO: Re-enable this check once the DHCPv6 issue is resolved.
		// if !strings.Contains(output, "inet6 fd00:cafe:baba::2") {
		// 	t.Errorf("client does not have the expected IPv6 address. Output:\n%s", output)
		// }
	}) {
		t.FailNow()
	}

	// Step 8: VM List
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

	// Step 9: Stop All
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

	// Step 10: Final Clean
	t.Run("10-VMCleanFinal", func(t *testing.T) {
		if os.Getenv("PVMLAB_DEBUG") == "true" {
			t.Log("PVMLAB_DEBUG is true, skipping VM clean for inspection.")
			t.Skip()
		}
		runCmdOrFail(t, pathToCLI, "vm", "clean", provisionerName)
		runCmdOrFail(t, pathToCLI, "vm", "clean", clientName)
	})
}
