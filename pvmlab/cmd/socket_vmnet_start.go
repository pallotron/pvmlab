package cmd

import (
	"pvmlab/internal/socketvmnet"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the socket_vmnet service",
	Long:  `Starts the socket_vmnet service using launchctl.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("i Starting socket_vmnet service... (this may require sudo password)")
		if err := socketvmnet.StartSocketVmnet(); err != nil {
			return err
		}
		color.Green("✔ %s service started successfully.", socketvmnet.ServiceName)
		return nil
	},
}

func init() {
	socketVmnetCmd.AddCommand(serviceStartCmd)
}
