package ssh

import (
	"fmt"
	"path/filepath"
	"strconv"

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
