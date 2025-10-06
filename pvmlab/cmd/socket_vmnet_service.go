package cmd

import (
	"github.com/spf13/cobra"
)

// socketVmnetServiceCmd represents the service command
var socketVmnetServiceCmd = &cobra.Command{
	Use:   "socket_vmnet_service",
	Short: "Manage the socket_vmnet service",
	Long:  `Manage the socket_vmnet service.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(socketVmnetServiceCmd)
}
