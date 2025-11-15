package util

import (
	"fmt"
	"strings"
)

// ParseSize converts a size string like "10G", "512M", "2048K" into bytes.
var ParseSize = func(sizeStr string) (int64, error) {
	var value int64
	var unit string

	// Try to parse with unit (e.g., "10G", "512M")
	n, err := fmt.Sscanf(sizeStr, "%d%s", &value, &unit)
	if err != nil || n != 2 {
		// If parsing with unit fails, try to parse as just a number (bytes)
		n, err = fmt.Sscanf(sizeStr, "%d", &value)
		if err != nil || n != 1 {
			return 0, fmt.Errorf("invalid size format '%s'. Expected format like '10G', '512M', or '2048'", sizeStr)
		}
		unit = "B" // Default to bytes if no unit is specified
	}

	unit = strings.ToUpper(unit)
	switch unit {
	case "K", "KB":
		value *= 1024
	case "M", "MB":
		value *= 1024 * 1024
	case "G", "GB":
		value *= 1024 * 1024 * 1024
	case "T", "TB":
		value *= 1024 * 1024 * 1024 * 1024
	case "", "B":
		// value is already in bytes
	default:
		return 0, fmt.Errorf("unknown size unit '%s' in '%s'", unit, sizeStr)
	}

	return value, nil
}
