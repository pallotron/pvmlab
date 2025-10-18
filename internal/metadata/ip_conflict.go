package metadata

import (
	"fmt"
	"net"
	"pvmlab/internal/config"
)

func CheckForDuplicateIPs(cfg *config.Config, ip, ipv6 string) error {
	allVMs, err := GetAll(cfg)
	if err != nil {
		return fmt.Errorf("failed to get all VMs: %w", err)
	}

	for vmName, meta := range allVMs {
		if meta.IP != "" && ip != "" {
			newIP, _, err := net.ParseCIDR(ip)
			if err != nil {
				return fmt.Errorf("invalid new ip: %w", err)
			}
			existingIP, _, err := net.ParseCIDR(meta.IP)
			if err != nil {
				return fmt.Errorf("invalid existing ip for %s: %w", vmName, err)
			}
			if newIP.Equal(existingIP) {
				return fmt.Errorf("IP address %s is already in use by VM %s", ip, vmName)
			}
		}
		if meta.IPv6 != "" && ipv6 != "" {
			newIP, _, err := net.ParseCIDR(ipv6)
			if err != nil {
				return fmt.Errorf("invalid new ipv6: %w", err)
			}
			existingIP, _, err := net.ParseCIDR(meta.IPv6)
			if err != nil {
				return fmt.Errorf("invalid existing ipv6 for %s: %w", vmName, err)
			}
			if newIP.Equal(existingIP) {
				return fmt.Errorf("IPv6 address %s is already in use by VM %s", ipv6, vmName)
			}
		}
	}

	return nil
}
