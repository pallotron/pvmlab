package cmd

import (
	"os/exec"
	"provisioning-vm-lab/internal/runner"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the socket_vmnet service",
	Long:  `Stops the socket_vmnet service using brew services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Stopping socket_vmnet service... (this may require sudo password)"
		s.Start()
		defer s.Stop()

		cmdRun := exec.Command("sudo", "brew", "services", "stop", "socket_vmnet")
		if err := runner.Run(cmdRun); err != nil {
			s.FinalMSG = color.RedString("✖ Error stopping socket_vmnet service.\n")
			return err
		}
		s.FinalMSG = color.GreenString("✔ Socket_vmnet service stopped successfully.\n")
		return nil
	},
}

func init() {
	socketVmnetCmd.AddCommand(serviceStopCmd)
}
