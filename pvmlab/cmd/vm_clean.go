package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// vmCleanCmd represents the clean command
var vmCleanCmd = &cobra.Command{
	Use:   "clean <vm-name>",
	Short: "Stops a VM and removes all its associated files",
	Long:  `Stops a VM and removes all its associated files (disk, iso, pid, monitor, log).`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Cleaning VM: %s", vmName)

		// First, stop the VM if it's running
		if err := vmStopCmd.RunE(cmd, args); err != nil {
			// Ignore "not running" errors, as the goal is to clean up.
			if !strings.Contains(err.Error(), "is not running") {
				return fmt.Errorf("error stopping VM '%s': %w", vmName, err)
			}
		}

		cfg, err := config.New()
		if err != nil {
			return err
		}
		appDir := cfg.GetAppDir()

		// Remove the metadata file
		if err := metadata.Delete(cfg, vmName); err != nil {
			color.Yellow("! Warning: could not remove metadata file for %s: %v", vmName, err)
		}

		filesToRemove := []string{
			filepath.Join(appDir, "vms", vmName+".qcow2"),
			filepath.Join(appDir, "configs", "cloud-init", vmName+".iso"),
			filepath.Join(appDir, "configs", "cloud-init", vmName),
			filepath.Join(appDir, "logs", vmName+".log"),
			filepath.Join(appDir, "pids", vmName+".pid"),
			filepath.Join(appDir, "monitors", vmName+".sock"),
		}
		for _, path := range filesToRemove {
			if err := os.RemoveAll(path); err != nil {
				// Ignore errors if the path doesn't exist
				if !os.IsNotExist(err) {
					color.Yellow("! Warning: could not remove path %s: %v", path, err)
				}
			}
		}

		color.Green("✔ VM '%s' files cleaned successfully.", vmName)
		return nil
	},
}

func init() {
	vmCmd.AddCommand(vmCleanCmd)
}
