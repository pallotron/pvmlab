package cmd

import (
	"provisioning-vm-lab/internal/socketvmnet"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// socketVmnetServiceStatusCmd represents the status command
var socketVmnetServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of the socket_vmnet service",
	Long:  `Checks the status of the socket_vmnet service using brew services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Checking socket_vmnet service status... (this may require sudo password)"
		s.Start()
		defer s.Stop()

		running, err := socketvmnet.IsSocketVmnetRunning()
		if err != nil {
			s.FinalMSG = color.RedString("✖ Error checking status.\n")
			return err
		}

		if running {
			s.FinalMSG = color.GreenString("✔ %s service is running.\n", socketvmnet.ServiceName)
		} else {
			s.FinalMSG = color.YellowString("i %s service is stopped.\n", socketvmnet.ServiceName)
		}
		return nil
	},
}

func init() {
	socketVmnetCmd.AddCommand(socketVmnetServiceStatusCmd)
}
