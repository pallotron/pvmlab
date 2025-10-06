
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"

	"github.com/spf13/cobra"
)

// vmShellCmd represents the shell command
var vmShellCmd = &cobra.Command{
	Use:   "shell <vm-name>",
	Short: "Connects to a VM via SSH",
	Long:  `Connects to a VM via SSH. Currently only supported for the 'provisioner' VM.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]

		if vmName != "provisioner" {
			fmt.Println("SSH shell is currently only supported for the 'provisioner' VM.")
			fmt.Println("Networking for target VMs is TBD.")
			os.Exit(1)
		}

		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		sshCmd := exec.Command("ssh", "-i", sshKeyPath, "-p", "2222", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost")
		sshCmd.Stdout = os.Stdout
		sshCmd.Stdin = os.Stdin
		sshCmd.Stderr = os.Stderr

		if err := sshCmd.Run(); err != nil {
			fmt.Println("Error connecting to VM:", err)
		}
	},
}

func init() {
	vmCmd.AddCommand(vmShellCmd)
}
