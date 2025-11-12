package main

// InstallerConfig is the configuration provided to the installer running in the initrd.

type InstallerConfig struct {
	CloudInitURL    string `json:"cloud_init_url"`
	Distro          string `json:"distro"`
	Arch            string `json:"arch"`
	RootfsURL       string `json:"rootfs_url"`
	KmodsURL        string `json:"kmods_url"`
	KernelURL       string `json:"kernel_url"`
	RebootOnSuccess bool   `json:"reboot_on_success"`
}
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
	ConfigURL     string // URL to fetch the installer config JSON
}
