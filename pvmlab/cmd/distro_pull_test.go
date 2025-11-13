package cmd

import (
	"strings"
	"testing"
)

func TestDistroPullCmd(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "missing distro flag - empty string",
			args:          []string{"distro", "pull", "--distro", ""},
			expectedError: "--distro is required",
		},
		{
			name:          "invalid architecture - arm",
			args:          []string{"distro", "pull", "--distro", "ubuntu-24.04", "--arch", "arm"},
			expectedError: "--arch must be either 'aarch64' or 'x86_64'",
		},
		{
			name:          "invalid architecture - amd64",
			args:          []string{"distro", "pull", "--distro", "ubuntu-24.04", "--arch", "amd64"},
			expectedError: "--arch must be either 'aarch64' or 'x86_64'",
		},
		{
			name:          "invalid architecture - arm64",
			args:          []string{"distro", "pull", "--distro", "ubuntu-24.04", "--arch", "arm64"},
			expectedError: "--arch must be either 'aarch64' or 'x86_64'",
		},
		{
			name:          "invalid architecture - x64",
			args:          []string{"distro", "pull", "--distro", "ubuntu-24.04", "--arch", "x64"},
			expectedError: "--arch must be either 'aarch64' or 'x86_64'",
		},
		// Note: We can't test successful execution without mocking distro.Pull
		// which is a regular function, not a variable. The successful case would
		// require actual downloads or significant refactoring to inject dependencies.
		// The validation logic above provides good coverage of the error paths.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset distroName and distroPullArch globals
			distroName = ""
			distroPullArch = "aarch64"

			// Reset root command for each test
			rootCmd.SetArgs(tt.args[1:])

			output, _, err := executeCommand(rootCmd, tt.args...)

			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected an error, but got none. output: %s", output)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', but got '%v'", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, but got: %v", err)
				}
			}
		})
	}
}

func TestDistroPullCmd_FlagDefaults(t *testing.T) {
	// Test that command accepts the expected flags
	// We can't test execution without actual downloads,
	// but we can verify the flags are registered correctly

	// Check that distroPullCmd has the expected flags
	distroFlag := distroPullCmd.Flags().Lookup("distro")
	if distroFlag == nil {
		t.Error("--distro flag should be registered")
	} else {
		if distroFlag.DefValue != "ubuntu-24.04" {
			t.Errorf("--distro default = %q, want %q", distroFlag.DefValue, "ubuntu-24.04")
		}
	}

	archFlag := distroPullCmd.Flags().Lookup("arch")
	if archFlag == nil {
		t.Error("--arch flag should be registered")
	} else {
		if archFlag.DefValue != "aarch64" {
			t.Errorf("--arch default = %q, want %q", archFlag.DefValue, "aarch64")
		}
	}
}

func TestDistroPullCmd_ValidArchitectures(t *testing.T) {
	// Test that only aarch64 and x86_64 are considered valid
	validArchs := []string{"aarch64", "x86_64"}

	for _, arch := range validArchs {
		t.Run("accepts_"+arch, func(t *testing.T) {
			// This will fail at distro.Pull() call since we can't mock it,
			// but it should NOT fail at the architecture validation step
			args := []string{"distro", "pull", "--distro", "ubuntu-24.04", "--arch", arch}
			rootCmd.SetArgs(args[1:])

			_, _, err := executeCommand(rootCmd, args...)

			// We expect an error (from distro.Pull), but NOT the arch validation error
			if err != nil && strings.Contains(err.Error(), "--arch must be either") {
				t.Errorf("valid architecture %q should pass validation", arch)
			}
		})
	}
}
