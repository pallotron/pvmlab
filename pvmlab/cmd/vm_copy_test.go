package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCopy_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
			errMsg:  "copy command requires exactly two arguments",
		},
		{
			name:    "one argument",
			args:    []string{"source"},
			wantErr: true,
			errMsg:  "copy command requires exactly two arguments",
		},
		{
			name:    "three arguments",
			args:    []string{"source", "dest", "extra"},
			wantErr: true,
			errMsg:  "copy command requires exactly two arguments",
		},
		{
			name:    "both remote",
			args:    []string{"vm1:/path1", "vm2:/path2"},
			wantErr: true,
			errMsg:  "copying between two VMs is not supported",
		},
		{
			name:    "both local",
			args:    []string{"/local/path1", "/local/path2"},
			wantErr: true,
			errMsg:  "at least one of source or destination must be a remote path",
		},
		{
			name:    "invalid remote source format - no path",
			args:    []string{"vm1:", "/local/path"},
			wantErr: true,
			errMsg:  "invalid remote source format",
		},
		{
			name:    "invalid remote destination format - no path",
			args:    []string{"/local/path", "vm1:"},
			wantErr: true,
			errMsg:  "invalid remote destination format",
		},
		{
			name:    "valid format but non-existent VM",
			args:    []string{"nonexistent-vm:/remote/path", "/local/path"},
			wantErr: true,
			// This will fail when trying to load metadata
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			err := copy(cmd, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("copy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("copy() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestCopy_RemotePathParsing(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		destination    string
		wantErr        bool
		expectRemote   bool
	}{
		{
			name:         "remote source to local",
			source:       "vm-name:/remote/path",
			destination:  "/local/path",
			wantErr:      true, // Will fail because VM doesn't exist
			expectRemote: true,
		},
		{
			name:         "local to remote destination",
			source:       "/local/path",
			destination:  "vm-name:/remote/path",
			wantErr:      true, // Will fail because VM doesn't exist
			expectRemote: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			err := copy(cmd, []string{tt.source, tt.destination})

			if (err != nil) != tt.wantErr {
				t.Errorf("copy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
