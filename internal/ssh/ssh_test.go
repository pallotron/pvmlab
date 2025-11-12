package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
)

func TestGetSSHArgs_Provisioner(t *testing.T) {
	tests := []struct {
		name     string
		meta     *metadata.Metadata
		forSCP   bool
		wantErr  bool
		wantPort string
		wantFlag string
	}{
		{
			name: "provisioner with SSH port for SSH",
			meta: &metadata.Metadata{
				Role:    "provisioner",
				SSHPort: 2222,
			},
			forSCP:   false,
			wantErr:  false,
			wantPort: "2222",
			wantFlag: "-p",
		},
		{
			name: "provisioner with SSH port for SCP",
			meta: &metadata.Metadata{
				Role:    "provisioner",
				SSHPort: 2222,
			},
			forSCP:   true,
			wantErr:  false,
			wantPort: "2222",
			wantFlag: "-P",
		},
		{
			name: "provisioner without SSH port",
			meta: &metadata.Metadata{
				Role:    "provisioner",
				SSHPort: 0,
			},
			forSCP:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.New()
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			args, err := GetSSHArgs(cfg, tt.meta, tt.forSCP)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSSHArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Check that args contain the expected port flag
			found := false
			for i := 0; i < len(args)-1; i++ {
				if args[i] == tt.wantFlag && args[i+1] == tt.wantPort {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("GetSSHArgs() args = %v, want to contain [%q, %q]", args, tt.wantFlag, tt.wantPort)
			}

			// Check base args are present
			expectedBaseArgs := []string{"-4", "-i", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null"}
			for _, expected := range expectedBaseArgs {
				if !containsString(args, expected) {
					t.Errorf("GetSSHArgs() args = %v, want to contain %q", args, expected)
				}
			}
		})
	}
}

func TestGetSSHArgs_Client_ErrorHandling(t *testing.T) {
	// Test that client VM code path attempts to get provisioner
	// We can't easily test the "no provisioner" case without a clean database
	// but we can test the "no SSH port" case by checking error messages
	cfg, err := config.New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	meta := &metadata.Metadata{
		Role: "target",
		IP:   "192.168.1.10",
	}

	// Try to get SSH args for a client VM
	// This will either:
	// 1. Return an error about no provisioner (if none exists)
	// 2. Return an error about SSH port (if provisioner exists but has no port)
	// 3. Return args successfully (if a valid provisioner exists in the system)
	args, err := GetSSHArgs(cfg, meta, false)

	// We expect either an error OR valid args
	// If we get an error, it should mention provisioner or SSH port
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "provisioner") && !strings.Contains(errMsg, "ssh port") {
			t.Errorf("GetSSHArgs() error = %v, expected error about provisioner or SSH port", err)
		}
		return
	}

	// If no error, args should contain ProxyCommand for client VMs
	found := false
	for _, arg := range args {
		if strings.Contains(arg, "ProxyCommand=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetSSHArgs() for client VM should include ProxyCommand in args")
	}
}

func TestGenerateKey_NewKey(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "test_rsa")

	// Generate the key
	err := GenerateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v, want nil", err)
	}

	// Check that private key was created
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("GenerateKey() did not create private key at %s", privateKeyPath)
	}

	// Check that public key was created
	publicKeyPath := privateKeyPath + ".pub"
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Errorf("GenerateKey() did not create public key at %s", publicKeyPath)
	}

	// Check private key permissions
	info, err := os.Stat(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to stat private key: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("GenerateKey() private key permissions = %o, want 0600", info.Mode().Perm())
	}

	// Check public key permissions
	pubInfo, err := os.Stat(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to stat public key: %v", err)
	}
	if pubInfo.Mode().Perm() != 0644 {
		t.Errorf("GenerateKey() public key permissions = %o, want 0644", pubInfo.Mode().Perm())
	}
}

func TestGenerateKey_ExistingKey(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "existing_rsa")

	// Create an existing key file
	existingContent := []byte("existing key content")
	if err := os.WriteFile(privateKeyPath, existingContent, 0600); err != nil {
		t.Fatalf("Failed to create existing key: %v", err)
	}

	// Try to generate key (should not overwrite)
	err := GenerateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v, want nil", err)
	}

	// Check that the file was not modified
	content, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}

	if string(content) != string(existingContent) {
		t.Errorf("GenerateKey() modified existing key, want it to be unchanged")
	}
}

func TestGenerateKey_InvalidPath(t *testing.T) {
	// Try to create a key in a non-existent directory without creating parent
	// This should still work because GenerateKey creates the directory
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "subdir", "test_rsa")

	err := GenerateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("GenerateKey() should create parent directory, got error: %v", err)
	}

	// Verify the key was created
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("GenerateKey() did not create private key at %s", privateKeyPath)
	}
}

// Helper function
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
