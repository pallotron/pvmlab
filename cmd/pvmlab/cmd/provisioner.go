package cmd

import (
	"github.com/spf13/cobra"
)

// provisionerCmd represents the provisioner command
var provisionerCmd = &cobra.Command{
	Use:   "provisioner",
	Short: "Manage the provisioner VM",
	Long:  `Manage the provisioner VM.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(provisionerCmd)
}
