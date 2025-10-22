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

func copyAndMakeExecutable(src, dst string) error {
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

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
