package integration

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func runCmdWithLiveOutput(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)

	err := cmd.Run()
	fullOutput := outBuf.String() + errBuf.String()

	if err != nil {
		return fullOutput, fmt.Errorf("command failed with error: %w\n--- Output ---\n%s", err, fullOutput)
	}

	return fullOutput, nil
}

func runCmdOrFail(t *testing.T, command string, args ...string) string {
	t.Helper()
	output, err := runCmdWithLiveOutput(command, args...)
	if err != nil {
		t.Fatalf("Command `%s %s` failed.\nError: %v\nOutput:\n%s", command, strings.Join(args, " "), err, output)
	}
	return output
}
