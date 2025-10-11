package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"

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
			// TODO: if it's a target VM we need to ssh thru the provisioner. To be done.
			return fmt.Errorf("target VM not supported for now. Please ssh from the provisioner VM")
		}

		sshCmd.Stdout = os.Stdout
		sshCmd.Stdin = os.Stdin
		sshCmd.Stderr = os.Stderr

		if err := sshCmd.Run(); err != nil {
			// Don't print error on normal SSH exit
		}
		return nil
	},
}

func init() {
	vmCmd.AddCommand(vmShellCmd)
}
