package main

// CloudInitData holds the cloud-init configuration
type CloudInitData struct {
	MetaData      string
	UserData      string
	NetworkConfig string
}

// NetworkConfig holds network configuration parsed from kernel command line
type NetworkConfig struct {
	IP            string // "dhcp" or IP address (TODO: support static IP)
	MAC           string
	Netmask       string // TODO: support static netmask
	Gateway       string // TODO: support static gateway
	DNS           string // TODO: support static DNS servers
	InterfaceName string
}
