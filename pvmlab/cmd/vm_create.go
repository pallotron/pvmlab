package cmd

import (
	"crypto/rand"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/errors"
	"pvmlab/internal/metadata"
	"pvmlab/internal/netutil"
	"pvmlab/internal/runner"
	"regexp"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	provisionerRole = "provisioner"
	targetRole      = "target"
	imageFileName   = config.UbuntuARMImageName
)

var (
	ip, ipv6, role, mac, pxebootStackTar, dockerImagesPath, vmsPath, diskSize, arch string
	pxeboot                                                                         bool
)

// vmCreateCmd represents the create command
var vmCreateCmd = &cobra.Command{
	Use:   "create <vm-name>",
	Short: "Creates a new VM",
	Long: `Creates a new VM.
The --role flag determines the type of VM to create.
- provisioner: runs pxeboot stack container
- target: runs the target VM and gets IP from the DHCP server running on the provisioner VM`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Creating VM: %s with role: %s", vmName, role)

		if err := validateRole(role); err != nil {
			return err
		}

		if arch != "aarch64" && arch != "x86_64" {
			return errors.E("vm-create", fmt.Errorf("--arch must be either 'aarch64' or 'x86_64'"))
		}

		if pxeboot && role != targetRole {
			return errors.E("vm-create", fmt.Errorf("--pxeboot can only be used with --role=target"))
		}

		if err := validateIP(ip); err != nil {
			return err
		}

		if err := validateIPv6(ipv6); err != nil {
			return err
		}

		if role == provisionerRole && ip == "" {
			return errors.E("vm-create", fmt.Errorf("--ip must be specified for provisioner VMs"))
		}

		cfg, err := config.New()
		if err != nil {
			return errors.E("vm-create", err)
		}
		appDir := cfg.GetAppDir()

		if err := metadata.CheckForDuplicateIPs(cfg, ip, ipv6); err != nil {
			return errors.E("vm-create", err)
		}

		finalDockerImagesPath, err := resolvePath(dockerImagesPath, filepath.Join(appDir, "docker_images"))
		if err != nil {
			return errors.E("vm-create", err)
		}

		finalVMsPath, err := resolvePath(vmsPath, filepath.Join(appDir, "vms"))
		if err != nil {
			return errors.E("vm-create", err)
		}

		if err := checkExistingVMs(cfg, vmName, role); err != nil {
			return err
		}

		macForMetadata, err := getMac(mac)
		if err != nil {
			return errors.E("vm-create", err)
		}

		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		if pxeboot {
			if err := createBlankDisk(vmDiskPath, diskSize); err != nil {
				return errors.E("vm-create", err)
			}
		} else {
			var imageUrl, imageName string
			if arch == "aarch64" {
				imageUrl = config.UbuntuARMImageURL
				imageName = config.UbuntuARMImageName
			} else {
				imageUrl = config.UbuntuAMD64ImageURL
				imageName = config.UbuntuAMD64ImageName
			}
			imagePath := filepath.Join(appDir, "images", imageName)
			if err := downloader.DownloadImageIfNotExists(imagePath, imageUrl); err != nil {
				return errors.E("vm-create", err)
			}
			if err := createDisk(imagePath, vmDiskPath, diskSize); err != nil {
				return errors.E("vm-create", err)
			}
			isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
			if err := createISO(vmName, role, appDir, isoPath, ip, ipv6, macForMetadata, pxebootStackTar); err != nil {
				return errors.E("vm-create", err)
			}
		}

		if role != provisionerRole {
			pxebootStackTar = ""
		}

		var ipForMetadata, subnetForMetadata string
		if ip != "" {
			parsedIP, parsedCIDR, err := net.ParseCIDR(ip)
			if err != nil {
				return fmt.Errorf("internal error: failed to parse already validated IP/CIDR '%s': %w", ip, err)
			}
			ipForMetadata = parsedIP.String()
			subnetForMetadata = parsedCIDR.String()
		}

		var ipv6ForMetadata, subnetv6ForMetadata string
		if ipv6 != "" {
			parsedIP, parsedCIDR, err := net.ParseCIDR(ipv6)
			if err != nil {
				return fmt.Errorf("internal error: failed to parse already validated IPv6/CIDR '%s': %w", ipv6, err)
			}
			ipv6ForMetadata = parsedIP.String()
			subnetv6ForMetadata = parsedCIDR.String()
		}

		// The provisioner is the only VM that gets a forwarded port from the host,
		// as it acts as a jump-box to the other VMs on the private network.
		var sshPort int
		if role == provisionerRole {
			var err error
			sshPort, err = netutil.FindRandomPort()
			if err != nil {
				return errors.E("vm-create", fmt.Errorf("could not find an available SSH port: %w", err))
			}
		}

		if err := metadata.Save(cfg, vmName, role, arch, ipForMetadata, subnetForMetadata, ipv6ForMetadata, subnetv6ForMetadata, macForMetadata, pxebootStackTar, finalDockerImagesPath, finalVMsPath, sshPort, pxeboot); err != nil {
			color.Yellow("Warning: failed to save VM metadata: %v", err)
		}
		color.Green("✔ VM '%s' created successfully.", vmName)

		return nil
	},
}

func validateRole(role string) error {
	if role != provisionerRole && role != targetRole {
		return fmt.Errorf("--role must be either '%s' or '%s'", provisionerRole, targetRole)
	}
	return nil
}

