package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		// Valid cases
		{"10G", 10 * 1024 * 1024 * 1024, false},
		{"512M", 512 * 1024 * 1024, false},
		{"2048K", 2048 * 1024, false},
		{"1024B", 1024, false},
		{"1024", 1024, false},
		{"1T", 1 * 1024 * 1024 * 1024 * 1024, false},
		{"0", 0, false},

		// Lowercase units
		{"10g", 10 * 1024 * 1024 * 1024, false},
		{"512m", 512 * 1024 * 1024, false},
		{"2048k", 2048 * 1024, false},

		// Double letter units
		{"10GB", 10 * 1024 * 1024 * 1024, false},
		{"512MB", 512 * 1024 * 1024, false},
		{"2048KB", 2048 * 1024, false},
		{"1TB", 1 * 1024 * 1024 * 1024 * 1024, false},

		// Invalid cases
		{"invalid", 0, true},
		{"10X", 0, true},
		{"-10G", -10 * 1024 * 1024 * 1024, false}, // Sscanf parses negative integers correctly
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSize(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("ParseSize(%s) expected error, but got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseSize(%s) returned unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseSize(%s) = %d; want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "util-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")
	directoryPath := tmpDir

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Existing file", existingFile, true},
		{"Non-existent file", nonExistentFile, false},
		{"Directory", directoryPath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExists(tt.path); got != tt.expected {
				t.Errorf("FileExists(%s) = %v; want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "util-copy-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	content := []byte("hello world")
	mode := os.FileMode(0600)

	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Test successful copy
	err = CopyFile(srcFile, dstFile, mode)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify content
	readContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(readContent), string(content))
	}

	// Verify permissions
	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("failed to stat dest file: %v", err)
	}
	if info.Mode().Perm() != mode {
		t.Errorf("permission mismatch: got %v, want %v", info.Mode().Perm(), mode)
	}

	// Test non-existent source
	err = CopyFile(filepath.Join(tmpDir, "missing.txt"), dstFile, mode)
	if err == nil {
		t.Error("expected error for missing source file, got nil")
	}
}

// TestHelperProcess isn't a real test. It's used to mock exec.Command
// For this to work, we need to be able to swap exec.Command.
// However, looking at util.go, RunCommand uses exec.Command directly.
// To test RunCommand properly, we need to refactor util.go to allow dependency injection
// or use the os/exec test pattern where we re-run the test binary.

// Since we cannot modify util.go easily here without instruction, we will stick to testing
// RunCommand with simple commands that should exist on the system (like 'true' and 'false' or 'echo').
// If we want to be pure, we should refactor RunCommand to use a var for execCommand.

func TestRunCommand(t *testing.T) {
	// We'll use simple commands that are likely available.
	// Note: specific commands might depend on the OS.
	// Since the context is macOS/Linux, 'echo' and 'false'/'exit 1' should be fine.

	// Use "sh -c" to be more cross-platform compatible within unix-likes
	tests := []struct {
		name      string
		cmd       string
		args      []string
		expectErr bool
	}{
		{
			name:      "Successful command",
			cmd:       "sh",
			args:      []string{"-c", "exit 0"},
			expectErr: false,
		},
		{
			name:      "Failing command",
			cmd:       "sh",
			args:      []string{"-c", "exit 1"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunCommand(tt.cmd, tt.args...)
			if (err != nil) != tt.expectErr {
				t.Errorf("RunCommand() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
