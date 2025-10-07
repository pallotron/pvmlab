package cmd

import (
	"github.com/spf13/cobra"
)

// socketVmnetCmd represents the service command
var socketVmnetCmd = &cobra.Command{
	Use:   "socket_vmnet",
	Short: "Manage the socket_vmnet service",
	Long:  `Manage the socket_vmnet service.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(socketVmnetCmd)
}
