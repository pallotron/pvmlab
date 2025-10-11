package cmd

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/cloudinit"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/downloader"
	"provisioning-vm-lab/internal/errors"
	"provisioning-vm-lab/internal/metadata"
	"provisioning-vm-lab/internal/runner"
	"regexp"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	provisionerRole = "provisioner"
	targetRole      = "target"
	imageFileName   = "ubuntu-24.04-server-cloudimg-arm64.img"
)

var (
	ip, role, mac, pxebootStackTar, dockerImagesPath, vmsPath, diskSize string
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

		if err := validateIP(ip); err != nil {
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

		imagePath := filepath.Join(appDir, "images", imageFileName)
		if err := downloadImage(imagePath, config.UbuntuARMImageURL); err != nil {
			return errors.E("vm-create", err)
		}

		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		if err := createDisk(imagePath, vmDiskPath, diskSize); err != nil {
			return errors.E("vm-create", err)
		}

		isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
		if err := createISO(vmName, role, appDir, isoPath, ip, macForMetadata, pxebootStackTar); err != nil {
			return errors.E("vm-create", err)
		}

		if role != provisionerRole {
			pxebootStackTar = ""
		}

		if err := metadata.Save(cfg, vmName, role, ip, macForMetadata, pxebootStackTar, finalDockerImagesPath, finalVMsPath); err != nil {
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
	if ip != "" && net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
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

func downloadImage(imagePath, imageUrl string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Downloading Ubuntu cloud image..."
	s.Start()
	defer s.Stop()

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		if err := downloader.DownloadFile(imagePath, imageUrl); err != nil {
			s.FinalMSG = color.RedString("✖ Failed to download Ubuntu cloud image.\n")
			return err
		}
		s.FinalMSG = color.GreenString("✔ Ubuntu cloud image downloaded successfully.\n")
	} else {
		s.FinalMSG = color.YellowString("i Ubuntu cloud image already exists.\n")
	}
	return nil
}

func createDisk(imagePath, vmDiskPath, diskSize string) error {
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

func createISO(vmName, role, appDir, isoPath, ip, mac, pxebootStackTar string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating cloud-config ISO..."
	s.Start()
	defer s.Stop()

	if err := cloudinit.CreateISO(vmName, role, appDir, isoPath, ip, mac, pxebootStackTar); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to generate cloud-config ISO.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ Cloud-config ISO generated successfully.\n")
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
		"Path to the pxeboot stack docker tar file (Required for the provisioner VM)",
	)
	vmCreateCmd.Flags().StringVar(&dockerImagesPath, "docker-images-path", "", "Path to docker images to share with the provisioner VM. Defaults to ~/.provisioning-vm-lab/docker_images")
	vmCreateCmd.Flags().StringVar(&vmsPath, "vms-path", "", "Path to vms to share with the provisioner VM. Defaults to ~/.provisioning-vm-lab/vms")
	vmCreateCmd.Flags().StringVar(&ip, "ip", "", "The IP address of the VM (Required for Provisioner and Target VMs)")
	vmCreateCmd.Flags().StringVar(&diskSize, "disk-size", "10G", "The size of the VM disk")
}