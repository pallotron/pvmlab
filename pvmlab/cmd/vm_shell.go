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

// vmShellCmd represents the shell command
var vmShellCmd = &cobra.Command{
	Use:   "shell <vm-name>",
	Short: "Connects to a VM via SSH",
	Long:  `Connects to a VM via SSH.`,
	Args:  cobra.ExactArgs(1),
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
			fmt.Println("Connecting to provisioner VM via forwarded port 2222...")
			sshCmd = exec.Command("ssh", "-i", sshKeyPath, "-p", "2222", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost")
		} else {
			// TODO: if it's a target VM we need to ssh thru the provisioner. To be done.
			fmt.Println("Error: Target VM not supported for now. Please ssh from the provisioner VM.")
			os.Exit(1)
		}

		sshCmd.Stdout = os.Stdout
		sshCmd.Stdin = os.Stdin
		sshCmd.Stderr = os.Stderr

		if err := sshCmd.Run(); err != nil {
			// Don't print error on normal SSH exit
		}
	},
}

func init() {
	vmCmd.AddCommand(vmShellCmd)
}
