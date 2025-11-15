package qemu

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// mockExecCommand is a helper to mock the exec.Command for testing.
func mockExecCommand(stdout, stderr string, err error) func(command string, args ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)

		env := os.Environ()
		env = append(env, "GO_WANT_HELPER_PROCESS=1")
		env = append(env, "STDOUT="+stdout)
		env = append(env, "STDERR="+stderr)
		if err != nil {
			env = append(env, "EXIT_CODE=1")
		} else {
			env = append(env, "EXIT_CODE=0")
		}
		cmd.Env = env
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	stdout := os.Getenv("STDOUT")
	stderr := os.Getenv("STDERR")
	exitCode := os.Getenv("EXIT_CODE")

	if stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	if exitCode == "1" {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestGetImageVirtualSize(t *testing.T) {
	tests := []struct {
		name          string
		mockStdout    string
		mockErr       error
		expectedSize  int64
		expectErr     bool
	}{
		{
			name: "Successful execution",
			mockStdout:   `{"virtual-size": 10737418240}`,
			mockErr:      nil,
			expectedSize: 10737418240,
			expectErr:    false,
		},
		{
			name: "qemu-img command fails",
			mockStdout:   "",
			mockErr:      errors.New("qemu-img failed"),
			expectedSize: 0,
			expectErr:    true,
		},
		{
			name: "Invalid JSON output",
			mockStdout:   `{"virtual-size": "not-a-number"}`,
			mockErr:      nil,
			expectedSize: 0,
			expectErr:    true,
		},
	}

	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = mockExecCommand(tt.mockStdout, "", tt.mockErr)
			size, err := GetImageVirtualSize("dummy-path")

			if (err != nil) != tt.expectErr {
				t.Errorf("GetImageVirtualSize() error = %v, wantErr %v", err, tt.expectErr)
				return
			}

			if size != tt.expectedSize {
				t.Errorf("GetImageVirtualSize() size = %v, want %v", size, tt.expectedSize)
			}
		})
	}
}