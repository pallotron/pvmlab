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
	Use:               "shell <vm-name> [command]",
	Short:             "Connects to a VM via SSH or executes a command",
	Long:              `Connects to a VM via SSH. If a command is provided, it will be executed non-interactively.`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

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

		baseSSHArgs := []string{
			"-4",
			"-i", sshKeyPath,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
		}

		if meta.Role == "provisioner" {
			if meta.SSHPort == 0 {
				return fmt.Errorf("SSH port not found in metadata, is the VM running?")
			}
			sshPort := fmt.Sprintf("%d", meta.SSHPort)
			provisionerArgs := append(baseSSHArgs, "-p", sshPort, "ubuntu@127.0.0.1")
			sshCmd = exec.Command("ssh", provisionerArgs...)
		} else {
			provisioner, err := metadata.GetProvisioner(cfg)
			if err != nil {
				return fmt.Errorf("failed to find provisioner: %w", err)
			}
			if provisioner.SSHPort == 0 {
				return fmt.Errorf("provisioner SSH port not found in metadata, is the provisioner running?")
			}
			provisionerPort := fmt.Sprintf("%d", provisioner.SSHPort)
			proxyCommand := fmt.Sprintf("ssh -4 -i %s -p %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %%h:%%p ubuntu@127.0.0.1", sshKeyPath, provisionerPort)
			targetIP := meta.IP
			targetConnect := fmt.Sprintf("ubuntu@%s", targetIP)
			targetArgs := append(baseSSHArgs, "-o", fmt.Sprintf("ProxyCommand=%s", proxyCommand), targetConnect)
			sshCmd = exec.Command("ssh", targetArgs...)
		}

		// If there are arguments after the vmName, treat them as a command to execute
		if len(args) > 1 {
			sshCmd.Args = append(sshCmd.Args, args[1:]...)
		} else {
			color.Cyan("i Connecting to VM via SSH: %s", vmName)
		}

		sshCmd.Stdout = os.Stdout
		sshCmd.Stdin = os.Stdin
		sshCmd.Stderr = os.Stderr

		if err := sshCmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// SSH commands often exit with a non-zero status, which is not necessarily an error in execution.
				// We just pass on the exit code.
				os.Exit(exitErr.ExitCode())
			}
			return fmt.Errorf("error running ssh: %w", err)
		}
		return nil
	},
}

func init() {
	vmCmd.AddCommand(vmShellCmd)
}
