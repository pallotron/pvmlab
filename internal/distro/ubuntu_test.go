package distro

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "file exists",
			setup: func(t *testing.T) string {
				tmpFile, err := os.CreateTemp("", "test-*.txt")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer tmpFile.Close()
				return tmpFile.Name()
			},
			expected: true,
		},
		{
			name: "directory exists",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			expected: true,
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			expected: false,
		},
		{
			name: "empty path",
			setup: func(t *testing.T) string {
				return ""
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)

			// Clean up temp file if it exists
			if tt.expected && path != "" {
				defer os.Remove(path)
			}

			result := fileExists(path)

			if result != tt.expected {
				t.Errorf("fileExists(%q) = %v, want %v", path, result, tt.expected)
			}
		})
	}
}
