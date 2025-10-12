package waiter

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/hpcloud/tail"
)

var (
	// execCommand is a variable to allow mocking of exec.Command in tests
	execCommand = exec.Command
)

// ForPort polls a TCP port until it becomes available or a timeout is reached.
func ForPort(host string, port int, timeout time.Duration) error {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Waiting for SSH port %s to become available...", address)
	s.Start()
	defer s.Stop()

	timeoutChan := time.After(timeout)
	for {
		select {
		case <-timeoutChan:
			s.FinalMSG = color.RedString("✖ Timed out waiting for port %s\n", address)
			return fmt.Errorf("timed out waiting for port %s", address)
		default:
			conn, err := net.DialTimeout("tcp", address, 1*time.Second)
			if err == nil {
				conn.Close()
				s.FinalMSG = color.GreenString("✔ Port %s is now available.\n", address)
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// ForMessage tails a log file and waits for a specific message to appear.
func ForMessage(logPath, message string, timeout time.Duration) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Waiting for message '%s' in log file...", message)
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
		return fmt.Errorf("error tailing log file: %w", err)
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
				s.FinalMSG = color.GreenString("✔ Found message '%s' in log file.\n", message)
				return nil
			}
		case <-timeoutChan:
			s.FinalMSG = color.RedString("✖ Timed out waiting for message '%s'.\n", message)
			return fmt.Errorf("timed out waiting for message in log file")
		}
	}
}

// ForCloudInitTarget polls the VM via SSH until the cloud-init.target is active.
func ForCloudInitTarget(sshPort int, sshKeyPath string, timeout time.Duration) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Waiting for cloud-init.target to become active (via ssh on port %d)...", sshPort)
	s.Start()
	defer s.Stop()

	sshPortStr := fmt.Sprintf("%d", sshPort)
	command := "systemctl show cloud-init.target --property ActiveState"

	timeoutChan := time.After(timeout)
	for {
		select {
		case <-timeoutChan:
			s.FinalMSG = color.RedString("✖ Timed out waiting for cloud-init to complete.\n")
			return fmt.Errorf("timed out waiting for cloud-init.target to become active")
		default:
			cmd := execCommand("ssh", "-i", sshKeyPath, "-p", sshPortStr, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost", command)
			output, err := cmd.CombinedOutput()
			if err == nil && strings.Contains(string(output), "ActiveState=active") {
				s.FinalMSG = color.GreenString("✔ Cloud-init completed successfully.\n")
				return nil
			}
			// Don't spam the log with connection errors, just wait and retry.
			time.Sleep(2 * time.Second)
		}
	}
}
