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

	var newIP net.IP
	if ip != "" {
		var err error
		newIP, _, err = net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("invalid new ip/cidr '%s': %w", ip, err)
		}
	}

	var newIPv6 net.IP
	if ipv6 != "" {
		var err error
		newIPv6, _, err = net.ParseCIDR(ipv6)
		if err != nil {
			return fmt.Errorf("invalid new ipv6/cidr '%s': %w", ipv6, err)
		}
	}

	for vmName, meta := range allVMs {
		if meta.IP != "" && newIP != nil {
			existingIP := net.ParseIP(meta.IP)
			if existingIP == nil {
				return fmt.Errorf("invalid IP address format for existing VM '%s': %s", vmName, meta.IP)
			}
			if newIP.Equal(existingIP) {
				return fmt.Errorf("IP address from %s is already in use by VM '%s'", ip, vmName)
			}
		}
		if meta.IPv6 != "" && newIPv6 != nil {
			existingIPv6 := net.ParseIP(meta.IPv6)
			if existingIPv6 == nil {
				return fmt.Errorf("invalid IPv6 address format for existing VM '%s': %s", vmName, meta.IPv6)
			}
			if newIPv6.Equal(existingIPv6) {
				return fmt.Errorf("IPv6 address from %s is already in use by VM '%s'", ipv6, vmName)
			}
		}
	}

	return nil
}
