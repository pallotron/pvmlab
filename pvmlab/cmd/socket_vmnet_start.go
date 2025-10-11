package cmd

import (
	"provisioning-vm-lab/internal/socketvmnet"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the socket_vmnet service",
	Long:  `Starts the socket_vmnet service using brew services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Starting socket_vmnet service... (this may require sudo password)"
		s.Start()
		defer s.Stop()

		if err := socketvmnet.StartSocketVmnet(); err != nil {
			s.FinalMSG = color.RedString("✖ Error starting socket_vmnet service.\n")
			return err
		}
		s.FinalMSG = color.GreenString("✔ Socket_vmnet service started successfully.\n")
		return nil
	},
}

func init() {
	socketVmnetCmd.AddCommand(serviceStartCmd)
}
