package logwatcher

import (
	"fmt"
	"io"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"strings"
	"time"

	"github.com/hpcloud/tail"
)

// WaitForMessage tails the console log for a VM and waits for a specific message.
func WaitForMessage(vmName, message string, timeout time.Duration) error {
	appDir, err := config.GetAppDirFunc()
	if err != nil {
		return err
	}
	logPath := filepath.Join(appDir, "logs", vmName+".log")

	fmt.Printf("Waiting for cloud-init to complete... (watching log file for '%s')\n", message)

	t, err := tail.TailFile(logPath, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	})
	if err != nil {
		return err
	}
	defer t.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case line := <-t.Lines:
			if line.Err != nil {
				return fmt.Errorf("error reading log file: %w", line.Err)
			}
			if strings.Contains(strings.TrimSpace(strings.ToLower(line.Text)), strings.ToLower(message)) {
				fmt.Printf("Message '%s' found in log file.\n", message)
				return nil
			}
		case <-timeoutChan:
			return fmt.Errorf("timed out waiting for message in log file")
		}
	}
}
