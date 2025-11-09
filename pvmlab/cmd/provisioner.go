package cmd

import (
	"github.com/spf13/cobra"
)

// provisionerCmd represents the provisioner command
var provisionerCmd = &cobra.Command{
	Use:   "provisioner",
	Short: "Manage the provisioner VM",
	Long:  `Manage the special-purpose provisioner VM, which provides PXE boot and other services to target VMs.`,
}

func init() {
	rootCmd.AddCommand(provisionerCmd)
}
