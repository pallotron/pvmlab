package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/ssh"

	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
)

// GetSSHArgs returns the necessary arguments for an SSH/SCP command to a VM.
// It handles the logic for connecting to a provisioner or a client VM via a proxy.
func GetSSHArgs(cfg *config.Config, meta *metadata.Metadata, forSCP bool) ([]string, error) {
	appDir := cfg.GetAppDir()
	sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")

	baseArgs := []string{
		"-4",
		"-i", sshKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}

	if meta.Role == "provisioner" {
		if meta.SSHPort == 0 {
			return nil, fmt.Errorf("SSH port not found in metadata, is the VM running?")
		}
		portArg := "-p"
		if forSCP {
			portArg = "-P"
		}
		return append(baseArgs, portArg, strconv.Itoa(meta.SSHPort)), nil
	}

	// Client VM
	provisioner, err := metadata.GetProvisioner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to find provisioner: %w", err)
	}
	if provisioner.SSHPort == 0 {
		return nil, fmt.Errorf("provisioner SSH port not found in metadata, is the provisioner running?")
	}
	provisionerPort := fmt.Sprintf("%d", provisioner.SSHPort)
	proxyCommand := fmt.Sprintf("ssh -4 -i %s -p %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %%h:%%p ubuntu@127.0.0.1", sshKeyPath, provisionerPort)

	return append(baseArgs, "-o", fmt.Sprintf("ProxyCommand=%s", proxyCommand)), nil
}

// GenerateKey generates a new SSH key pair and saves it to the specified path.
// If the key already exists, it does nothing.
var GenerateKey = func(privateKeyPath string) error {
	if _, err := os.Stat(privateKeyPath); err == nil {
		// Key already exists
		return nil
	}

	sshDir := filepath.Dir(privateKeyPath)
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		return fmt.Errorf("failed to create ssh directory: %w", err)
	}

	publicKeyPath := privateKeyPath + ".pub"

	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode the private key to the PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)
	if err := os.WriteFile(privateKeyPath, privateKeyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Create the public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create public key: %w", err)
	}

	// Write the public key to a file
	if err := os.WriteFile(publicKeyPath, ssh.MarshalAuthorizedKey(publicKey), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	if err := os.Chmod(publicKeyPath, 0644); err != nil {
		return fmt.Errorf("failed to set public key permissions: %w", err)
	}

	return nil
}