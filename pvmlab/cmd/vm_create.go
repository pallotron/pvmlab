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
	"path/filepath"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/errors"
	"pvmlab/internal/metadata"
	"pvmlab/internal/netutil"
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
	ip, ipv6, role, mac, pxebootStackTar, dockerImagesPath, distro string
	vmsPath, diskSize, arch, pxebootStackImage             string
	pxeboot                                                bool
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

		if pxeboot && distro != "ubuntu-24.04" {
			return errors.E("vm-create", fmt.Errorf("only --distro=ubuntu-24.04 is supported for --pxeboot"))
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

		if err := createDirectories(appDir); err != nil {
			return errors.E("vm-create", fmt.Errorf("failed to create app directories: %w", err))
		}

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

		if role == provisionerRole {
			// If user specifies a tar file, use it. Otherwise, pull from registry.
			if cmd.Flags().Changed("docker-pxeboot-stack-tar") {
				absTarPath, err := filepath.Abs(pxebootStackTar)
				if err != nil {
					return errors.E("vm-create", fmt.Errorf("failed to resolve path for --docker-pxeboot-stack-tar: %w", err))
				}

				if _, err := os.Stat(absTarPath); err == nil {
					// File exists at the provided path, so copy it.
					if err := os.MkdirAll(finalDockerImagesPath, 0755); err != nil {
						return errors.E("vm-create", fmt.Errorf("failed to create docker images directory: %w", err))
					}
					destTarPath := filepath.Join(finalDockerImagesPath, filepath.Base(absTarPath))
					if err := copyFile(absTarPath, destTarPath); err != nil {
						return errors.E("vm-create", fmt.Errorf("failed to copy pxeboot stack tar file: %w", err))
					}
					pxebootStackTar = filepath.Base(absTarPath)
				} else if !os.IsNotExist(err) {
					// Some other error with os.Stat
					return errors.E("vm-create", fmt.Errorf("error checking --docker-pxeboot-stack-tar path: %w", err))
				}
				pxebootStackImage = config.GetPxeBootStackImageName()
			} else { // Pull image from registry
				s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
				s.Suffix = fmt.Sprintf(" Pulling docker image %s...", pxebootStackImage)
				s.Start()

				// Pull the docker image
				pullCmd := exec.Command("docker", "pull", pxebootStackImage)
				if err := runner.Run(pullCmd); err != nil {
					s.FinalMSG = color.RedString("✖ Failed to pull docker image.\n")
					return errors.E("vm-create", fmt.Errorf("failed to pull docker image %s: %w", pxebootStackImage, err))
				}
				s.FinalMSG = color.GreenString("✔ Docker image pulled successfully.\n")
				s.Stop()

				s.Suffix = " Saving docker image to tar..."
				s.Start()

				// Save the docker image to a tar file
				if err := os.MkdirAll(finalDockerImagesPath, 0755); err != nil {
					s.FinalMSG = color.RedString("✖ Failed to create docker images directory.\n")
					return errors.E("vm-create", fmt.Errorf("failed to create docker images directory: %w", err))
				}
				destTarPath := filepath.Join(finalDockerImagesPath, pxebootStackTar)
				saveCmd := exec.Command("docker", "save", pxebootStackImage, "-o", destTarPath)
				if err := runner.Run(saveCmd); err != nil {
					s.FinalMSG = color.RedString("✖ Failed to save docker image.\n")
					return errors.E("vm-create", fmt.Errorf("failed to save docker image %s to %s: %w", pxebootStackImage, destTarPath, err))
				}
				pxebootStackTar = filepath.Base(destTarPath)
				pxebootStackImage = config.GetPxeBootStackImageName()
				s.FinalMSG = color.GreenString("✔ Docker image saved successfully in %s.\n", destTarPath)
				s.Stop()
			}
		}

		if err := checkExistingVMs(cfg, vmName, role); err != nil {
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
		            if err := handlePxeBootAssets(cfg, distro, arch); err != nil {
		                return errors.E("vm-create", err)
		            }
		            if err := createBlankDisk(vmDiskPath, diskSize); err != nil {
		                return errors.E("vm-create", err)
		            }
		        } else {
		            var imageUrl, imageName string
		            if role == provisionerRole {
		                imageUrl, imageName = config.GetProvisionerImageURL(arch)
		            } else {
						if arch == "aarch64" {
					imageUrl = config.UbuntuARMImageURL
					imageName = config.UbuntuARMImageName
				} else {
					imageUrl = config.UbuntuAMD64ImageURL
					imageName = config.UbuntuAMD64ImageName
				}
			}
			imagePath := filepath.Join(appDir, "images", imageName)
			if err := downloader.DownloadImageIfNotExists(imagePath, imageUrl); err != nil {
				return errors.E("vm-create", err)
			}
			if err := createDisk(imagePath, vmDiskPath, diskSize); err != nil {
				return errors.E("vm-create", err)
			}
			isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
			if err := cloudinit.CreateISO(
				vmName, role, appDir, isoPath, ip, ipv6, macForMetadata,
				pxebootStackTar, pxebootStackImage,
			); err != nil {
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

		if err := metadata.Save(cfg, vmName, role, arch, ipForMetadata, subnetForMetadata, ipv6ForMetadata, subnetv6ForMetadata, macForMetadata, pxebootStackTar, finalDockerImagesPath, finalVMsPath, string(sshPubKey), sshPort, pxeboot, distro); err != nil {
			color.Yellow("Warning: failed to save VM metadata: %v", err)
		}
		color.Green("✔ VM '%s' created successfully.", vmName)

		return nil
	},
}

func handlePxeBootAssets(cfg *config.Config, distro, arch string) error {
	if _, err := exec.LookPath("7z"); err != nil {
		return fmt.Errorf("7z is not installed. Please install it to extract PXE boot assets")
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Preparing PXE boot assets..."
	s.Start()
	defer s.Stop()

	distroPath := filepath.Join(cfg.GetAppDir(), "images", distro, arch)
	color.Cyan("i Distro path: %s", distroPath)
	if err := os.MkdirAll(distroPath, 0755); err != nil {
		return fmt.Errorf("failed to create distro directory: %w", err)
	}

	var isoURL, isoName string
	if distro == "ubuntu-24.04" {
		if arch == "aarch64" {
			isoURL = config.Ubuntu2404ARMISOURL
			isoName = config.Ubuntu2404ARMISOName
		} else {
			isoURL = config.Ubuntu2404AMD64ISOURL
			isoName = config.Ubuntu2404AMD64ISOName
		}
	} else {
		return fmt.Errorf("unsupported distro: %s", distro)
	}

	isoPath := filepath.Join(distroPath, isoName)
	color.Cyan("i ISO path: %s", isoPath)
	if err := downloader.DownloadImageIfNotExists(isoPath, isoURL); err != nil {
		return err
	}

	s.Suffix = " Extracting vmlinuz kernel..."
	extractCmd := exec.Command("7z", "x", "-y", isoPath, "-o"+distroPath, "casper/vmlinuz")
	color.Cyan("i Running command: %s", extractCmd.String())
	if output, err := extractCmd.CombinedOutput(); err != nil {
		color.Red("! 7z extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract vmlinuz kernel: %w", err)
	}

	extractedVmlinuzPath := filepath.Join(distroPath, "casper", "vmlinuz")
	targetVmlinuzPath := filepath.Join(distroPath, "vmlinuz")
	color.Cyan("i Moving %s to %s", extractedVmlinuzPath, targetVmlinuzPath)

	if _, err := os.Stat(extractedVmlinuzPath); os.IsNotExist(err) {
		return fmt.Errorf("vmlinuz not found at expected path after extraction: %s", extractedVmlinuzPath)
	}

	if err := os.Rename(extractedVmlinuzPath, targetVmlinuzPath); err != nil {
		return fmt.Errorf("failed to move vmlinuz to target path: %w", err)
	}

	if err := os.RemoveAll(filepath.Join(distroPath, "casper")); err != nil {
		color.Yellow("Warning: failed to clean up temporary casper directory: %v", err)
	}

	if err := os.Chmod(targetVmlinuzPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on vmlinuz: %w", err)
	}

	s.Suffix = " Extracting kernel modules..."
	extractPoolCmd := exec.Command("7z", "x", "-y", isoPath, "-o"+distroPath, "pool")
	if output, err := extractPoolCmd.CombinedOutput(); err != nil {
		color.Red("! 7z pool extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract pool directory: %w", err)
	}

	s.Suffix = " Finding kernel modules package..."
	modulesDebPath, err := findFileWithPrefix(filepath.Join(distroPath, "pool", "main", "l", "linux"), "linux-modules-")
	if err != nil {
		return fmt.Errorf("failed to find linux-modules package: %w", err)
	}

	s.Suffix = " Extracting modules from .deb package..."
	modulesExtractDir := filepath.Join(distroPath, "modules_extract")
	if err := os.MkdirAll(modulesExtractDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for module extraction: %w", err)
	}
	extractModulesCmd := exec.Command("7z", "x", "-y", modulesDebPath, "-o"+modulesExtractDir)
	if output, err := extractModulesCmd.CombinedOutput(); err != nil {
		color.Red("! 7z module extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract modules from .deb: %w", err)
	}

	s.Suffix = " Extracting data.tar from .deb package..."
	dataTarPath := filepath.Join(modulesExtractDir, "data.tar")
	extractDataTarCmd := exec.Command("7z", "x", "-y", dataTarPath, "-o"+modulesExtractDir)
	if output, err := extractDataTarCmd.CombinedOutput(); err != nil {
		color.Red("! 7z data.tar extraction failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to extract data.tar from .deb: %w", err)
	}

	s.Suffix = " Creating modules.cpio..."
	modulesCpioPath := filepath.Join(distroPath, "modules.cpio") // Uncompressed CPIO
	// The working directory is set to modulesExtractDir, so find will pick up `lib/modules/...`
	cpioCmd := exec.Command("sh", "-c", fmt.Sprintf("find lib -print | cpio -o -H newc > %s", modulesCpioPath))
	cpioCmd.Dir = modulesExtractDir
	if output, err := cpioCmd.CombinedOutput(); err != nil {
		color.Red("! cpio creation failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to create modules cpio: %w", err)
	}

	// Gzip the CPIO archive
	gzipCmd := exec.Command("gzip", "-f", modulesCpioPath)
	if output, err := gzipCmd.CombinedOutput(); err != nil {
		color.Red("! gzip compression failed. Output:\n%s", string(output))
		return fmt.Errorf("failed to gzip modules cpio: %w", err)
	}

	s.Suffix = " Cleaning up temporary module directories..."
	if err := os.RemoveAll(filepath.Join(distroPath, "pool")); err != nil {
		color.Yellow("Warning: failed to clean up temporary pool directory: %v", err)
	}
	if err := os.RemoveAll(modulesExtractDir); err != nil {
		color.Yellow("Warning: failed to clean up temporary module extraction directory: %v", err)
	}

	s.FinalMSG = color.GreenString("✔ PXE boot assets prepared successfully (vmlinuz and modules.cpio.gz extracted).\n")
	return nil
}

func findFileWithPrefix(dir, prefix string) (string, error) {
	var foundPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), prefix) {
			foundPath = path
			return filepath.SkipDir // Stop searching once found
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if foundPath == "" {
		return "", fmt.Errorf("no file with prefix '%s' found in '%s'", prefix, dir)
	}
	return foundPath, nil
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
	vmCreateCmd.Flags().StringVar(&role, "role", "", "The role of the VM ('provisioner' or 'target')")
	vmCreateCmd.MarkFlagRequired("role")
	vmCreateCmd.Flags().StringVar(&mac, "mac", "", "The MAC address of the VM (Required for Target VMs)")
	vmCreateCmd.Flags().StringVar(
		&pxebootStackTar,
		"docker-pxeboot-stack-tar",
		"pxeboot_stack.tar",
		"Path to the pxeboot stack docker tar file (Required for the provisioner VM, otherwise defaults to ~/.pvmlab/pxeboot_stack.tar)",
	)
	vmCreateCmd.Flags().StringVar(
		&pxebootStackImage,
		"docker-pxeboot-stack-image",
		config.GetPxeBootStackImageURL(),
		"Docker image for the pxeboot stack to pull from a registry.",
	)
	vmCreateCmd.Flags().StringVar(&dockerImagesPath, "docker-images-path", "", "Path to docker images to share with the provisioner VM. Defaults to ~/.pvmlab/docker_images")
	vmCreateCmd.Flags().StringVar(&vmsPath, "vms-path", "", "Path to vms to share with the provisioner VM. Defaults to ~/.pvmlab/vms")
	vmCreateCmd.Flags().StringVar(&ip, "ip", "", "The IP address for the provisioner VM in CIDR format (e.g. 192.168.1.1/24)")
	vmCreateCmd.Flags().StringVar(&ipv6, "ipv6", "", "The IPv6 address for the provisioner VM in CIDR format (e.g. fd00:cafe:babe::1/64)")
	vmCreateCmd.Flags().StringVar(&diskSize, "disk-size", "15G", "The size of the VM disk")
	vmCreateCmd.Flags().BoolVar(&pxeboot, "pxeboot", false, "Create a VM that boots from the network (target role only)")
	vmCreateCmd.Flags().StringVar(&arch, "arch", "aarch64", "The architecture of the VM ('aarch64' or 'x86_64')")
	vmCreateCmd.Flags().StringVar(&distro, "distro", "", "The distribution for the VM (e.g. ubuntu-24.04)")
}
