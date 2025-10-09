package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"strconv"
	"strings"
	"syscall"
)

func getPIDFilePath(vmName string) (string, error) {
	appDir, err := config.GetAppDirFunc()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "pids", vmName+".pid"), nil
}

func IsRunning(vmName string) (bool, error) {
	pid, err := Read(vmName)
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

func Read(vmName string) (int, error) {
	pidPath, err := getPIDFilePath(vmName)
	if err != nil {
		return 0, err
	}

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