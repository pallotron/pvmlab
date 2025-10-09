package cmd

import (
	"fmt"
	"provisioning-vm-lab/internal/socketvmnet"

	"github.com/spf13/cobra"
)

// socketVmnetServiceStatusCmd represents the status command
var socketVmnetServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of the socket_vmnet service",
	Long:  `Checks the status of the socket_vmnet service using brew services.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking socket_vmnet service status... (this may require sudo password)")
		running, err := socketvmnet.IsSocketVmnetRunning()
		if err != nil {
			fmt.Println("Error checking status:", err)
			return
		}

		if running {
			fmt.Printf("%s service is running.\n", socketvmnet.ServiceName)
		} else {
			fmt.Printf("%s service is stopped.", socketvmnet.ServiceName)
		}
	},
}

func init() {
	socketVmnetCmd.AddCommand(socketVmnetServiceStatusCmd)
}
