package cmd

import (
	"provisioning-vm-lab/internal/socketvmnet"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// socketVmnetServiceStatusCmd represents the status command
var socketVmnetServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of the socket_vmnet service",
	Long:  `Checks the status of the socket_vmnet service using launchctl.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("i Checking socket_vmnet service status... (this may require sudo password)")
		running, err := socketvmnet.IsSocketVmnetRunning()
		if err != nil {
			return err
		}

		if running {
			color.Green("✔ %s service is running.", socketvmnet.ServiceName)
		} else {
			color.Yellow("i %s service is stopped.", socketvmnet.ServiceName)
		}
		return nil
	},
}

func init() {
	socketVmnetCmd.AddCommand(socketVmnetServiceStatusCmd)
}
