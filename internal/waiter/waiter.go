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

// pollCloudInit is a helper that contains the core polling logic.
func pollCloudInit(s *spinner.Spinner, sshArgs []string, timeout time.Duration, timeoutMsg string) error {
	s.Start()
	defer s.Stop()

	command := "systemctl show cloud-init.target --property ActiveState"

	timeoutChan := time.After(timeout)
	for {
		select {
		case <-timeoutChan:
			s.FinalMSG = color.RedString(timeoutMsg)
			return fmt.Errorf("%s", strings.TrimSpace(timeoutMsg))
		default:
			// We create the command inside the loop because an exec.Cmd cannot be reused.
			cmdArgs := append(sshArgs, command)
			cmd := execCommand("ssh", cmdArgs...)
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

// ForCloudInitProvisioner polls the provisioner VM via SSH until the cloud-init.target is active.
func ForCloudInitProvisioner(sshPort int, sshKeyPath string, timeout time.Duration) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Waiting for cloud-init.target to become active (via ssh on port %d)...", sshPort)

	sshPortStr := fmt.Sprintf("%d", sshPort)
	sshArgs := []string{
		"-i", sshKeyPath,
		"-p", sshPortStr,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"ubuntu@localhost",
	}
	timeoutMsg := "✖ Timed out waiting for cloud-init to complete.\n"

	return pollCloudInit(s, sshArgs, timeout, timeoutMsg)
}

// ForCloudInitTarget polls a target VM via SSH through a provisioner jump host.
func ForCloudInitTarget(provisionerPort int, targetIP, sshKeyPath string, timeout time.Duration) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Waiting for cloud-init.target on %s (via provisioner on port %d)...", targetIP, provisionerPort)

	provisionerPortStr := fmt.Sprintf("%d", provisionerPort)
	targetConnect := fmt.Sprintf("ubuntu@%s", targetIP)
	proxyCommand := fmt.Sprintf("ssh -i %s -p %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %%h:%%p ubuntu@localhost", sshKeyPath, provisionerPortStr)

	sshArgs := []string{
		"-i", sshKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", fmt.Sprintf("ProxyCommand=%s", proxyCommand),
		targetConnect,
	}
	timeoutMsg := fmt.Sprintf("✖ Timed out waiting for cloud-init to complete on %s.\n", targetIP)

	return pollCloudInit(s, sshArgs, timeout, timeoutMsg)
}
