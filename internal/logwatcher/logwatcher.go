package logwatcher

import (
	"fmt"
	"io"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/hpcloud/tail"
)

// WaitForMessage tails the console log for a VM and waits for a specific message.
func WaitForMessage(cfg *config.Config, vmName, message string, timeout time.Duration) error {
	appDir := cfg.GetAppDir()
	logPath := filepath.Join(appDir, "logs", vmName+".log")

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Waiting for cloud-init to complete... (watching log file for '%s')", message)
	s.Start()
	defer s.Stop()

	t, err := tail.TailFile(logPath, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
		Logger:   tail.DiscardingLogger,
	})
	if err != nil {
		s.FinalMSG = color.RedString("✖ Error tailing log file.\n")
		return err
	}
	defer t.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case line := <-t.Lines:
			if line.Err != nil {
				s.FinalMSG = color.RedString("✖ Error reading log file.\n")
				return fmt.Errorf("error reading log file: %w", line.Err)
			}
			if strings.Contains(strings.TrimSpace(strings.ToLower(line.Text)), strings.ToLower(message)) {
				s.FinalMSG = color.GreenString("✔ Cloud-init completed successfully.\n")
				return nil
			}
		case <-timeoutChan:
			s.FinalMSG = color.RedString("✖ Timed out waiting for cloud-init.\n")
			return fmt.Errorf("timed out waiting for message in log file")
		}
	}
}
