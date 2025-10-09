package cmd

import (
	"github.com/spf13/cobra"
)

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Manage docker containers in a VM",
}

func init() {
	vmCmd.AddCommand(dockerCmd)
}
