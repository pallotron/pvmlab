package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// setupNetworking configures the network interface
func setupNetworking() (*NetworkConfig, error) {
	fmt.Println("  -> Parsing network configuration from kernel command line...")

	netConfig, err := parseNetworkConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse network config: %w", err)
	}

	fmt.Printf("  -> Network mode: %s\n", netConfig.IP)
	if netConfig.MAC != "" {
		fmt.Printf("  -> Target MAC: %s\n", netConfig.MAC)
	}

	// Find the network interface
	fmt.Println("  -> Detecting network interfaces...")
	iface, err := findNetworkInterface(netConfig.MAC)
	if err != nil {
		return nil, fmt.Errorf("failed to find network interface: %w", err)
	}

	netConfig.InterfaceName = iface
	fmt.Printf("  -> Using interface: %s\n", iface)

	// Bring interface up
	fmt.Println("  -> Bringing interface up...")
	if err := runCommand("ip", "link", "set", iface, "up"); err != nil {
		return nil, fmt.Errorf("failed to bring interface up: %w", err)
	}

	// Configure network based on mode
	if netConfig.IP == "dhcp" {
		if err := setupDHCP(iface); err != nil {
			return nil, fmt.Errorf("failed to setup DHCP: %w", err)
		}
	} else {
		// TODO: Implement static IP configuration
		// This would parse ip=<IP>:<gateway>:<netmask>:<hostname>:<interface>:<dns>
		// or similar format and configure the interface accordingly
		return nil, fmt.Errorf("static IP configuration not yet implemented (TODO)")
	}

	// Wait a bit for network to be ready
	fmt.Println("  -> Waiting for network to be ready...")
	time.Sleep(2 * time.Second)

	// Verify network is working
	fmt.Println("  -> Verifying network connectivity...")
	if err := runCommand("ip", "addr", "show", iface); err != nil {
		fmt.Printf("  -> Warning: failed to show interface details: %v\n", err)
	}

	return netConfig, nil
}

// parseNetworkConfig reads /proc/cmdline and extracts network parameters
func parseNetworkConfig() (*NetworkConfig, error) {
	data, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/cmdline: %w", err)
	}

	cmdline := string(data)
	config := &NetworkConfig{
		IP: "dhcp", // default to DHCP
	}

	// Parse kernel command line arguments
	for _, arg := range strings.Fields(cmdline) {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "ip":
			config.IP = value
			// TODO: Parse more complex ip= formats like:
			// ip=<client-ip>:<server-ip>:<gw-ip>:<netmask>:<hostname>:<device>:<autoconf>:<dns0-ip>:<dns1-ip>
		case "installer_mac":
			config.MAC = value
		case "config_url":
			config.ConfigURL = value
		// TODO: Add support for additional network parameters:
		// case "netmask":
		//     config.Netmask = value
		// case "gateway":
		//     config.Gateway = value
		// case "dns":
		//     config.DNS = value
		}
	}

	return config, nil
}

// findNetworkInterface finds the network interface by MAC address or returns the first available one
func findNetworkInterface(targetMAC string) (string, error) {
	// If no MAC specified, find the first non-loopback interface
	if targetMAC == "" {
		return findFirstInterface()
	}

	// Normalize MAC address (remove colons, make lowercase)
	targetMAC = strings.ToLower(strings.ReplaceAll(targetMAC, ":", ""))

	// Look for interface with matching MAC
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return "", fmt.Errorf("failed to read /sys/class/net: %w", err)
	}

	for _, entry := range entries {
		ifname := entry.Name()
		if ifname == "lo" {
			continue
		}

		// Read MAC address
		macPath := fmt.Sprintf("/sys/class/net/%s/address", ifname)
		macData, err := os.ReadFile(macPath)
		if err != nil {
			continue
		}

		ifMAC := strings.TrimSpace(string(macData))
		ifMAC = strings.ToLower(strings.ReplaceAll(ifMAC, ":", ""))

		if ifMAC == targetMAC {
			return ifname, nil
		}
	}

	return "", fmt.Errorf("no interface found with MAC %s", targetMAC)
}

// findFirstInterface returns the first non-loopback network interface
func findFirstInterface() (string, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return "", fmt.Errorf("failed to read /sys/class/net: %w", err)
	}

	for _, entry := range entries {
		ifname := entry.Name()
		if ifname != "lo" {
			return ifname, nil
		}
	}

	return "", fmt.Errorf("no network interface found")
}

// setupDHCP configures the interface using DHCP
func setupDHCP(iface string) error {
	fmt.Printf("  -> Running DHCP on %s...\n", iface)

	// Try using dhclient first
	if err := runCommand("dhclient", "-v", iface); err == nil {
		fmt.Println("  -> DHCP configuration successful (dhclient)")
		return nil
	}

	// Fallback to udhcpc (common in minimal environments)
	if err := runCommand("udhcpc", "-i", iface, "-n", "-q"); err == nil {
		fmt.Println("  -> DHCP configuration successful (udhcpc)")
		return nil
	}

	// Fallback to dhcpcd
	if err := runCommand("dhcpcd", iface); err == nil {
		fmt.Println("  -> DHCP configuration successful (dhcpcd)")
		return nil
	}

	return fmt.Errorf("no DHCP client available (tried dhclient, udhcpc, dhcpcd)")
}

// parseDNSServers reads /etc/resolv.conf and returns DNS servers
func parseDNSServers() ([]string, error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				servers = append(servers, parts[1])
			}
		}
	}

	return servers, scanner.Err()
}
