package cloudinit

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"
	"strings"

	"gopkg.in/yaml.v3"
)

func marshal(in any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(in)
	return buf.Bytes(), err
}

// MetaData structs
type MetaData struct {
	InstanceID        string   `yaml:"instance-id"`
	LocalHostname     string   `yaml:"local-hostname"`
	PublicKeys        []string `yaml:"public-keys"`
	PxeBootStackTar   string   `yaml:"pxe_boot_stack_tar"`
	PxeBootStackName  string   `yaml:"pxe_boot_stack_name"`
	PxeBootStackImage string   `yaml:"pxe_boot_stack_image"`
	ProvisionerIP     string   `yaml:"provisioner_ip,omitempty"`
	ProvisionerIPv6   string   `yaml:"provisioner_ipv6,omitempty"`
	DhcpRangeStart    string   `yaml:"dhcp_range_start,omitempty"`
	DhcpRangeEnd      string   `yaml:"dhcp_range_end,omitempty"`
	DhcpRangeV6Start  string   `yaml:"dhcp_range_v6_start,omitempty"`
	DhcpRangeV6End    string   `yaml:"dhcp_range_v6_end,omitempty"`
	IPv6Subnet        string   `yaml:"ipv6_subnet,omitempty"`
}

// UserData structs
type SSHKeys string

func (s SSHKeys) MarshalYAML() (any, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: string(s),
		Style: yaml.LiteralStyle,
	}, nil
}

type User struct {
	Name              string  `yaml:"name"`
	Sudo              string  `yaml:"sudo"`
	Groups            string  `yaml:"groups"`
	Shell             string  `yaml:"shell"`
	SSHAuthorizedKeys SSHKeys `yaml:"ssh_authorized_keys"`
}

type ChPasswdUser struct {
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
	Type     string `yaml:"type"`
}

type ChPasswd struct {
	Users  []ChPasswdUser `yaml:"users"`
	Expire bool           `yaml:"expire"`
}

type WriteFile struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions"`
	Content     string `yaml:"content"`
}

type Mount []string

func (m Mount) MarshalYAML() (any, error) {
	node := yaml.Node{
		Kind:  yaml.SequenceNode,
		Style: yaml.FlowStyle,
	}
	for _, s := range m {
		node.Content = append(node.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s,
		})
	}
	return &node, nil
}

type RunCmd []string

func (r RunCmd) MarshalYAML() (any, error) {
	node := yaml.Node{
		Kind: yaml.SequenceNode,
	}
	for _, s := range r {
		node.Content = append(node.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s,
			Style: yaml.SingleQuotedStyle,
		})
	}
	return &node, nil
}

type UserData struct {
	SSHpwauth  bool        `yaml:"ssh_pwauth"`
	Users      []User      `yaml:"users"`
	ChPasswd   ChPasswd    `yaml:"chpasswd"`
	WriteFiles []WriteFile `yaml:"write_files"`
	RunCmd     RunCmd      `yaml:"runcmd"`
	Mounts     []Mount     `yaml:"mounts,omitempty"`
}

// NetworkConfig structs
type NetplanNameservers struct {
	Addresses []string `yaml:"addresses"`
	Search    []string `yaml:"search"`
}

type NetplanMatch struct {
	MacAddress string `yaml:"macaddress"`
}

type NetplanEthernet struct {
	Match       *NetplanMatch       `yaml:"match,omitempty"`
	DHCP4       bool                `yaml:"dhcp4"`
	DHCP6       bool                `yaml:"dhcp6,omitempty"`
	Nameservers *NetplanNameservers `yaml:"nameservers,omitempty"`
	Addresses   []string            `yaml:"addresses,omitempty"`
}

type NetplanConfig struct {
	Version   int                        `yaml:"version"`
	Ethernets map[string]NetplanEthernet `yaml:"ethernets"`
}

