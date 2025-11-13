package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDistroPull_Integration tests the distro pull command end-to-end.
// This test requires:
// - 7z binary installed
// - docker binary installed
// - network access to download ISO
// - RUN_INTEGRATION_TESTS=true environment variable
//
// To run: RUN_INTEGRATION_TESTS=true go test -v ./tests/integration -run TestDistroPull
func TestDistroPull_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if required binaries are not available
	if _, err := os.Stat("/usr/local/bin/7z"); os.IsNotExist(err) {
		if _, err := os.Stat("/opt/homebrew/bin/7z"); os.IsNotExist(err) {
			t.Skip("7z binary not found, skipping distro pull test")
		}
	}

	// Skip if docker is not available
	if _, err := os.Stat("/usr/local/bin/docker"); os.IsNotExist(err) {
		if _, err := os.Stat("/opt/homebrew/bin/docker"); os.IsNotExist(err) {
			if _, err := os.Stat("/usr/bin/docker"); os.IsNotExist(err) {
				t.Skip("docker binary not found, skipping distro pull test")
			}
		}
	}

	// TODO: add fedora
	tests := []struct {
		name         string
		distro       string
		arch         string
		expectError  bool
		skipCleanup  bool
		validateFunc func(t *testing.T, homeDir string)
	}{
		{
			name:        "pull ubuntu-24.04 aarch64",
			distro:      "ubuntu-24.04",
			arch:        "aarch64",
			expectError: false,
			validateFunc: func(t *testing.T, homeDir string) {
				// Verify expected files were created
				expectedDir := filepath.Join(homeDir, "images", "ubuntu-24.04", "aarch64")

				// Check directory exists
				if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
					t.Errorf("Expected directory not created: %s", expectedDir)
					return
				}

				// Check for expected files (kernel, initrd, etc.)
				// Note: The exact files depend on the distro implementation
				entries, err := os.ReadDir(expectedDir)
				if err != nil {
					t.Errorf("Failed to read distro directory: %v", err)
					return
				}

				if len(entries) == 0 {
					t.Error("Distro directory is empty, expected files to be extracted")
				}

				t.Logf("Successfully pulled distro, found %d files/dirs in %s", len(entries), expectedDir)
			},
		},
		{
			name:        "pull ubuntu-24.04 x86_64",
			distro:      "ubuntu-24.04",
			arch:        "x86_64",
			expectError: false,
			validateFunc: func(t *testing.T, homeDir string) {
				// Verify expected files were created
				expectedDir := filepath.Join(homeDir, "images", "ubuntu-24.04", "x86_64")

				// Check directory exists
				if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
					t.Errorf("Expected directory not created: %s", expectedDir)
					return
				}

				// Check for expected files (kernel, initrd, etc.)
				entries, err := os.ReadDir(expectedDir)
				if err != nil {
					t.Errorf("Failed to read distro directory: %v", err)
					return
				}

				if len(entries) == 0 {
					t.Error("Distro directory is empty, expected files to be extracted")
				}

				t.Logf("Successfully pulled distro, found %d files/dirs in %s", len(entries), expectedDir)
			},
		},
		{
			name:        "pull with invalid distro name",
			distro:      "nonexistent-distro",
			arch:        "aarch64",
			expectError: true,
			validateFunc: func(t *testing.T, homeDir string) {
				// No validation needed for error case
			},
		},
		{
			name:        "pull with invalid architecture",
			distro:      "ubuntu-24.04",
			arch:        "invalid-arch",
			expectError: true,
			validateFunc: func(t *testing.T, homeDir string) {
				// This should fail validation before attempting download
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a test-specific subdirectory to avoid conflicts
			testHomeDir := filepath.Join(os.Getenv("PVMLAB_HOME"), "distro_pull_test_"+strings.ReplaceAll(tt.name, " ", "_"))
			_ = os.MkdirAll(testHomeDir, 0755)
			defer func() {
				if !tt.skipCleanup {
					os.RemoveAll(testHomeDir)
				}
			}()

			// Set PVMLAB_HOME for this test
			oldHome := os.Getenv("PVMLAB_HOME")
			os.Setenv("PVMLAB_HOME", testHomeDir)
			defer os.Setenv("PVMLAB_HOME", oldHome)

			// Run distro pull command
			args := []string{"distro", "pull", "--distro", tt.distro, "--arch", tt.arch}
			output, err := runCmdWithLiveOutput(pathToCLI, args...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none. Output:\n%s", output)
				} else {
					t.Logf("Got expected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v\nOutput:\n%s", err, output)
			}

			// Validate the results
			if tt.validateFunc != nil {
				tt.validateFunc(t, testHomeDir)
			}
		})
	}
}

// TestDistroPull_Cancellation tests that the distro pull can be cancelled.
func TestDistroPull_Cancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require sending SIGINT to the running process
	// which is more complex. For now, we just test that the validation works.
	t.Skip("Cancellation test requires complex process control")
}

// TestDistroPull_ResumeAbility tests that pulling the same distro again doesn't re-download.
func TestDistroPull_ResumeAbility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Resume/idempotency test - requires checking if files already exist")

	// Future enhancement: This could test that:
	// 1. First pull downloads everything
	// 2. Second pull detects existing files and skips download
	// 3. Verify timestamps or use mock time to check behavior
}
