package cmd

import (
	"fmt"
	"provisioning-vm-lab/internal/brew"

	"github.com/spf13/cobra"
)

// socketVmnetServiceStatusCmd represents the status command
var socketVmnetServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of the socket_vmnet service",
	Long:  `Checks the status of the socket_vmnet service using brew services.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking socket_vmnet service status... (this may require sudo password)")
		running, err := brew.IsSocketVmnetRunning()
		if err != nil {
			fmt.Println("Error checking status:", err)
			return
		}

		if running {
			fmt.Println("socket_vmnet service is running.")
		} else {
			fmt.Println("socket_vmnet service is stopped.")
		}
	},
}

func init() {
	socketVmnetCmd.AddCommand(socketVmnetServiceStatusCmd)
}
