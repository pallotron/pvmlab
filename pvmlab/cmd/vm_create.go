package cmd

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/errors"
	"pvmlab/internal/metadata"
	"pvmlab/internal/qemu"
	"pvmlab/internal/ssh"
	"pvmlab/internal/util"
	"regexp"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	provisionerRole = "provisioner"
	targetRole      = "target"
)

var (
	ip, ipv6, mac, diskSize, arch string
	pxeboot                       bool

	// readFile is a wrapper around os.ReadFile to allow mocking in tests.
	readFile = os.ReadFile
)

// vmCreateCmd represents the create command
var vmCreateCmd = &cobra.Command{
	Use:   "create <vm-name>",
	Short: "Creates a new target VM",
	Long:  `Creates a new target VM that can be provisioned by the provisioner VM.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a context that is cancelled on a SIGINT or SIGTERM.
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		vmName := args[0]
		color.Cyan("i Creating Target VM: %s", vmName)

		if arch != "aarch64" && arch != "x86_64" {
			return errors.E("vm-create", fmt.Errorf("--arch must be either 'aarch64' or 'x86_64'"))
		}

		if pxeboot && distroName == "" {
			return errors.E("vm-create", fmt.Errorf("--distro is required for --pxeboot. Run 'pvmlab distro ls' to see a list of available distributions"))
		}

		cfg, err := config.New()
		if err != nil {
			return errors.E("vm-create", err)
		}

		if ip == "" {
			if err := suggestNextIP(cfg); err != nil {
				return errors.E("vm-create", fmt.Errorf("failed to suggest next available IP: %w", err))
			}
			return nil
		}

		if err := validateIP(ip); err != nil {
			return err
		}

		if err := validateIPv6(ipv6); err != nil {
			return err
		}

		appDir := cfg.GetAppDir()
		if err := createDirectories(appDir); err != nil {
			return errors.E("vm-create", fmt.Errorf("failed to create app directories: %w", err))
		}
		if err := metadata.CheckForDuplicateIPs(cfg, ip, ipv6); err != nil {
			return errors.E("vm-create", err)
		}

		if err := checkExistingVMs(cfg, vmName, targetRole); err != nil {
			return errors.E("vm-create", err)
		}

		macForMetadata, err := getMac(mac)
		if err != nil {
			return errors.E("vm-create", err)
		}

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		if err := ssh.GenerateKey(sshKeyPath); err != nil {
			return errors.E("vm-create", fmt.Errorf("failed to ensure ssh key exists: %w", err))
		}
		sshPubKey, err := readFile(sshKeyPath + ".pub")
		if err != nil {
			return errors.E("vm-create", fmt.Errorf("failed to read ssh public key: %w", err))
		}

		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		if pxeboot {
			distroPath := filepath.Join(appDir, "images", distroName, arch)
			distroInfo, err := config.GetDistro(distroName, arch)
			if err != nil {
				return errors.E("vm-create", fmt.Errorf("failed to get distro info: %w", err))
			}
			kernelPath := filepath.Join(distroPath, filepath.Base(distroInfo.KernelPath))
			initrdPath := filepath.Join(distroPath, filepath.Base(distroInfo.InitrdPath))

			if _, err := os.Stat(kernelPath); os.IsNotExist(err) {
				return errors.E("vm-create", fmt.Errorf("kernel image not found at %s. Please run 'pvmlab distro pull --distro %s --arch %s' first", kernelPath, distroName, arch))
			}
			if _, err := os.Stat(initrdPath); os.IsNotExist(err) {
				return errors.E("vm-create", fmt.Errorf("initrd image not found at %s. Please run 'pvmlab distro pull --distro %s --arch %s' first", initrdPath, distroName, arch))
			}

			if err := createBlankDisk(ctx, vmDiskPath, diskSize); err != nil {
				return errors.E("vm-create", err)
			}
		} else { // For target, get image info from the configured distros
			distroInfo, err := config.GetDistro(distroName, arch)
			if err != nil {
				return errors.E("vm-create", fmt.Errorf("failed to get distro info for non-pxeboot target: %w", err))
			}
			imageUrl := distroInfo.Qcow2URL
			imageName := path.Base(distroInfo.Qcow2URL)

			distroPath := filepath.Join(appDir, "images", distroName, arch)
			if err := os.MkdirAll(distroPath, 0755); err != nil {
				return errors.E("vm-create", fmt.Errorf("failed to create distro image directory: %w", err))
			}
			imagePath := filepath.Join(distroPath, imageName)
			if err := downloader.DownloadImageIfNotExists(ctx, imagePath, imageUrl); err != nil {
				return errors.E("vm-create", err)
			}
			if err := createDisk(ctx, imagePath, vmDiskPath, diskSize); err != nil {
				return errors.E("vm-create", err)
			}
			isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
			if err := cloudinit.CreateISO(
				ctx, vmName, targetRole, appDir, isoPath, ip, ipv6, macForMetadata,
				"", "",
			); err != nil {
				return errors.E("vm-create", err)
			}
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

		var kernel, initrd string
		if distroName != "" {
			distroInfo, err := config.GetDistro(distroName, arch)
			if err != nil {
				return errors.E("vm-create", fmt.Errorf("failed to get distro info: %w", err))
			}
			kernel = filepath.Base(distroInfo.KernelPath)
			initrd = filepath.Base(distroInfo.InitrdPath)
		}

		if err := metadata.Save(cfg, vmName, targetRole, arch, ipForMetadata, subnetForMetadata, ipv6ForMetadata, subnetv6ForMetadata, macForMetadata, "", "", "", string(sshPubKey), kernel, initrd, 0, pxeboot, distroName); err != nil {
			color.Yellow("Warning: failed to save VM metadata: %v", err)
		}
		color.Green("✔ Target VM '%s' created successfully.", vmName)

		return nil
	},
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

var checkExistingVMs = func(cfg *config.Config, vmName, role string) error {
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

var getMac = func(mac string) (string, error) {
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

var createDisk = func(ctx context.Context, imagePath, vmDiskPath, diskSize string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Creating VM disk (press Ctrl+C to cancel)..."
	s.Start()
	defer s.Stop()

	// Ensure the destination directory exists.
	dir := filepath.Dir(vmDiskPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create VM disk directory.\n")
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	cmdRun := exec.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", imagePath, vmDiskPath)
	if err := cmdRun.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		s.FinalMSG = color.RedString("✖ Failed to create VM disk.\n")
		return err
	}

	// Get base image size
	baseImageSize, err := qemu.GetImageVirtualSize(imagePath)
	if err != nil {
		// Don't fail the whole process, just warn and proceed.
		// qemu-img will fail anyway if it's a shrink, and we can rely on that.
		// However, our custom error is more user-friendly.
		color.Yellow("Warning: could not determine base image size: %v. Proceeding with resize...", err)
	} else {
		// Get target disk size
		targetDiskSize, err := util.ParseSize(diskSize)
		if err != nil {
			// This should be caught by validation earlier, but as a safeguard:
			return fmt.Errorf("invalid disk size format '%s': %w", diskSize, err)
		}

		if targetDiskSize < baseImageSize {
			s.FinalMSG = color.RedString("✖ Failed to create VM disk.\n")
			// Convert baseImageSize to human-readable format for the error message
			humanReadableBaseSize := fmt.Sprintf("%.2fG", float64(baseImageSize)/float64(1024*1024*1024))
			return fmt.Errorf("requested disk size '%s' is smaller than the base image size '%s'. Shrinking images is not supported", diskSize, humanReadableBaseSize)
		}
	}

	s.Suffix = " Resizing VM disk (press Ctrl+C to cancel)..."
	cmdRun = exec.CommandContext(ctx, "qemu-img", "resize", vmDiskPath, diskSize)
	if err := cmdRun.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		s.FinalMSG = color.RedString("✖ Failed to resize VM disk.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ VM disk created successfully.\n")
	return nil
}

var createISO = func(vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating cloud-config ISO..."
	s.Start()
	defer s.Stop()

	if err := cloudinit.CreateISO(context.Background(), vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to generate cloud-config ISO.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ Cloud-config ISO generated successfully.\n")
	return nil
}

var createBlankDisk = func(ctx context.Context, vmDiskPath, diskSize string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Creating blank VM disk (press Ctrl+C to cancel)..."
	s.Start()
	defer s.Stop()

	// Ensure the destination directory exists.
	dir := filepath.Dir(vmDiskPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create VM disk directory.\n")
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	cmdRun := exec.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", vmDiskPath, diskSize)
	if err := cmdRun.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			color.Yellow("\nOperation cancelled by user.")
			return nil
		}
		s.FinalMSG = color.RedString("✖ Failed to create blank VM disk.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ Blank VM disk created successfully.\n")
	return nil
}

func copyFile(src, dst string) error {

	in, err := os.Open(src)

	if err != nil {

		return err

	}

	defer in.Close()

	out, err := os.Create(dst)

	if err != nil {

		return err

	}

	defer out.Close()

	_, err = io.Copy(out, in)

	if err != nil {

		return err

	}

	return out.Close()

}

func init() {

	vmCmd.AddCommand(vmCreateCmd)

	vmCreateCmd.Flags().StringVar(&mac, "mac", "", "The MAC address of the VM")

	vmCreateCmd.Flags().StringVar(&ip, "ip", "", "The static IP address for the VM in CIDR format (e.g. 192.168.1.2/24)")

	vmCreateCmd.Flags().StringVar(&ipv6, "ipv6", "", "The static IPv6 address for the VM in CIDR format (e.g. fd00:cafe:babe::2/64)")

	vmCreateCmd.Flags().StringVar(&diskSize, "disk-size", "15G", "The size of the VM disk")

	vmCreateCmd.Flags().BoolVar(&pxeboot, "pxeboot", false, "Create a VM that boots from the network for installation")

	vmCreateCmd.Flags().StringVar(&arch, "arch", "aarch64", "The architecture of the VM ('aarch64' or 'x86_64')")

	vmCreateCmd.Flags().StringVar(&distroName, "distro", "", "The distribution for the VM (e.g. ubuntu-24.04)")

}

func suggestNextIP(cfg *config.Config) error {
	provisioner, err := metadata.GetProvisioner(cfg)
	if err != nil {
		return fmt.Errorf("failed to get provisioner metadata: %w", err)
	}
	if provisioner == nil {
		return fmt.Errorf("provisioner VM not found. Please create a provisioner first")
	}

	_, ipNet, err := net.ParseCIDR(provisioner.IP + "/24")
	if err != nil {
		return fmt.Errorf("failed to parse provisioner IP CIDR: %w", err)
	}

	ip4 := ipNet.IP.To4()
	if ip4 == nil {
		return fmt.Errorf("provisioner IP is not a valid IPv4 address")
	}

	allMeta, err := metadata.GetAll(cfg)
	if err != nil {
		return fmt.Errorf("failed to get all VM metadata: %w", err)
	}

	usedIPs := make(map[string]bool)
	for _, meta := range allMeta {
		if meta.IP != "" {
			usedIPs[meta.IP] = true
		}
	}

	for i := 2; i < 255; i++ {
		ip := net.IPv4(ip4[0], ip4[1], ip4[2], byte(i))
		if !usedIPs[ip.String()] {
			color.Yellow("The --ip flag is required.")
			color.Cyan("  To create a VM with the next available IP, run:")
			fmt.Printf("  pvmlab vm create <vm-name> --distro <distro> --ip %s/24\n", ip.String())
			return nil
		}
	}

	return fmt.Errorf("no available IPs in the range 100-254")
}
