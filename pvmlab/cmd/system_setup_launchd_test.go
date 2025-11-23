package cmd

import (
	"os"
	"testing"
)

func TestSystemSetupLaunchdCmd(t *testing.T) {
	// Save original functions
	origGeteuid := osGeteuid
	origMkdirAll := osMkdirAll
	origExecutable := osExecutable
	origGetwd := osGetwd
	origCopyFile := utilCopyFile
	origRunCommand := utilRunCommand
	origFileExists := utilFileExists

	// Restore after test
	defer func() {
		osGeteuid = origGeteuid
		osMkdirAll = origMkdirAll
		osExecutable = origExecutable
		osGetwd = origGetwd
		utilCopyFile = origCopyFile
		utilRunCommand = origRunCommand
		utilFileExists = origFileExists
	}()

	// Mocks
	mockGeteuid := func() int { return 0 }
	mockMkdirAll := func(path string, perm os.FileMode) error { return nil }
	mockExecutable := func() (string, error) { return "/mock/bin/pvmlab", nil }
	mockGetwd := func() (string, error) { return "/mock/cwd", nil }
	mockCopyFile := func(src, dst string, mode os.FileMode) error { return nil }
	
	var runCommandCalls []string
	mockRunCommand := func(name string, args ...string) error {
		cmd := name
		for _, arg := range args {
			cmd += " " + arg
		}
		runCommandCalls = append(runCommandCalls, cmd)
		return nil
	}

	mockFileExists := func(path string) bool { return true }

	// Set mocks
	osGeteuid = mockGeteuid
	osMkdirAll = mockMkdirAll
	osExecutable = mockExecutable
	osGetwd = mockGetwd
	utilCopyFile = mockCopyFile
	utilRunCommand = mockRunCommand
	utilFileExists = mockFileExists

	t.Run("Requires Root", func(t *testing.T) {
		osGeteuid = func() int { return 1000 }
		defer func() { osGeteuid = mockGeteuid }()

		err := systemSetupLaunchdCmd.RunE(systemSetupLaunchdCmd, []string{})
		if err == nil {
			t.Error("Expected error when not running as root, got nil")
		}
	})

	t.Run("Success with Homebrew Layout", func(t *testing.T) {
		runCommandCalls = []string{}
		// Setup file existence for Homebrew layout
		// /mock/libexec/socket_vmnet_wrapper.sh
		// /mock/io.github.pallotron.pvmlab.socket_vmnet.plist
		utilFileExists = func(path string) bool {
			if path == "/mock/libexec/socket_vmnet_wrapper.sh" {
				return true
			}
			if path == "/mock/io.github.pallotron.pvmlab.socket_vmnet.plist" {
				return true
			}
			return false
		}

		err := systemSetupLaunchdCmd.RunE(systemSetupLaunchdCmd, []string{})
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		// Verify calls
		expectedCalls := []string{
			"launchctl bootout system /Library/LaunchDaemons/io.github.pallotron.pvmlab.socket_vmnet.plist",
			"launchctl enable system/io.github.pallotron.pvmlab.socket_vmnet",
			"launchctl bootstrap system /Library/LaunchDaemons/io.github.pallotron.pvmlab.socket_vmnet.plist",
			"launchctl kickstart -kp system/io.github.pallotron.pvmlab.socket_vmnet",
		}

		if len(runCommandCalls) != len(expectedCalls) {
			t.Errorf("Expected %d runCommand calls, got %d", len(expectedCalls), len(runCommandCalls))
		}

		for i, call := range runCommandCalls {
			if call != expectedCalls[i] {
				t.Errorf("Call %d: expected '%s', got '%s'", i, expectedCalls[i], call)
			}
		}
	})

	t.Run("Success with Dev Layout", func(t *testing.T) {
		runCommandCalls = []string{}
		utilFileExists = func(path string) bool {
			// Dev layout paths relative to /mock/bin/pvmlab -> /mock/bin
			// /mock/launchd/socket_vmnet_wrapper.sh
			if path == "/mock/launchd/socket_vmnet_wrapper.sh" || path == "/mock/launchd/io.github.pallotron.pvmlab.socket_vmnet.plist" {
				return true
			}
			return false
		}

		err := systemSetupLaunchdCmd.RunE(systemSetupLaunchdCmd, []string{})
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	})

	t.Run("Files Not Found", func(t *testing.T) {
		utilFileExists = func(path string) bool { return false }
		err := systemSetupLaunchdCmd.RunE(systemSetupLaunchdCmd, []string{})
		if err == nil {
			t.Error("Expected error when files not found, got nil")
		}
	})
}
