package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the socket_vmnet service",
	Long:  `Starts the socket_vmnet service using brew services.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting socket_vmnet service...")
		cmdRun := exec.Command("brew", "services", "start", "socket_vmnet")
		if err := cmdRun.Run(); err != nil {
			fmt.Println("Error starting socket_vmnet service:", err)
		} else {
			fmt.Println("socket_vmnet service started successfully.")
		}
	},
}

func init() {
	serviceCmd.AddCommand(serviceStartCmd)
}