func buildProvisionerMetaData(
	sshKey, tar, image, provisionerIP, dhcpStart, dhcpEnd,
	provisionerIpV6, dhcpV6Start, dhcpV6End, ipv6Subnet string,
) *MetaData {
	var pxebootStackName string
	if tar != "" {
		pxebootStackName = strings.TrimSuffix(tar, filepath.Ext(tar))
	} else {
		pxebootStackName, _ = config.GetPxeBootStackImageName()
	}
	return &MetaData{
		InstanceID:        "iid-cloudimg-provisioner",
		LocalHostname:     "provisioner",
		PublicKeys:        []string{sshKey},
		PxeBootStackTar:   tar,
		PxeBootStackName:  pxebootStackName,
		PxeBootStackImage: image,
		ProvisionerIP:     provisionerIP,
		ProvisionerIPv6:   provisionerIpV6,
		DhcpRangeStart:    dhcpStart,
		DhcpRangeEnd:      dhcpEnd,
		DhcpRangeV6Start:  dhcpV6Start,
		DhcpRangeV6End:    dhcpV6End,
		IPv6Subnet:        ipv6Subnet,
	}
}

func buildTargetMetaData(vmName, sshKey string) *MetaData {
	return &MetaData{
		InstanceID:    fmt.Sprintf("iid-cloudimg-%s", vmName),
		LocalHostname: vmName,
		PublicKeys:    []string{sshKey},
	}
}

//go:embed assets/pxeboot_stack_reload.sh
var pxeReloadScript string

//go:embed assets/pxeboot.service
var pxeBootService string

//go:embed assets/radvd.conf
var radvdConf string

//go:embed assets/dhcpv6_duid_llt.conf
var dhcpv6DuidLltConf string

func buildProvisionerUserData(hasIPv6 bool) *UserData {
	writeFiles := []WriteFile{
		{Path: "/usr/local/bin/pxeboot_stack_reload.sh", Permissions: "0755", Content: pxeReloadScript},
		{Path: "/etc/systemd/system/pxeboot.service", Permissions: "0644", Content: pxeBootService},
	}

	if hasIPv6 {
		writeFiles = append(writeFiles, WriteFile{
			Path:        "/etc/radvd.conf",
			Permissions: "0644",
			Content:     radvdConf,
		})
	}

	userData := &UserData{
		SSHpwauth: true,
		Users: []User{
			{
				Name:              "ubuntu",
				Sudo:              "ALL=(ALL) NOPASSWD:ALL",
				Groups:            "sudo",
				Shell:             "/bin/bash",
				SSHAuthorizedKeys: SSHKeys("{{ ds.meta_data.public_keys | join('\\n') }}"),
			},
		},
		ChPasswd: ChPasswd{
			Users: []ChPasswdUser{
				{Name: "ubuntu", Password: "pass", Type: "text"},
				{Name: "root", Password: "pass", Type: "text"},
			},
			Expire: false,
		},
		WriteFiles: writeFiles,
		RunCmd: RunCmd{
			`sed -i "/net.ipv4.ip_forward/d" /etc/sysctl.conf`,
			`echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf`,
			`sed -i "/net.ipv6.conf.all.forwarding/d" /etc/sysctl.conf`,
			`echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf`,
			`sysctl -p`,
			`mkdir -p /mnt/host/docker_images`,
			`mkdir -p /mnt/host/vms`,
			`mkdir -p /mnt/host/images`,
			`systemctl daemon-reload`,
			`systemctl enable --now pxeboot.service`,
			`echo "iptables-persistent iptables-persistent/autosave_v4 boolean true" | debconf-set-selections`,
			`echo "iptables-persistent iptables-persistent/autosave_v6 boolean true" | debconf-set-selections`,
			`iptables -t nat -A POSTROUTING -o enp0s1 -j MASQUERADE`,
			`iptables -A FORWARD -i enp0s2 -o enp0s1 -j ACCEPT`,
			`iptables -A FORWARD -i enp0s1 -o enp0s2 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT`,
			`ip6tables -t nat -A POSTROUTING -o enp0s1 -j MASQUERADE`,
			`ip6tables -A FORWARD -i enp0s2 -o enp0s1 -j ACCEPT`,
			`ip6tables -A FORWARD -i enp0s1 -o enp0s2 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT`,
			`iptables-save > /etc/iptables/rules.v4`,
			`ip6tables-save > /etc/iptables/rules.v6`,
			`systemctl enable --now radvd.service`,
		}, Mounts: []Mount{
			{"host_share_docker_images", "/mnt/host/docker_images", "9p", "trans=virtio,version=9p2000.L,rw", "0", "0"},
			{"host_share_vms", "/mnt/host/vms", "9p", "trans=virtio,version=9p2000.L,rw", "0", "0"},
			{"host_share_images", "/mnt/host/images", "9p", "trans=virtio,version=9p2000.L,rw", "0", "0"},
		},
	}
	// Prepend the required jinja template directive
	return userData
}

