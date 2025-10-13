package cmd

import (
	"github.com/spf13/cobra"
)

// vmCmd represents the vm command
var vmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Manage VMs",
	Long:  `Manage the provisioner and target VMs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(vmCmd)
}
