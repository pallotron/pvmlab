package cloudinit

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/runner"
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
provisioner_ip: "[[ .ProvisionerIp ]]"
dhcp_range_start: "[[ .DhcpRangeStart ]]"
dhcp_range_end: "[[ .DhcpRangeEnd ]]"
[[ if .DhcpRangeV6Start -]]
dhcp_range_v6_start: "[[ .DhcpRangeV6Start ]]"
dhcp_range_v6_end: "[[ .DhcpRangeV6End ]]"
ipv6_subnet: "[[ .ProvisionerIpV6 ]]/[[ .PrefixLenV6 ]]"
[[ end -]]
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
      if [ "$#" -lt 2 ]; then
          echo "Usage: $0 <tar_file> <container_name> [docker_run_flags...]" >&2
          exit 1
      fi
      TAR_FILE=$1
      CONTAINER_NAME=$2
      shift 2
      DOCKER_RUN_FLAGS="$@"
      echo "Stopping and removing old container..."
      docker stop ${CONTAINER_NAME} || true
      docker rm ${CONTAINER_NAME} || true
      echo "Loading new image from /mnt/host/docker_images/${TAR_FILE}..."
      docker load -i /mnt/host/docker_images/${TAR_FILE}
      export PROVISIONER_IP={{ ds.meta_data.provisioner_ip }}
      export DHCP_RANGE_START={{ ds.meta_data.dhcp_range_start }}
      export DHCP_RANGE_END={{ ds.meta_data.dhcp_range_end }}
      {% if ds.meta_data.dhcp_range_v6_start %}
      export DHCP_RANGE_V6_START={{ ds.meta_data.dhcp_range_v6_start }}
      export DHCP_RANGE_V6_END={{ ds.meta_data.dhcp_range_v6_end }}
      {% endif %}
      echo "Starting new container..."
      docker run --mount type=bind,source=/mnt/host/vms,target=/mnt/host/vms -d \
        -e PROVISIONER_IP=$PROVISIONER_IP \
        -e DHCP_RANGE_START=$DHCP_RANGE_START \
        -e DHCP_RANGE_END=$DHCP_RANGE_END \
        {% if ds.meta_data.dhcp_range_v6_start %}
        -e DHCP_RANGE_V6_START=$DHCP_RANGE_V6_START \
        -e DHCP_RANGE_V6_END=$DHCP_RANGE_V6_END \
        {% endif %} \
        --name ${CONTAINER_NAME} ${DOCKER_RUN_FLAGS} ${CONTAINER_NAME}:latest
      echo "Done."
  - path: /etc/systemd/system/pxeboot.service
    permissions: '0644'
    content: |
      [Unit]
      Description=PXE Boot Stack Docker Container
      After=docker.service network-online.target
      Requires=docker.service network-online.target

      [Service]
      Type=oneshot
      RemainAfterExit=yes
      ExecStart=/usr/local/bin/pxeboot_stack_reload.sh {{ ds.meta_data.pxe_boot_stack_tar }} {{ ds.meta_data.pxe_boot_stack_name }} --net=host --privileged

      [Install]
      WantedBy=multi-user.target
  {% if ds.meta_data.dhcp_range_v6_start -%}
  - path: /etc/radvd.conf
    permissions: '0644'
    content: |
        interface enp0s2
        {
            AdvSendAdvert on;
            AdvManagedFlag on;  # <-- This is the crucial "M" flag
            AdvOtherConfigFlag off; # Or on, if you also provide DNS via DHCPv6
            prefix {{ ds.meta_data.ipv6_subnet }}
            {
                AdvOnLink on;
                AdvAutonomous on; # Tells clients to also use SLAAC for addresses
                AdvRouterAddr on;
            };
        };
   {%- endif %}

