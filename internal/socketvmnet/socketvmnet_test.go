package socketvmnet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHelperProcess isn't a real test. It's used as a helper process
// to simulate the behavior of the `launchctl` command.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) > 0 {
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command specified")
		os.Exit(1)
	}

	cmd, args := args[0], args[1:]
	if cmd == "sudo" {
		cmd, args = args[0], args[1:]
	}

	if cmd == "launchctl" && len(args) > 1 && args[0] == "list" {
		if strings.Contains(os.Getenv("LAUNCHCTL_LIST_OUTPUT"), "PID") {
			fmt.Println(os.Getenv("LAUNCHCTL_LIST_OUTPUT"))
			os.Exit(0)
		} else {
			os.Exit(1) // Simulate service not found
		}
	}

	if cmd == "launchctl" && len(args) > 1 && args[0] == "start" {
		if os.Getenv("LAUNCHCTL_START_FAIL") == "1" {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if cmd == "launchctl" && len(args) > 1 && args[0] == "stop" {
		if os.Getenv("LAUNCHCTL_STOP_FAIL") == "1" {
			os.Exit(1)
		}
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "Unknown command: %v", args)
	os.Exit(1)
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func TestIsSocketVmnetRunning(t *testing.T) {
	originalExecCommand := execCommand
	execCommand = mockExecCommand
	t.Cleanup(func() {
		execCommand = originalExecCommand
	})

	t.Run("service is running", func(t *testing.T) {
		os.Setenv("LAUNCHCTL_LIST_OUTPUT", `{"PID": 123}`)
		defer os.Unsetenv("LAUNCHCTL_LIST_OUTPUT")

		running, err := IsSocketVmnetRunning()
		if err != nil {
			t.Fatalf("IsSocketVmnetRunning() returned an error: %v", err)
		}
		if !running {
			t.Errorf("IsSocketVmnetRunning() = false, want true")
		}
	})

	t.Run("service is not running", func(t *testing.T) {
		os.Setenv("LAUNCHCTL_LIST_OUTPUT", "")
		defer os.Unsetenv("LAUNCHCTL_LIST_OUTPUT")

		running, err := IsSocketVmnetRunning()
		if err != nil {
			t.Fatalf("IsSocketVmnetRunning() returned an error: %v", err)
		}
		if running {
			t.Errorf("IsSocketVmnetRunning() = true, want false")
		}
	})
}

func TestStartSocketVmnet(t *testing.T) {
	originalExecCommand := execCommand
	execCommand = mockExecCommand
	t.Cleanup(func() {
		execCommand = originalExecCommand
	})

	t.Run("start succeeds", func(t *testing.T) {
		os.Unsetenv("LAUNCHCTL_START_FAIL")
		err := StartSocketVmnet()
		if err != nil {
			t.Fatalf("StartSocketVmnet() returned an error: %v", err)
		}
	})

	t.Run("start fails", func(t *testing.T) {
		os.Setenv("LAUNCHCTL_START_FAIL", "1")
		defer os.Unsetenv("LAUNCHCTL_START_FAIL")
		err := StartSocketVmnet()
		if err == nil {
			t.Fatal("StartSocketVmnet() did not return an error")
		}
	})
}

func TestStopSocketVmnet(t *testing.T) {
	originalExecCommand := execCommand
	execCommand = mockExecCommand
	t.Cleanup(func() {
		execCommand = originalExecCommand
	})

	t.Run("stop succeeds", func(t *testing.T) {
		os.Unsetenv("LAUNCHCTL_STOP_FAIL")
		err := StopSocketVmnet()
		if err != nil {
			t.Fatalf("StopSocketVmnet() returned an error: %v", err)
		}
	})

	t.Run("stop fails", func(t *testing.T) {
		os.Setenv("LAUNCHCTL_STOP_FAIL", "1")
		defer os.Unsetenv("LAUNCHCTL_STOP_FAIL")
		err := StopSocketVmnet()
		if err == nil {
			t.Fatal("StopSocketVmnet() did not return an error")
		}
	})
}