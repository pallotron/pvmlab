package main

import (
	"installer/log"
	"os"
	"os/exec"
	"strings"
)

// runCommand executes a command and streams output to stdout/stderr
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Command(name, args...)
	return cmd.Run()
}

// dropToShell drops to a debug shell when an error occurs
func dropToShell() {
	log.Step("Installation failed, exiting...")
	os.Exit(1)
}

// getKernelCmdline reads and returns the kernel command line from /proc/cmdline.
func getKernelCmdline() (string, error) {
	data, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}