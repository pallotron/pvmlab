package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// vmShellCmd represents the shell command
var vmShellCmd = &cobra.Command{
	Use:               "shell <vm-name>",
	Short:             "Connects to a VM via SSH",
	Long:              `Connects to a VM via SSH.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Connecting to VM via SSH: %s", vmName)

		cfg, err := config.New()
		if err != nil {
			return err
		}

		meta, err := metadata.Load(cfg, vmName)
		if err != nil {
			return fmt.Errorf("error loading VM metadata: %w", err)
		}

		appDir := cfg.GetAppDir()

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		var sshCmd *exec.Cmd

		if meta.Role == "provisioner" {
			if meta.SSHPort == 0 {
				return fmt.Errorf("SSH port not found in metadata, is the VM running?")
			}
			sshPort := fmt.Sprintf("%d", meta.SSHPort)
			color.Cyan("i Connecting to provisioner VM via forwarded port %s...", sshPort)
			sshCmd = exec.Command("ssh", "-i", sshKeyPath, "-p", sshPort, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost")
		} else {
			provisioner, err := metadata.GetProvisioner(cfg)
			if err != nil {
				return fmt.Errorf("failed to find provisioner: %w", err)
			}

			if provisioner.SSHPort == 0 {
				return fmt.Errorf("provisioner SSH port not found in metadata, is the provisioner running?")
			}

			provisionerPort := fmt.Sprintf("%d", provisioner.SSHPort)
			var targetIP string
if meta.IPv6 != "" {
    targetIP = meta.IPv6
} else {
    targetIP = meta.IP
}
targetConnect := fmt.Sprintf("ubuntu@%s", targetIP)
			proxyCommand := fmt.Sprintf("ssh -i %s -p %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %%h:%%p ubuntu@localhost", sshKeyPath, provisionerPort)

			color.Cyan("i Connecting to target VM via provisioner on port %s...", provisionerPort)
			sshCmd = exec.Command("ssh", "-i", sshKeyPath, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", fmt.Sprintf("ProxyCommand=%s", proxyCommand), targetConnect)
		}

		sshCmd.Stdout = os.Stdout
		sshCmd.Stdin = os.Stdin
		sshCmd.Stderr = os.Stderr

		if err := sshCmd.Run(); err != nil {
			// Ignore normal SSH exit errors.
			if _, ok := err.(*exec.ExitError); !ok {
				return fmt.Errorf("error running ssh: %w", err)
			}
		}
		return nil
	},
}

func init() {
	vmCmd.AddCommand(vmShellCmd)
}
