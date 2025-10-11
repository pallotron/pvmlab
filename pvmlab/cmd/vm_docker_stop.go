package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var dockerStopCmd = &cobra.Command{
	Use:   "stop <vm_name> <container_name>",
	Short: "Stop a docker container in a VM",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		containerName := args[1]
		color.Cyan("i Stopping docker container %s in %s", containerName, vmName)

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
			remoteCmd := fmt.Sprintf("sudo docker stop %s", containerName)
			sshCmd = exec.Command(
				"ssh", "-i", sshKeyPath, "-p", "2222", "-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost",
				remoteCmd,
			)
		} else {
			return fmt.Errorf("error: Target VM not supported for now. Please ssh from the provisioner VM")
		}

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error stopping container %s: %w\n%s", containerName, err, string(output))
		}
		color.Green("✔ Container %s stopped successfully.", containerName)
		fmt.Println(string(output))
		return nil
	},
}

func init() {
	dockerCmd.AddCommand(dockerStopCmd)
}
