package cmd

import (
	"fmt"
	"os"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var purge bool

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Stops all VMs and removes all generated files",
	Long: `Stops all running VMs and cleans up their associated files.
The socket_vmnet service is left running.
Use the --purge flag to remove the entire ~/.provisioning-vm-lab directory.`, 
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("i Cleaning up all VMs...")
		cfg, err := config.New()
		if err != nil {
			return err
		}
		allMeta, err := metadata.GetAll(cfg)
		if err != nil {
			return fmt.Errorf("error getting VM list: %w", err)
		}

		// Sort VM names for consistent output
		vmNames := make([]string, 0, len(allMeta))
		for name := range allMeta {
			vmNames = append(vmNames, name)
		}
		sort.Strings(vmNames)

		if len(vmNames) > 0 {
			for _, vmName := range vmNames {
				color.Cyan("i Cleaning VM: %s", vmName)
				if err := vmCleanCmd.RunE(cmd, []string{vmName}); err != nil {
					// Log the error but continue cleaning other VMs
					color.Yellow("! Warning: failed to clean VM %s: %v", vmName, err)
				}
			}
		} else {
			color.Yellow("No VMs to clean.")
		}

		if purge {
			color.Cyan("i Purging the entire application directory...")
			appDir := cfg.GetAppDir()
			if err := os.RemoveAll(appDir); err != nil {
				return fmt.Errorf("error removing application directory: %w", err)
			}
			color.Green("✔ Application directory purged successfully.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&purge, "purge", false, "Remove the entire ~/.provisioning-vm-lab directory")
}
