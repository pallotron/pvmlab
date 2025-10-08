package socketvmnet

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
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

// IsSocketVmnetRunning checks if the socket_vmnet service is running.
func IsSocketVmnetRunning() (bool, error) {
	cmd := exec.Command("sudo", "launchctl", "list", "io.github.pallotron.pvmlab.socket_vmnet")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		// launchctl list returns a non-zero exit code if the service is not found.
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("error checking socket_vmnet service status: %w", err)
	}

	// If the service is running, the output will contain a PID.
	return strings.Contains(out.String(), "PID"), nil
}

// CheckSocketVmnet checks if the socket_vmnet service is running and warns the user if it is not.
func CheckSocketVmnet() error {
	running, err := IsSocketVmnetRunning()
	if err != nil {
		return err
	}
	if !running {
		fmt.Println("Warning: socket_vmnet service is not running.")
		fmt.Println("For more details, see the project's README.md on github or the project Makefile")

	}
	return nil
}
