package cmd

import (
	"os"
	"testing"
)

func TestSystemSetupLaunchdCmd(t *testing.T) {
	// Save original functions
	origGeteuid := osGeteuid
	origMkdirAll := osMkdirAll
	origCopyFile := utilCopyFile
	origRunCommand := utilRunCommand

	// Restore after test
	defer func() {
		osGeteuid = origGeteuid
		osMkdirAll = origMkdirAll
		utilCopyFile = origCopyFile
		utilRunCommand = origRunCommand
	}()

	// Mocks
	mockGeteuid := func() int { return 0 }
	mockMkdirAll := func(path string, perm os.FileMode) error { return nil }
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

	// Set mocks
	osGeteuid = mockGeteuid
	osMkdirAll = mockMkdirAll
	utilCopyFile = mockCopyFile
	utilRunCommand = mockRunCommand

	t.Run("Requires Root", func(t *testing.T) {
		osGeteuid = func() int { return 1000 }
		defer func() { osGeteuid = mockGeteuid }()

		err := systemSetupLaunchdCmd.RunE(systemSetupLaunchdCmd, []string{})
		if err == nil {
			t.Error("Expected error when not running as root, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		runCommandCalls = []string{}
		
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
}