package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pvmlab",
	Short: "pvmlab is a CLI for managing provisioning VM labs",
	Run: func(cmd *cobra.Command, args []string) {
		// Print the help message if no subcommand is provided
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
