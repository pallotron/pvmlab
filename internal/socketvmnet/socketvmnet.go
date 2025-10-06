package socketvmnet

import (
	"os/exec"
	"strings"
)

// GetSocketPath returns the path to the socket_vmnet socket.
func GetSocketPath() (string, error) {
	cmd := exec.Command("brew", "--prefix")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	prefix := strings.TrimSpace(string(out))
	return prefix + "/var/run/socket_vmnet", nil
}
