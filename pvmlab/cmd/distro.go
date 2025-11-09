package cmd

import (
	"github.com/spf13/cobra"
)

// distroCmd represents the distro command
var distroCmd = &cobra.Command{
	Use:   "distro",
	Short: "Manage distributions",
	Long:  `Manage distributions that can be used to provision VMs.`,
}

func init() {
	rootCmd.AddCommand(distroCmd)
}
