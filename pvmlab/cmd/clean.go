package cmd

import (
	"fmt"
	"os"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"
	"sort"

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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Cleaning up all VMs...")
		allMeta, err := metadata.GetAll()
		if err != nil {
			fmt.Println("Error getting VM list:", err)
			os.Exit(1)
		}

		// Sort VM names for consistent output
		vmNames := make([]string, 0, len(allMeta))
		for name := range allMeta {
			vmNames = append(vmNames, name)
		}
		sort.Strings(vmNames)

		if len(vmNames) > 0 {
			for _, vmName := range vmNames {
				fmt.Printf("Cleaning VM: %s\n", vmName)
				vmCleanCmd.Run(cmd, []string{vmName})
			}
		} else {
			fmt.Println("No VMs to clean.")
		}

		if purge {
			fmt.Println("Purging the entire application directory...")
			appDir, err := config.GetAppDir()
			if err != nil {
				fmt.Println("Error getting application directory:", err)
				os.Exit(1)
			}
			if err := os.RemoveAll(appDir); err != nil {
				fmt.Println("Error removing application directory:", err)
				os.Exit(1)
			}
			fmt.Println("Application directory purged successfully.")
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&purge, "purge", false, "Remove the entire ~/.provisioning-vm-lab directory")
}
