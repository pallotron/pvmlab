package cmd

import (
	"github.com/spf13/cobra"
)

// systemCmd represents the system command
var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System-level configuration and setup",
	Long:  `Commands for system-level configuration and setup, such as installing launchd services.`,
}

func init() {
	rootCmd.AddCommand(systemCmd)
}
