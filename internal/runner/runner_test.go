package runner

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	// The "true" command should always succeed.
	cmd := exec.Command("true")
	err := run(cmd)
	if err != nil {
		t.Errorf("run() with a succeeding command returned an error: %v", err)
	}
}

func TestRunFailure(t *testing.T) {
	// The "false" command should always fail.
	cmd := exec.Command("false")
	err := run(cmd)
	if err == nil {
		t.Errorf("run() with a failing command did not return an error")
	}

	// Check if the error message contains the command string.
	// This is a basic check to ensure our error formatting is working.
	if err != nil && !strings.Contains(err.Error(), "command failed") {
		t.Errorf("run() error message for failing command was not in the expected format: %v", err)
	}
}

func TestRunFailureWithOutput(t *testing.T) {
	// This command will fail and print a specific message to stderr.
	cmd := exec.Command("sh", "-c", "echo 'test error' >&2; exit 1")
	err := run(cmd)
	if err == nil {
		t.Fatal("run() with a failing command did not return an error")
	}

	expectedOutput := "test error"
	if !strings.Contains(err.Error(), expectedOutput) {
		t.Errorf("run() error message did not contain the command's output. Got: %q, want to contain: %q", err.Error(), expectedOutput)
	}
}
