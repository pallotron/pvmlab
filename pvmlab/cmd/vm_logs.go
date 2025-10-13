
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"

	"github.com/spf13/cobra"
)

// vmLogsCmd represents the logs command
var vmLogsCmd = &cobra.Command{
	Use:   "logs <vm-name>",
	Short: "Tails the console logs for a VM",
	Long:  `Tails the console logs for a VM.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		cfg, err := config.New()
		if err != nil {
			return err
		}
		appDir := cfg.GetAppDir()

		logPath := filepath.Join(appDir, "logs", vmName+".log")
		tailCmd := exec.Command("tail", "-f", logPath)
		tailCmd.Stdout = os.Stdout
		tailCmd.Stderr = os.Stderr

		if err := tailCmd.Run(); err != nil {
			return fmt.Errorf("error tailing log file: %w", err)
		}
		return nil
	},
}

func init() {
	vmCmd.AddCommand(vmLogsCmd)
}