func validateIP(ip string) error {
	if ip != "" {
		if _, _, err := net.ParseCIDR(ip); err != nil {
			return fmt.Errorf("invalid IP/CIDR address '%s': %w. Please use CIDR notation, e.g., 192.168.1.1/24", ip, err)
		}
	}
	return nil
}

func validateIPv6(ipv6 string) error {
	if ipv6 != "" {
		if _, _, err := net.ParseCIDR(ipv6); err != nil {
			return fmt.Errorf("invalid IPv6/CIDR address '%s': %w. Please use CIDR notation, e.g., fd00:cafe:babe::1/64", ipv6, err)
		}
	}
	return nil
}

func validateMac(mac string) error {
	if mac != "" {
		// regex for mac address
		re := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
		if !re.MatchString(mac) {
			return fmt.Errorf("invalid MAC address: %s", mac)
		}
	}
	return nil
}

func resolvePath(path, defaultPath string) (string, error) {
	if path != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("error resolving path specified by --docker-images-path: %v", err)
		}
		return absPath, nil
	}
	return defaultPath, nil
}

func checkExistingVMs(cfg *config.Config, vmName, role string) error {
	if role == provisionerRole {
		existingProvisioner, err := metadata.FindProvisioner(cfg)
		if err != nil {
			return fmt.Errorf("error checking for existing provisioner: %v", err)
		}
		if existingProvisioner != "" {
			return fmt.Errorf("a provisioner VM named '%s' already exists. Only one provisioner is allowed", existingProvisioner)
		}
	} else { // target
		existingVM, err := metadata.FindVM(cfg, vmName)
		if err != nil {
			return fmt.Errorf("error checking for existing VM: %v", err)
		}
		if existingVM != "" {
			return fmt.Errorf("a VM named '%s' already exists. Only one target VM is allowed", existingVM)
		}
	}
	return nil
}

func getMac(mac string) (string, error) {
	if err := validateMac(mac); err != nil {
		return "", err
	}
	if mac == "" {
		buf := make([]byte, 6)
		_, err := rand.Read(buf)
		if err != nil {
			return "", fmt.Errorf("failed to generate random mac address: %v", err)
		}
		// Set the local bit and clear the multicast bit
		buf[0] = (buf[0] | 2) & 0xfe
		mac = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
		color.Cyan("i Generated random MAC address: %s", mac)
	}
	return mac, nil
}

var createDisk = func(imagePath, vmDiskPath, diskSize string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Creating VM disk..."
	s.Start()
	defer s.Stop()

	cmdRun := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", imagePath, vmDiskPath)
	if err := runner.Run(cmdRun); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create VM disk.\n")
		return err
	}

	s.Suffix = " Resizing VM disk..."
	cmdRun = exec.Command("qemu-img", "resize", vmDiskPath, diskSize)
	if err := runner.Run(cmdRun); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to resize VM disk.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ VM disk created successfully.\n")
	return nil
}

var createISO = func(vmName, role, appDir, isoPath, ip, ipv6, mac, tar string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating cloud-config ISO..."
	s.Start()
	defer s.Stop()

	if err := cloudinit.CreateISO(vmName, role, appDir, isoPath, ip, ipv6, mac, tar); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to generate cloud-config ISO.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ Cloud-config ISO generated successfully.\n")
	return nil
}

var createBlankDisk = func(vmDiskPath, diskSize string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Creating blank VM disk..."
	s.Start()
	defer s.Stop()

	cmdRun := exec.Command("qemu-img", "create", "-f", "qcow2", vmDiskPath, diskSize)
	if err := runner.Run(cmdRun); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create blank VM disk.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ Blank VM disk created successfully.\n")
	return nil
}

func init() {
	vmCmd.AddCommand(vmCreateCmd)
	vmCreateCmd.Flags().StringVar(&role, "role", "target", "The role of the VM ('provisioner' or 'target')")
	vmCreateCmd.Flags().StringVar(&mac, "mac", "", "The MAC address of the VM (Required for Target VMs)")
	vmCreateCmd.Flags().StringVar(
		&pxebootStackTar,
		"docker-pxeboot-stack-tar",
		"pxeboot_stack.tar",
		"Path to the pxeboot stack docker tar file (Required for the provisioner VM, otherwise defaults to ~/.pvmlab/pxeboot_stack.tar)",
	)
	vmCreateCmd.Flags().StringVar(&dockerImagesPath, "docker-images-path", "", "Path to docker images to share with the provisioner VM. Defaults to ~/.pvmlab/docker_images")
	vmCreateCmd.Flags().StringVar(&vmsPath, "vms-path", "", "Path to vms to share with the provisioner VM. Defaults to ~/.pvmlab/vms")
	vmCreateCmd.Flags().StringVar(&ip, "ip", "", "The IP address for the provisioner VM in CIDR format (e.g. 192.168.254.1/24)")
	vmCreateCmd.Flags().StringVar(&ipv6, "ipv6", "", "The IPv6 address for the provisioner VM in CIDR format (e.g. fd00:cafe:babe::1/64)")
	vmCreateCmd.Flags().StringVar(&diskSize, "disk-size", "10G", "The size of the VM disk")
	vmCreateCmd.Flags().BoolVar(&pxeboot, "pxeboot", false, "Create a VM that boots from the network (target role only)")
	vmCreateCmd.Flags().StringVar(&arch, "arch", "aarch64", "The architecture of the VM ('aarch64' or 'x86_64')")
}
