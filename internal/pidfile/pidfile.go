package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"pvmlab/internal/config"
	"strconv"
	"strings"
	"syscall"
)

func getPIDFilePath(cfg *config.Config, vmName string) string {
	return filepath.Join(cfg.GetAppDir(), "pids", vmName+".pid")
}

var IsRunning = func(cfg *config.Config, vmName string) (bool, error) {
	pid, err := Read(cfg, vmName)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, nil // Process not found
	}

	// Sending signal 0 to a process on Unix-like systems checks for its existence.
	err = process.Signal(syscall.Signal(0))
	return err == nil, nil
}

func Read(cfg *config.Config, vmName string) (int, error) {
	pidPath := getPIDFilePath(cfg, vmName)

	content, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in pidfile: %w", err)
	}

	return pid, nil
}