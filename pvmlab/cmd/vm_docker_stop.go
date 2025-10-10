package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"

	"github.com/spf13/cobra"
)

var dockerStopCmd = &cobra.Command{
	Use:   "stop <vm_name> <container_name>",
	Short: "Stop a docker container in a VM",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: VmNameCompleter,
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]
		containerName := args[1]

		meta, err := metadata.Load(vmName)
		if err != nil {
			fmt.Println("Error loading VM metadata:", err)
			os.Exit(1)
		}

		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

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
			fmt.Println("Error: Target VM not supported for now. Please ssh from the provisioner VM.")
			os.Exit(1)
		}

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error stopping container %s: %s\n", containerName, err)
			fmt.Println(string(output))
			os.Exit(1)
		}
		fmt.Printf("Container %s stopped successfully.\n", containerName)
		fmt.Println(string(output))
	},
}

func init() {
	dockerCmd.AddCommand(dockerStopCmd)
}
