package qemu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

var execCommand = exec.Command

// GetImageVirtualSize executes `qemu-img info` to get the virtual size of an image in bytes.
var GetImageVirtualSize = func(imagePath string) (int64, error) {
	cmd := execCommand("qemu-img", "info", "--output=json", imagePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to get image info for %s: %w", imagePath, err)
	}

	var info struct {
		VirtualSize int64 `json:"virtual-size"`
	}
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return 0, fmt.Errorf("failed to parse qemu-img info output: %w", err)
	}

	return info.VirtualSize, nil
}
