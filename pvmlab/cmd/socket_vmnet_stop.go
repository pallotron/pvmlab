package cmd

import (
	"fmt"
	"os/exec"
	"provisioning-vm-lab/internal/runner"

	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the socket_vmnet service",
	Long:  `Stops the socket_vmnet service using brew services.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Stopping socket_vmnet service... (this may require sudo password)")
		cmdRun := exec.Command("sudo", "brew", "services", "stop", "socket_vmnet")
		if err := runner.Run(cmdRun); err != nil {
			fmt.Println("Error stopping socket_vmnet service:", err)
		} else {
			fmt.Println("socket_vmnet service stopped successfully.")
		}
	},
}

func init() {
	socketVmnetCmd.AddCommand(serviceStopCmd)
}
