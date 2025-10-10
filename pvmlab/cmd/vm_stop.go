
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

	"github.com/spf13/cobra"
)

// vmStopCmd represents the stop command
var vmStopCmd = &cobra.Command{
	Use:   "stop <vm-name>",
	Short: "Stops a VM",
	Long:  `Stops a VM. It first attempts a graceful shutdown, then resorts to force-stopping the process.`,
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]
		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		pid, err := pidfile.Read(vmName)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("VM '%s' is not running (no PID file found).\n", vmName)
				return
			}
			fmt.Println("Error reading PID file:", err)
			os.Exit(1)
		}

		// 1. Attempt Graceful Shutdown via Monitor
		monitorPath := filepath.Join(appDir, "monitors", vmName+".sock")
		if _, err := os.Stat(monitorPath); err == nil {
			fmt.Println("Attempting graceful shutdown...")
			conn, err := net.DialTimeout("unix", monitorPath, 1*time.Second)
			if err == nil {
				defer conn.Close()
				_, err := conn.Write([]byte("system_powerdown\n"))
				if err == nil {
					// Wait up to 10 seconds for the process to exit
					for i := 0; i < 10; i++ {
						if !isProcessRunning(pid) {
							fmt.Println("Graceful shutdown successful.")
							cleanupFiles(vmName, appDir)
							return
						}
						time.Sleep(1 * time.Second)
					}
					fmt.Println("VM did not shut down gracefully, proceeding to force stop.")
				}
			}
		}

		// 2. Forceful Shutdown (SIGTERM, then SIGKILL)
		if isProcessRunning(pid) {
			fmt.Println("Sending SIGTERM to process", pid)
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				fmt.Println("Failed to send SIGTERM:", err)
			}
			// Wait up to 5 seconds
			for i := 0; i < 5; i++ {
				if !isProcessRunning(pid) {
					fmt.Println("VM stopped successfully.")
					cleanupFiles(vmName, appDir)
					return
				}
				time.Sleep(1 * time.Second)
			}

			fmt.Println("Process did not respond to SIGTERM, sending SIGKILL...")
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				fmt.Println("Failed to send SIGKILL:", err)
				os.Exit(1)
			}
			time.Sleep(1 * time.Second) // Give SIGKILL a moment
		}

		if isProcessRunning(pid) {
			fmt.Printf("Error: Failed to stop VM '%s' (PID: %d).\n", vmName, pid)
			os.Exit(1)
		}

		fmt.Printf("VM '%s' stopped successfully.\n", vmName)
		cleanupFiles(vmName, appDir)
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
		fmt.Println("Warning: could not remove pid file:", err)
	}
	if err := os.Remove(monitorPath); err != nil && !os.IsNotExist(err) {
		fmt.Println("Warning: could not remove monitor socket:", err)
	}
}

func init() {
	vmCmd.AddCommand(vmStopCmd)
}
