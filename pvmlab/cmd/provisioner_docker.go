package cmd

import (
	"github.com/spf13/cobra"
)

var provisionerDockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Manages the docker daemon in the provisioner VM",
}

func init() {
	provisionerCmd.AddCommand(provisionerDockerCmd)
}
