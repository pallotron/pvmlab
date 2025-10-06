package brew

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsSocketVmnetRunning checks if the socket_vmnet service is running.
func IsSocketVmnetRunning() (bool, error) {
	out, err := exec.Command("sudo", "brew", "services", "info", "socket_vmnet").Output()
	if err != nil {
		// Brew services info can return a non-zero exit code if the service is not running,
		// so we check the output even if there's an ExitError.
		if _, ok := err.(*exec.ExitError); !ok {
			return false, fmt.Errorf("error checking socket_vmnet service status: %w", err)
		}
	}

	output := string(out)
	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		if strings.HasPrefix(line, "Running:") {
			if strings.Contains(line, "✔") || strings.Contains(line, "true") {
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	return false, fmt.Errorf("could not determine socket_vmnet service status from output")
}

// StartSocketVmnet starts the socket_vmnet service.
func StartSocketVmnet() error {
	fmt.Println("Starting socket_vmnet service... (this may require sudo password)")
	cmdRun := exec.Command("sudo", "brew", "services", "start", "socket_vmnet")
	output, err := cmdRun.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s\n%s", cmdRun.String(), string(output))
	}
	fmt.Println("socket_vmnet service started successfully.")
	return nil
}
