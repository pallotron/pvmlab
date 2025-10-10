
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"

	"github.com/spf13/cobra"
)

// vmLogsCmd represents the logs command
var vmLogsCmd = &cobra.Command{
	Use:   "logs <vm-name>",
	Short: "Tails the console logs for a VM",
	Long:  `Tails the console logs for a VM.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]
		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		logPath := filepath.Join(appDir, "logs", vmName+".log")
		tailCmd := exec.Command("tail", "-f", logPath)
		tailCmd.Stdout = os.Stdout
		tailCmd.Stderr = os.Stderr

		if err := tailCmd.Run(); err != nil {
			fmt.Println("Error tailing log file:", err)
		}
	},
}

func init() {
	vmCmd.AddCommand(vmLogsCmd)
}
