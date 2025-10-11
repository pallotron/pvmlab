package cmd

import (
	"provisioning-vm-lab/internal/socketvmnet"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the socket_vmnet service",
	Long:  `Stops the socket_vmnet service using launchctl.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("i Stopping socket_vmnet service... (this may require sudo password)")
		if err := socketvmnet.StopSocketVmnet(); err != nil {
			return err
		}
		color.Green("✔ Socket_vmnet service stopped successfully.")
		return nil
	},
}

func init() {
	socketVmnetCmd.AddCommand(serviceStopCmd)
}
