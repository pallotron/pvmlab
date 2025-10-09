package socketvmnet

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const (
	// ServiceName is the name of the socket_vmnet service.
	ServiceName = "io.github.pallotron.pvmlab.socket_vmnet"
)

// GetSocketPath returns the path to the socket_vmnet socket.
func GetSocketPath() (string, error) {
	// TODO: when https://github.com/lima-vm/socket_vmnet/pull/140 is merged
	// we can use the brew socket_vmnet path
	// cmd := exec.Command("brew", "--prefix")
	// out, err := cmd.Output()
	// if err != nil {
	// 	return "", err
	// }
	// prefix := strings.TrimSpace(string(out))
	// return prefix + "/var/run/socket_vmnet", nil
	return "/var/run/vmlab.socket_vmnet", nil
}

var execCommand = exec.Command

// IsSocketVmnetRunning checks if the socket_vmnet service is running.
func IsSocketVmnetRunning() (bool, error) {
	cmd := execCommand("sudo", "launchctl", "list", ServiceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		// launchctl list returns a non-zero exit code if the service is not found.
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("error checking %s service status: %w", ServiceName, err)
	}

	// If the service is running, the output will contain a PID.
	return strings.Contains(out.String(), "PID"), nil
}

func StartSocketVmnet() error {
	cmd := execCommand("sudo", "launchctl", "start", ServiceName)
	return cmd.Run()
}

func StopSocketVmnet() error {
	cmd := execCommand("sudo", "launchctl", "stop", ServiceName)
	return cmd.Run()
}

// CheckSocketVmnet checks if the socket_vmnet service is running and warns the user if it is not.
func CheckSocketVmnet() error {
	running, err := IsSocketVmnetRunning()
	if err != nil {
		return err
	}
	if !running {
		return fmt.Errorf("%s service is not running", ServiceName)
	}
	return nil
}