func buildTargetUserData() *UserData {

	return &UserData{
		SSHpwauth: true,
		Users: []User{
			{
				Name:              "ubuntu",
				Sudo:              "ALL=(ALL) NOPASSWD:ALL",
				Groups:            "sudo",
				Shell:             "/bin/bash",
				SSHAuthorizedKeys: SSHKeys("{{ ds.meta_data.public_keys | join('\n') }}"),
			},
		},
		ChPasswd: ChPasswd{
			Users: []ChPasswdUser{
				{Name: "ubuntu", Password: "pass", Type: "text"},
				{Name: "root", Password: "pass", Type: "text"},
			},
			Expire: false,
		},
		WriteFiles: []WriteFile{
			{Path: "/etc/systemd/networkd.conf.d/dhcpv6_duid_llt.conf", Permissions: "0644", Content: dhcpv6DuidLltConf},
		},
		RunCmd: RunCmd{
			"rm /etc/update-motd.d/50-landscape-sysinfo",
			"rm /etc/update-motd.d/10-help-text",
			"rm /etc/update-motd.d/50-motd-news",
			"rm /etc/update-motd.d/90-updates-available",
			"systemctl restart systemd-networkd",
		},
	}
}

func buildProvisionerNetworkConfig(provisionerIP string, prefixLen int, provisionerIpV6 string, prefixLenV6 int) *NetplanConfig {
	addresses := []string{fmt.Sprintf("%s/%d", provisionerIP, prefixLen)}
	if provisionerIpV6 != "" {
		addresses = append(addresses, fmt.Sprintf("%s/%d", provisionerIpV6, prefixLenV6))
	}

	cfg := &NetplanConfig{}
	cfg.Version = 2
	cfg.Ethernets = map[string]NetplanEthernet{
		"enp0s1": {
			DHCP4: true,
			DHCP6: true,
		},
		"enp0s2": {
			DHCP4: false,
			Nameservers: &NetplanNameservers{
				Addresses: []string{provisionerIP, "8.8.8.8", "1.1.1.1"},
				Search:    []string{"~pvmlab.local"},
			},
			Addresses: addresses,
		},
	}
	return cfg
}

func buildTargetNetworkConfig(mac string) *NetplanConfig {
	cfg := &NetplanConfig{}
	cfg.Version = 2
	cfg.Ethernets = map[string]NetplanEthernet{
		"static-interface": {
			Match: &NetplanMatch{MacAddress: mac},
			DHCP4: true,
			DHCP6: true,
		},
	}
	return cfg
}

var execCommand = exec.Command