runcmd:
  - 'sed -i "/net.ipv4.ip_forward/d" /etc/sysctl.conf'
  - 'echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf'
  - 'sed -i "/net.ipv6.conf.all.forwarding/d" /etc/sysctl.conf'
  - 'echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf'
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
  - 'systemctl daemon-reload'
  - 'systemctl enable --now pxeboot.service'
  - 'echo "iptables-persistent iptables-persistent/autosave_v4 boolean true" | debconf-set-selections'
  - 'echo "iptables-persistent iptables-persistent/autosave_v6 boolean true" | debconf-set-selections'
  - 'DEBIAN_FRONTEND=noninteractive apt-get -y install iptables-persistent'
  - 'iptables -t nat -A POSTROUTING -o enp0s1 -j MASQUERADE'
  - 'ip6tables -t nat -A POSTROUTING -o enp0s1 -j MASQUERADE'
  - 'iptables-save > /etc/iptables/rules.v4'
  - 'ip6tables-save > /etc/iptables/rules.v6'
  - 'DEBIAN_FRONTEND=noninteractive apt-get -y install radvd'
  - 'systemctl daemon-reload'
  - 'systemctl enable --now radvd.service'
  - rm /etc/update-motd.d/50-landscape-sysinfo
  - rm /etc/update-motd.d/10-help-text
  - rm /etc/update-motd.d/50-motd-news
  - rm /etc/update-motd.d/90-updates-available

mounts:
  - ["host_share_docker_images", "/mnt/host/docker_images", "9p", "trans=virtio,version=9p2000.L,rw", "0", "0"]
  - ["host_share_vms", "/mnt/host/vms", "9p", "trans=virtio,version=9p2000.L,rw", "0", "0"]
`
	// jinja templates not supported for network-config in cloud-init/cloud-config
	provisionerNetworkConfigTemplate = `version: 2
ethernets:
  enp0s1:
    dhcp4: true
  enp0s2:
    dhcp4: false
    addresses:
      - "[[ .ProvisionerIp ]]/[[ .PrefixLen ]]"
      [[ if .ProvisionerIpV6 -]]
      - "[[ .ProvisionerIpV6 ]]/[[ .PrefixLenV6 ]]"
      [[ end -]]
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

runcmd:
  - rm /etc/update-motd.d/50-landscape-sysinfo
  - rm /etc/update-motd.d/10-help-text
  - rm /etc/update-motd.d/50-motd-news
  - rm /etc/update-motd.d/90-updates-available

write_files:
  - path: /etc/systemd/networkd.conf.d/dhcpv6_duid_llt.conf
    permissions: '0644'
    content: |
        [DHCPv6]
        DUIDType=link-layer-time
`

	// jinja templates not supported for network-config in cloud-init/cloud-config
	targetNetworkConfigTemplate = `network:
  version: 2
  ethernets:
    static-interface:
      match:
        macaddress: "[[ .Mac ]]"
      dhcp4: true
      dhcp6: true
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

var CreateISO = func(vmName, role, appDir, isoPath, ip, ipv6, mac, tar string) error {
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
			subnetV6 = ipv6Net.IP.String()

			// When using "constructor" in dnsmasq, the prefix must be zero.
			// We provide only the host part of the address.
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

		pxebootStackName := strings.TrimSuffix(tar, filepath.Ext(tar))
		data := struct {
			SshKey           string
			PxeBootStackTar  string
			PxeBootStackName string
			ProvisionerIp    string
			PrefixLen        int
			ProvisionerIpV6  string
			PrefixLenV6      int
			DhcpRangeStart   string
			DhcpRangeEnd     string
			DhcpRangeV6Start string
			DhcpRangeV6End   string
			SubnetV6         string
		}{
			SshKey:           sshKey,
			PxeBootStackTar:  tar,
			PxeBootStackName: pxebootStackName,
			ProvisionerIp:    parsedIP.String(),
			PrefixLen:        prefixLen,
			ProvisionerIpV6:  provisionerIpV6,
			PrefixLenV6:      prefixLenV6,
			DhcpRangeStart:   dhcpStart,
			DhcpRangeEnd:     dhcpEnd,
			DhcpRangeV6Start: dhcpV6Start,
			DhcpRangeV6End:   dhcpV6End,
			SubnetV6:         subnetV6,
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
