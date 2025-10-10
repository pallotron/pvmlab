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

var dockerStatusCmd = &cobra.Command{
	Use:   "status <vm_name>",
	Short: "Show docker container status in a VM",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]

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
			remoteCmd := "sudo docker ps -a"
			sshCmd = exec.Command(
				"ssh", "-i", sshKeyPath,
				"-p", "2222", "-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost", remoteCmd,
			)
		} else {
			fmt.Println("Error: Target VM not supported for now. Please ssh from the provisioner VM.")
			os.Exit(1)
		}

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error getting docker status: %s\n", err)
			fmt.Println(string(output))
			os.Exit(1)
		}
		fmt.Println(string(output))
	},
}

func init() {
	dockerCmd.AddCommand(dockerStatusCmd)
}
