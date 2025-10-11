
package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/pidfile"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// vmStopCmd represents the stop command
var vmStopCmd = &cobra.Command{
	Use:   "stop <vm-name>",
	Short: "Stops a VM",
	Long:  `Stops a VM. It first attempts a graceful shutdown, then resorts to force-stopping the process.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Stopping VM: %s", vmName)

		cfg, err := config.New()
		if err != nil {
			return err
		}
		appDir := cfg.GetAppDir()

		pid, err := pidfile.Read(cfg, vmName)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("VM '%s' is not running (no PID file found)", vmName)
			}
			return fmt.Errorf("error reading PID file: %w", err)
		}

		// 1. Attempt Graceful Shutdown via Monitor
		monitorPath := filepath.Join(appDir, "monitors", vmName+".sock")
		if _, err := os.Stat(monitorPath); err == nil {
			color.Cyan("i Attempting graceful shutdown...")
			conn, err := net.DialTimeout("unix", monitorPath, 1*time.Second)
			if err == nil {
				defer conn.Close()
				_, err := conn.Write([]byte("system_powerdown\n"))
				if err == nil {
					// Wait up to 10 seconds for the process to exit
					for i := 0; i < 10; i++ {
						if !isProcessRunning(pid) {
							color.Green("✔ Graceful shutdown successful.")
							cleanupFiles(vmName, appDir)
							return nil
						}
						time.Sleep(1 * time.Second)
					}
					color.Yellow("! VM did not shut down gracefully, proceeding to force stop.")
				}
			}
		}

		// 2. Forceful Shutdown (SIGTERM, then SIGKILL)
		if isProcessRunning(pid) {
			color.Cyan("i Sending SIGTERM to process %d", pid)
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				color.Yellow("! Failed to send SIGTERM: %v", err)
			}
			// Wait up to 5 seconds
			for i := 0; i < 5; i++ {
				if !isProcessRunning(pid) {
					color.Green("✔ VM stopped successfully.")
					cleanupFiles(vmName, appDir)
					return nil
				}
				time.Sleep(1 * time.Second)
			}

			color.Yellow("! Process did not respond to SIGTERM, sending SIGKILL...")
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				return fmt.Errorf("failed to send SIGKILL: %w", err)
			}
			time.Sleep(1 * time.Second) // Give SIGKILL a moment
		}

		if isProcessRunning(pid) {
			return fmt.Errorf("failed to stop VM '%s' (PID: %d)", vmName, pid)
		}

		color.Green("✔ VM '%s' stopped successfully.", vmName)
		cleanupFiles(vmName, appDir)
		return nil
	},
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false // Cannot find process, assume not running
	}
	// On Unix-like systems, sending signal 0 to a process checks if it exists.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func cleanupFiles(vmName string, appDir string) {
	pidPath := filepath.Join(appDir, "pids", vmName+".pid")
	monitorPath := filepath.Join(appDir, "monitors", vmName+".sock")

	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		color.Yellow("! Warning: could not remove pid file: %v", err)
	}
	if err := os.Remove(monitorPath); err != nil && !os.IsNotExist(err) {
		color.Yellow("! Warning: could not remove monitor socket: %v", err)
	}
}

func init() {
	vmCmd.AddCommand(vmStopCmd)
}
