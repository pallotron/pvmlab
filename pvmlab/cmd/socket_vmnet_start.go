package cmd

import (
	"fmt"
	"provisioning-vm-lab/internal/brew"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the socket_vmnet service",
	Long:  `Starts the socket_vmnet service using brew services.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := brew.StartSocketVmnet(); err != nil {
			fmt.Println("Error starting socket_vmnet service:", err)
		}
	},
}

func init() {
	socketVmnetCmd.AddCommand(serviceStartCmd)
}
