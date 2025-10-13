package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/pidfile"
	"pvmlab/internal/socketvmnet"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var purge bool

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Stops all VMs and removes all generated files",
	Long: `Cleans up all pvmlab generated files, including VMs, ISOs, and logs.
Use the --purge flag to remove the entire ~/.pvmlab directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("i Cleaning up pvmlab environment...")

		cfg, err := config.New()
		if err != nil {
			return err
		}

		appDir := cfg.GetAppDir()

		// Stop all VMs first
		allMeta, err := metadata.GetAll(cfg)
		if err != nil {
			return fmt.Errorf("error getting VM list: %w", err)
		}
		for vmName := range allMeta {
			color.Cyan("i Stopping VM: %s", vmName)
			// We can ignore errors here, as the VM might already be stopped.
			_ = stopVM(cfg, vmName)
		}

		// Stop the socket_vmnet service
		color.Cyan("i Stopping socket_vmnet service...")
		if err := socketvmnet.StopSocketVmnet(); err != nil {
			color.Yellow("! socket_vmnet service not running or could not be stopped: %v", err)
		}

		if purge {
			color.Yellow("! Purging entire pvmlab directory: %s", appDir)
			if err := os.RemoveAll(appDir); err != nil {
				return fmt.Errorf("error removing app directory: %w", err)
			}
		} else {
			// Just remove the contents of the subdirectories
			subDirs := []string{"vms", "pids", "monitors", "logs", "configs", "images", "docker_images"}
			for _, dir := range subDirs {
				fullPath := filepath.Join(appDir, dir)
				color.Cyan("i Cleaning directory: %s", fullPath)
				if err := os.RemoveAll(fullPath); err != nil {
					// Log error but continue
					color.Red("! Error cleaning directory %s: %v", fullPath, err)
				}
			}
		}

		color.Green("✔ Cleanup complete.")
		return nil
	},
}

func stopVM(cfg *config.Config, vmName string) error {
	running, err := pidfile.IsRunning(cfg, vmName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Not running
		}
		return fmt.Errorf("error reading PID file for %s: %w", vmName, err)
	}
	if !running {
		color.Yellow("! VM %s is not running.", vmName)
		// Clean up stale PID file
		pidPath := filepath.Join(cfg.GetAppDir(), "pids", vmName+".pid")
		if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error deleting stale PID file: %w", err)
		}
		return nil
	}

	monitorPath := filepath.Join(cfg.GetAppDir(), "monitors", vmName+".sock")
	cmd := exec.Command("socat", "-", monitorPath)
	cmd.Stdin = strings.NewReader("system_powerdown\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error sending powerdown command to %s: %w", vmName, err)
	}

	// Wait for the process to exit
	for i := 0; i < 10; i++ {
		running, err := pidfile.IsRunning(cfg, vmName)
		if err != nil {
			return fmt.Errorf("error checking if %s is running: %w", vmName, err)
		}
		if !running {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	running, err = pidfile.IsRunning(cfg, vmName)
	if err != nil {
		return fmt.Errorf("error checking if %s is running: %w", vmName, err)
	}
	if running {
		return fmt.Errorf("timed out waiting for VM %s to stop", vmName)
	}

	pidPath := filepath.Join(cfg.GetAppDir(), "pids", vmName+".pid")
	return os.Remove(pidPath)
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&purge, "purge", false, "Remove the entire ~/.pvmlab directory")
}