var CreateISO = func(ctx context.Context, vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image string) error {
	sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa.pub")
	sshKeyBytes, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return err
	}
	sshKey := strings.TrimSpace(string(sshKeyBytes))

	configDir := filepath.Join(appDir, "configs", "cloud-init", vmName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	var metaDataBytes, userDataBytes, networkConfigBytes, vendorDataBytes []byte

	if role == "provisioner" {
		if ip == "" {
			return fmt.Errorf("ip is required for provisioner VMs")
		}
		parsedIP, ipNet, err := net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("failed to parse CIDR %s: %w", ip, err)
		}

		var provisionerIpV6 string
		var prefixLenV6 int
		var dhcpV6Start, dhcpV6End string
		var subnetV6 string
		if ipv6 != "" {
			parsedIPV6, ipv6Net, err := net.ParseCIDR(ipv6)
			if err != nil {
				return fmt.Errorf("failed to parse CIDR %s: %w", ipv6, err)
			}
			prefixLenV6, _ = ipv6Net.Mask.Size()
			provisionerIpV6 = parsedIPV6.String()
			subnetV6 = ipv6

			startIP := net.ParseIP("::").To16()
			startIP[len(startIP)-1] = 100 // ::64 in hex
			endIP := net.ParseIP("::").To16()
			endIP[len(endIP)-1] = 200 // ::c8 in hex
			dhcpV6Start = startIP.String()
			dhcpV6End = endIP.String()
		}

		ip4 := parsedIP.To4()
		if ip4 == nil {
			return fmt.Errorf("only IPv4 is supported")
		}
		prefixLen, _ := ipNet.Mask.Size()
		networkIP := ipNet.IP.To4()
		dhcpStart := fmt.Sprintf("%d.%d.%d.100", networkIP[0], networkIP[1], networkIP[2])
		dhcpEnd := fmt.Sprintf("%d.%d.%d.200", networkIP[0], networkIP[1], networkIP[2])

		metaData := buildProvisionerMetaData(
			sshKey, tar, image, parsedIP.String(), dhcpStart, dhcpEnd,
			provisionerIpV6, dhcpV6Start, dhcpV6End, subnetV6,
		)
		metaDataBytes, err = marshal(metaData)
		if err != nil {
			return fmt.Errorf("failed to marshal meta-data: %w", err)
		}

		userData := buildProvisionerUserData(ipv6 != "")
		userDataBytes, err = marshal(userData)
		if err != nil {
			return fmt.Errorf("failed to marshal user-data: %w", err)
		}
		// Prepend the jinja template directive, which is not part of the YAML standard
		userDataBytes = append([]byte("## template: jinja\n#cloud-config\n"), userDataBytes...)

		networkConfig := buildProvisionerNetworkConfig(parsedIP.String(), prefixLen, provisionerIpV6, prefixLenV6)
		networkConfigBytes, err = marshal(networkConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal network-config: %w", err)
		}

	} else { // role == "target"
		if mac == "" {
			return fmt.Errorf("mac is required for target VMs")
		}

		metaData := buildTargetMetaData(vmName, sshKey)
		metaDataBytes, err = marshal(metaData)
		if err != nil {
			return fmt.Errorf("failed to marshal meta-data: %w", err)
		}

		userData := buildTargetUserData()
		userDataBytes, err = marshal(userData)
		if err != nil {
			return fmt.Errorf("failed to marshal user-data: %w", err)
		}
		// Prepend the jinja template directive
		userDataBytes = append([]byte("## template: jinja\n#cloud-config\n"), userDataBytes...)

		networkConfig := buildTargetNetworkConfig(mac)
		networkConfigBytes, err = marshal(networkConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal network-config: %w", err)
		}
	}

	if err := os.WriteFile(filepath.Join(configDir, "meta-data"), metaDataBytes, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(configDir, "user-data"), userDataBytes, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(configDir, "network-config"), networkConfigBytes, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(configDir, "vendor-data"), vendorDataBytes, 0644); err != nil {
		return err
	}

	cmd := execCommand("mkisofs", "-o", isoPath, "-V", "cidata", "-r", "-J", configDir)
	return cmd.Run()
}
