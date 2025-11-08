package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runCommand executes a command and streams output to stdout/stderr
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("  -> Running: %s %s\n", name, strings.Join(args, " "))
	return cmd.Run()
}

// dropToShell drops to a debug shell when an error occurs
func dropToShell() {
	fmt.Println("\n==> Installation failed, exiting...")
	os.Exit(1)
}
