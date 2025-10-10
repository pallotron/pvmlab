package cloudinit

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/runner"
	"strings"
	"text/template"
)

const (
	provisionerMetaDataTemplate = `instance-id: iid-cloudimg-provisioner
local-hostname: provisioner
public-keys:
  - "[[ .SshKey ]]"
pxe_boot_stack_tar: "[[ .PxeBootStackTar ]]"
pxe_boot_stack_name: "[[ .PxeBootStackName ]]"
`
	provisionerUserDataTemplate = `## template: jinja
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

write_files:
  - path: /usr/local/bin/pxeboot_stack_reload.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -e
      echo "Stopping and removing old container..."
      docker stop {{ ds.meta_data.pxe_boot_stack_name }} || true
      docker rm {{ ds.meta_data.pxe_boot_stack_name }} || true
      echo "Loading new image from /mnt/host/docker_images/{{ ds.meta_data.pxe_boot_stack_tar }}..."
      docker load -i /mnt/host/docker_images/{{ ds.meta_data.pxe_boot_stack_tar }}
      echo "Starting new container..."
      docker run --mount type=bind,source=/mnt/host/vms,target=/mnt/host/vms -d --name {{ ds.meta_data.pxe_boot_stack_name }} --net=host --privileged {{ ds.meta_data.pxe_boot_stack_name }}:latest
      echo "Done."

runcmd:
  - 'sed -i "/net.ipv4.ip_forward/d" /etc/sysctl.conf'
  - 'echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf'
  - 'sysctl -p'
  - 'DEBIAN_FRONTEND=noninteractive apt-get -y install acpid iptables-persistent ca-certificates curl gnupg'

  - 'install -m 0755 -d /etc/apt/keyrings'
  - 'curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg'
  - 'chmod a+r /etc/apt/keyrings/docker.gpg'
  - 'sh -c ''echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null'''
  - 'apt-get update'
  - 'DEBIAN_FRONTEND=noninteractive apt-get -y install docker-ce docker-ce-cli containerd.io'
  - 'mkdir -p /mnt/host/docker_images'
  - 'mkdir -p /mnt/host/vms'
  - 'mount -t 9p -o trans=virtio,version=9p2000.L host_share_docker_images /mnt/host/docker_images'
  - 'mount -t 9p -o trans=virtio,version=9p2000.L host_share_vms /mnt/host/vms'
  - 'docker load -i /mnt/host/docker_images/{{ ds.meta_data.pxe_boot_stack_tar }}'
  - 'docker run -d --name {{ ds.meta_data.pxe_boot_stack_name }} --net=host --privileged {{ ds.meta_data.pxe_boot_stack_name }}:latest'

  - 'echo "iptables-persistent iptables-persistent/autosave_v4 boolean true" | debconf-set-selections'
  - 'echo "iptables-persistent iptables-persistent/autosave_v6 boolean true" | debconf-set-selections'
  - 'DEBIAN_FRONTEND=noninteractive apt-get -y install iptables-persistent'
  - 'iptables -t nat -A POSTROUTING -o enp0s1 -j MASQUERADE'
  - 'iptables-save > /etc/iptables/rules.v4'
`
	// jinja templates not supported for network-config in cloud-init/cloud-config
	provisionerNetworkConfigTemplate = `version: 2
ethernets:
  enp0s1:
    dhcp4: true
  enp0s2:
    dhcp4: false
    addresses: ["[[ .Ip ]]/24"]
`
	provisionerVendorData = ``
)

const (
	targetMetaDataTemplate = `instance-id: iid-cloudimg-[[ .VmName ]]
local-hostname: [[ .VmName ]]
public-keys:
  - "[[ .SshKey ]]"
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

	// jinja templates not supported for network-config in cloud-init/cloud-config
	targetNetworkConfigTemplate = `network:
  version: 2
  ethernets:
    static-interface:
      match:
        macaddress: "[[ .Mac ]]"
      dhcp4: true
`
	targetVendorData = ``
)

func executeTemplate(name, tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Delims("[[", "]]").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func CreateISO(vmName, role, appDir, isoPath, ip, mac, pxeBootStackTar string) error {
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

	var metaData, userData, networkConfig, vendorData string

	if role == "provisioner" {
		if ip == "" {
			return fmt.Errorf("ip is required for provisioner VMs templates")
		}
		pxeBootStackName := strings.TrimSuffix(pxeBootStackTar, filepath.Ext(pxeBootStackTar))
		data := struct {
			SshKey           string
			PxeBootStackTar  string
			PxeBootStackName string
			Ip               string
		}{
			SshKey:           sshKey,
			PxeBootStackTar:  pxeBootStackTar,
			PxeBootStackName: pxeBootStackName,
			Ip:               ip,
		}

		metaData, err = executeTemplate("provisionerMetaData", provisionerMetaDataTemplate, data)
		if err != nil {
			return err
		}
		userData = provisionerUserDataTemplate
		networkConfig, err = executeTemplate("provisionerNetworkConfig", provisionerNetworkConfigTemplate, data)
		if err != nil {
			return err
		}
		vendorData = provisionerVendorData
	} else {
		if mac == "" {
			return fmt.Errorf("mac is required for target VMs templates")
		}
		data := struct {
			VmName string
			SshKey string
			Mac    string
		}{
			VmName: vmName,
			SshKey: sshKey,
			Mac:    mac,
		}
		metaData, err = executeTemplate("targetMetaData", targetMetaDataTemplate, data)
		if err != nil {
			return err
		}
		userData = targetUserData
		networkConfig, err = executeTemplate("targetNetworkConfig", targetNetworkConfigTemplate, data)
		if err != nil {
			return err
		}
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
