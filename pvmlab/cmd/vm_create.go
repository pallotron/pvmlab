package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/errors"
	"pvmlab/internal/metadata"
	"pvmlab/internal/runner"
	"pvmlab/internal/ssh"
	"regexp"
	"strconv"
	"strings"
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
	pxeboot                      bool
)

// vmCreateCmd represents the create command
var vmCreateCmd = &cobra.Command{
	Use:   "create <vm-name>",
	Short: "Creates a new target VM",
	Long:  `Creates a new target VM that can be provisioned by the provisioner VM.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Creating Target VM: %s", vmName)

		if arch != "aarch64" && arch != "x86_64" {
			return errors.E("vm-create", fmt.Errorf("--arch must be either 'aarch64' or 'x86_64'"))
		}

		if pxeboot && distroName == "" {
			return errors.E("vm-create", fmt.Errorf("--distro is required for --pxeboot. Run 'pvmlab distro ls' to see a list of available distributions"))
		}

		if err := validateIP(ip); err != nil {
			return err
		}

		if err := validateIPv6(ipv6); err != nil {
			return err
		}

		cfg, err := config.New()
		if err != nil {
			return errors.E("vm-create", err)
		}
		appDir := cfg.GetAppDir()

		if err := createDirectories(appDir); err != nil {
			return errors.E("vm-create", fmt.Errorf("failed to create app directories: %w", err))
		}

		if err := metadata.CheckForDuplicateIPs(cfg, ip, ipv6); err != nil {
			return errors.E("vm-create", err)
		}

		if err := checkExistingVMs(cfg, vmName, targetRole); err != nil {
			return err
		}

		macForMetadata, err := getMac(mac)
		if err != nil {
			return errors.E("vm-create", err)
		}

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		if err := ssh.GenerateKey(sshKeyPath); err != nil {
			return errors.E("vm-create", fmt.Errorf("failed to ensure ssh key exists: %w", err))
		}
		sshPubKey, err := os.ReadFile(sshKeyPath + ".pub")
		if err != nil {
			return errors.E("vm-create", fmt.Errorf("failed to read ssh public key: %w", err))
		}

		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		if pxeboot {
			distroPath := filepath.Join(appDir, "images", distroName, arch)
			kernelPath := filepath.Join(distroPath, "vmlinuz")
			modulesPath := filepath.Join(distroPath, "modules.cpio.gz")

			if _, err := os.Stat(kernelPath); os.IsNotExist(err) {
				return errors.E("vm-create", fmt.Errorf("kernel image not found at %s. Please run 'pvmlab distro pull %s --arch %s' first", kernelPath, distroName, arch))
			}
			if _, err := os.Stat(modulesPath); os.IsNotExist(err) {
				return errors.E("vm-create", fmt.Errorf("kernel modules not found at %s. Please run 'pvmlab distro pull %s --arch %s' first", modulesPath, distroName, arch))
			}

			if err := createBlankDisk(vmDiskPath, diskSize); err != nil {
				return errors.E("vm-create", err)
			}
		} else {
			// For target, get image info from the configured distros
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
			if err := downloader.DownloadImageIfNotExists(imagePath, imageUrl); err != nil {
				return errors.E("vm-create", err)
			}
			if err := createDisk(imagePath, vmDiskPath, diskSize); err != nil {
				return errors.E("vm-create", err)
			}
			isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
			if err := cloudinit.CreateISO(
				vmName, targetRole, appDir, isoPath, ip, ipv6, macForMetadata,
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

		if err := metadata.Save(cfg, vmName, targetRole, arch, ipForMetadata, subnetForMetadata, ipv6ForMetadata, subnetv6ForMetadata, macForMetadata, "", "", "", string(sshPubKey), 0, pxeboot, distroName); err != nil {
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

	// Ensure the destination directory exists.
	dir := filepath.Dir(vmDiskPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create VM disk directory.\n")
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	cmdRun := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", imagePath, vmDiskPath)
	if err := runner.Run(cmdRun); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create VM disk.\n")
		return err
	}

	// Get base image size
	baseImageSize, err := getImageVirtualSize(imagePath)
	if err != nil {
		// Don't fail the whole process, just warn and proceed.
		// qemu-img will fail anyway if it's a shrink, and we can rely on that.
		// However, our custom error is more user-friendly.
		color.Yellow("Warning: could not determine base image size: %v. Proceeding with resize...", err)
	} else {
		// Get target disk size
		targetDiskSize, err := parseSize(diskSize)
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

	s.Suffix = " Resizing VM disk..."
	cmdRun = exec.Command("qemu-img", "resize", vmDiskPath, diskSize)
	if err := runner.Run(cmdRun); err != nil {
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

	if err := cloudinit.CreateISO(vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image); err != nil {
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

	// Ensure the destination directory exists.
	dir := filepath.Dir(vmDiskPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create VM disk directory.\n")
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	cmdRun := exec.Command("qemu-img", "create", "-f", "qcow2", vmDiskPath, diskSize)
	if err := runner.Run(cmdRun); err != nil {
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

// getImageVirtualSize executes `qemu-img info` to get the virtual size of an image in bytes.
func getImageVirtualSize(imagePath string) (int64, error) {
	cmd := exec.Command("qemu-img", "info", "--output=json", imagePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to get image info for %s: %w", imagePath, err)
	}

	var info struct {
		VirtualSize int64 `json:"virtual-size"`
	}
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return 0, fmt.Errorf("failed to parse qemu-img info output: %w", err)
	}

	return info.VirtualSize, nil
}

// parseSize converts a size string like "10G", "512M", "2048K" into bytes.
func parseSize(sizeStr string) (int64, error) {
	// Trim whitespace
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, fmt.Errorf("size string is empty")
	}

	// Find the first non-digit character
	var valueStr string
	var unitStr string
	for i, r := range sizeStr {
		if r >= '0' && r <= '9' {
			valueStr += string(r)
		} else {
			unitStr = strings.TrimSpace(sizeStr[i:])
			break
		}
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value in '%s': %w", sizeStr, err)
	}

	unit := strings.ToUpper(unitStr)
	switch unit {
	case "K", "KB":
		value *= 1024
	case "M", "MB":
		value *= 1024 * 1024
	case "G", "GB":
		value *= 1024 * 1024 * 1024
	case "T", "TB":
		value *= 1024 * 1024 * 1024 * 1024
	case "", "B":
		// value is already in bytes
	default:
		return 0, fmt.Errorf("unknown size unit '%s' in '%s'", unit, sizeStr)
	}

	return value, nil
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