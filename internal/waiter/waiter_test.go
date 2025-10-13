package waiter

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestForPort(t *testing.T) {
	// Find a free port
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port

	// Test success case
	go func() {
		time.Sleep(200 * time.Millisecond)
		l.Accept()
	}()

	err = ForPort("localhost", port, 1*time.Second)
	if err != nil {
		t.Errorf("ForPort() returned an error for an available port: %v", err)
	}
	l.Close()

	// Test timeout case
	freePort := l.Addr().(*net.TCPAddr).Port // Get a new free port by closing the listener
	err = ForPort("localhost", freePort, 200*time.Millisecond)
	if err == nil {
		t.Error("ForPort() did not return an error for an unavailable port")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("ForPort() returned wrong error type: got %v, want timeout", err)
	}
}

func TestForMessage(t *testing.T) {
	// Setup temp log file
	tmpfile, err := os.CreateTemp("", "testlog.*.log")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Test success case
	go func() {
		time.Sleep(200 * time.Millisecond)
		tmpfile.WriteString("this is the magic message\n")
		tmpfile.Close()
	}()

	err = ForMessage(tmpfile.Name(), "magic message", 1*time.Second)
	if err != nil {
		t.Errorf("ForMessage() returned an error when message was present: %v", err)
	}

	// Test timeout case
	tmpfile2, err := os.CreateTemp("", "testlog2.*.log")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile2.Name())

	err = ForMessage(tmpfile2.Name(), "never appears", 200*time.Millisecond)
	if err == nil {
		t.Error("ForMessage() did not return an error when message was absent")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("ForMessage() returned wrong error type: got %v, want timeout", err)
	}
}

// mockExecCommand is a helper for mocking exec.Command
func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess isn't a real test. It's used as a helper process
// for TestForCloudInitTarget.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	if cmd == "ssh" {
		// Simulate ssh output for cloud-init check
		fmt.Fprint(os.Stdout, "ActiveState=active")
	}
}

func TestForCloudInitProvisioner(t *testing.T) {
	// Replace the real exec command with our mock
	originalExecCommand := execCommand
	execCommand = mockExecCommand
	defer func() { execCommand = originalExecCommand }()

	err := ForCloudInitProvisioner(22, "/dev/null", 500*time.Millisecond)
	if err != nil {
		t.Errorf("ForCloudInitProvisioner() returned an error: %v", err)
	}
}

func TestForCloudInitTarget(t *testing.T) {
	// Replace the real exec command with our mock
	originalExecCommand := execCommand
	execCommand = mockExecCommand
	defer func() { execCommand = originalExecCommand }()

	err := ForCloudInitTarget(2222, "192.168.1.1", "/dev/null", 500*time.Millisecond)
	if err != nil {
		t.Errorf("ForCloudInitTarget() returned an error: %v", err)
	}
}
