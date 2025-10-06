
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"

	"github.com/spf13/cobra"
)

// vmCleanCmd represents the clean command
var vmCleanCmd = &cobra.Command{
	Use:   "clean <vm-name>",
	Short: "Stops a VM and removes all its associated files",
	Long:  `Stops a VM and removes all its associated files (disk, iso, pid, monitor, log).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]

		// First, stop the VM if it's running
		vmStopCmd.Run(cmd, args)

		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Remove the metadata file
		if err := metadata.Delete(vmName); err != nil {
			fmt.Printf("Warning: could not remove metadata file for %s: %v\n", vmName, err)
		}

		pathsToRemove := []string{
			filepath.Join(appDir, "vms", vmName+".qcow2"),
			filepath.Join(appDir, "configs", vmName+".iso"),
			filepath.Join(appDir, "logs", vmName+".log"),
			filepath.Join(appDir, "configs", "cloud-init", vmName),
		}

		for _, path := range pathsToRemove {
			if err := os.RemoveAll(path); err != nil {
				// Ignore errors if the path doesn't exist
				if !os.IsNotExist(err) {
					fmt.Println("Error removing path:", err)
				}
			}
		}

		fmt.Printf("VM '%s' files cleaned successfully.\n", vmName)
	},
}

func init() {
	vmCmd.AddCommand(vmCleanCmd)
}
