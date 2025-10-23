package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/ssh"

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

		sshArgs, err := ssh.GetSSHArgs(cfg, meta, false)
		if err != nil {
			return err
		}

		var target string
		if meta.Role == "provisioner" {
			target = "ubuntu@127.0.0.1"
		} else {
			target = fmt.Sprintf("ubuntu@%s", meta.IP)
		}

		finalArgs := append(sshArgs, target)
		sshCmd := exec.Command("ssh", finalArgs...)

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
