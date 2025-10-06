package cloudinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/runner"
)


const (
	provisionerMetaDataTemplate = `instance-id: iid-cloudimg-provisioner
local-hostname: provisioner
public-keys:
  - "%s"
`
	provisionerUserData = `## template: jinja
#cloud-config
ssh_pwauth: true
users:
  - name: ubuntu
    sudo: "ALL=(ALL) NOPASSWD:ALL"
    groups: sudo
    shell: /bin/bash
    ssh_authorized_keys:
      {{ ds.meta_data.public_keys | join('\n') }}

chpasswd:
  users:
    - {name: ubuntu, password: pass, type: text}
    - {name: root, password: pass, type: text}
  expire: False

runcmd:
  - 'echo "iptables-persistent iptables-persistent/autosave_v4 boolean true" | debconf-set-selections'
  - 'echo "iptables-persistent iptables-persistent/autosave_v6 boolean true" | debconf-set-selections'
  - 'apt-get update'
  - 'DEBIAN_FRONTEND=noninteractive apt-get -y install acpid iptables-persistent'
  - 'sed -i "/net.ipv4.ip_forward/d" /etc/sysctl.conf'
  - 'echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf'
  - 'sysctl -p'
  - 'iptables -t nat -A POSTROUTING -o enp0s1 -j MASQUERADE'
  - 'iptables-save > /etc/iptables/rules.v4'
`
	provisionerNetworkConfig = `version: 2
ethernets:
  enp0s1:
    dhcp4: true
  enp0s2:
    dhcp4: false
    addresses: [192.168.100.1/24]
`
	provisionerVendorData = ``
)

const (
	targetMetaDataTemplate = `instance-id: iid-cloudimg-%s
local-hostname: %s
public-keys:
  - "%s"
`
	targetUserData = `## template: jinja
#cloud-config
ssh_pwauth: true
users:
  - name: ubuntu
    sudo: "ALL=(ALL) NOPASSWD:ALL"
    groups: sudo
    shell: /bin/bash
    ssh_authorized_keys:
      {{ ds.meta_data.public_keys | join('\n') }}

chpasswd:
  users:
    - {name: ubuntu, password: pass, type: text}
    - {name: root, password: pass, type: text}
  expire: False
`
	targetNetworkConfigTemplate = `network:
  version: 2
  ethernets:
    static-interface:
      match:
        macaddress: "%s"
      dhcp4: false
      addresses:
        - %s/24
      routes:
        - to: 0.0.0.0/0
          via: 192.168.100.1
      nameservers:
        addresses: [8.8.8.8]
`
	targetVendorData = ``
)

func CreateISO(vmName, role, appDir, isoPath, ip, mac string) error {
	sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa.pub")
	sshKey, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return err
	}

	configDir := filepath.Join(appDir, "configs", "cloud-init", vmName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	var metaData, userData, networkConfig, vendorData string

	if role == "provisioner" {
		metaData = fmt.Sprintf(provisionerMetaDataTemplate, string(sshKey))
		userData = provisionerUserData
		networkConfig = provisionerNetworkConfig
		vendorData = provisionerVendorData
	} else {
		if ip == "" || mac == "" {
			return fmt.Errorf("ip and mac are required for target VMs")
		}
		metaData = fmt.Sprintf(targetMetaDataTemplate, vmName, vmName, string(sshKey))
		userData = targetUserData
		networkConfig = fmt.Sprintf(targetNetworkConfigTemplate, mac, ip)
		vendorData = targetVendorData
	}

	if err := os.WriteFile(filepath.Join(configDir, "meta-data"), []byte(metaData), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(configDir, "user-data"), []byte(userData), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(configDir, "network-config"), []byte(networkConfig), 0644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(configDir, "vendor-data"), []byte(vendorData), 0644); err != nil {
		return err
	}

	cmd := exec.Command("mkisofs", "-o", isoPath, "-V", "cidata", "-r", "-J", configDir)
	return runner.Run(cmd)
}
