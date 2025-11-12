package cmd

import (
	"os"
	"testing"
)

func TestGetSocketVMNetClientPath(t *testing.T) {
	// Save original env var
	originalEnv := os.Getenv("PVMLAB_SOCKET_VMNET_CLIENT")
	defer func() {
		if originalEnv != "" {
			os.Setenv("PVMLAB_SOCKET_VMNET_CLIENT", originalEnv)
		} else {
			os.Unsetenv("PVMLAB_SOCKET_VMNET_CLIENT")
		}
	}()

	tests := []struct {
		name    string
		envVar  string
		wantErr bool
	}{
		{
			name:    "env var set",
			envVar:  "/custom/path/socket_vmnet_client",
			wantErr: false,
		},
		{
			name:    "env var not set - will search paths",
			envVar:  "",
			wantErr: false, // May succeed or fail depending on system
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("PVMLAB_SOCKET_VMNET_CLIENT", tt.envVar)
			} else {
				os.Unsetenv("PVMLAB_SOCKET_VMNET_CLIENT")
			}

			result, err := getSocketVMNetClientPath()

			if tt.envVar != "" {
				// When env var is set, it should always succeed and return that path
				if err != nil {
					t.Errorf("getSocketVMNetClientPath() unexpected error when env var set: %v", err)
				}
				if result != tt.envVar {
					t.Errorf("getSocketVMNetClientPath() = %q, want %q", result, tt.envVar)
				}
			} else {
				// When env var is not set, it searches system paths
				// We don't enforce success/failure as it depends on the system
				t.Logf("getSocketVMNetClientPath() result: %q, err: %v", result, err)
			}
		})
	}
}

func TestFindFile(t *testing.T) {
	// Create a temp directory with a test file
	tmpDir := t.TempDir()
	existingFile := tmpDir + "/existing.txt"
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		paths   []string
		wantErr bool
		want    string
	}{
		{
			name:    "file found in first path",
			paths:   []string{existingFile, "/non/existent/path1"},
			wantErr: false,
			want:    existingFile,
		},
		{
			name:    "file found in second path",
			paths:   []string{"/non/existent/path1", existingFile},
			wantErr: false,
			want:    existingFile,
		},
		{
			name:    "file not found in any path",
			paths:   []string{"/non/existent/path1", "/non/existent/path2"},
			wantErr: true,
		},
		{
			name:    "empty paths list",
			paths:   []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findFile(tt.paths)

			if (err != nil) != tt.wantErr {
				t.Errorf("findFile(%v) error = %v, wantErr %v", tt.paths, err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("findFile(%v) = %q, want %q", tt.paths, result, tt.want)
			}
		})
	}
}
