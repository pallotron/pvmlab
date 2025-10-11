package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pvmlab",
	Short: "pvmlab is a CLI for managing provisioning VM labs",
	// SilenceErrors is used to prevent cobra from printing the error,
	// as we handle it ourselves in the Execute function.
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Print the help message if no subcommand is provided
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// color.Error is a pre-configured Color object that writes to os.Stderr in red
		color.Red("Error: %v\n", err)
		os.Exit(1)
	}
}
