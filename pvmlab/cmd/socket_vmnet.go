package cmd

import (
	"github.com/spf13/cobra"
)

// socketVmnetCmd represents the service command
var socketVmnetCmd = &cobra.Command{
	Use:   "socket_vmnet",
	Short: "Manage the socket_vmnet service",
	Long:  `Manage the socket_vmnet service.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(socketVmnetCmd)
}
