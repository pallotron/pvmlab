package runner

import (
	"fmt"
	"os/exec"
)

// Run executes a command and returns an error with the combined output if it fails.
func Run(cmd *exec.Cmd) error {
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s\n%s", cmd.String(), string(output))
	}
	return nil
}
