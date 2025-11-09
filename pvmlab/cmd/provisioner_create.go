package cmd

import (
	"fmt"
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
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	provIP, provIPv6, provMAC, provPxebootStackTar, provDockerImagesPath string
	provVMsPath, provDiskSize, provArch, provPxebootStackImage      string
)

// provisionerCreateCmd represents the create command
var provisionerCreateCmd = &cobra.Command{
	Use:   "create <vm-name>",
	Short: "Creates the provisioner VM",
	Long:  `Creates the provisioner VM, which runs the pxeboot stack container to provision target VMs.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Creating Provisioner VM: %s", vmName)

		if provArch != "aarch64" && provArch != "x86_64" {
			return errors.E("provisioner-create", fmt.Errorf("--arch must be either 'aarch64' or 'x86_64'"))
		}

		if err := validateIP(provIP); err != nil {
			return err
		}

		if err := validateIPv6(provIPv6); err != nil {
			return err
		}

		if provIP == "" {
			return errors.E("provisioner-create", fmt.Errorf("--ip must be specified for the provisioner VM"))
		}

		cfg, err := config.New()
		if err != nil {
			return errors.E("provisioner-create", err)
		}
		appDir := cfg.GetAppDir()

		if err := createDirectories(appDir); err != nil {
			return errors.E("provisioner-create", fmt.Errorf("failed to create app directories: %w", err))
		}

		if err := metadata.CheckForDuplicateIPs(cfg, provIP, provIPv6); err != nil {
			return errors.E("provisioner-create", err)
		}

		finalDockerImagesPath, err := resolvePath(provDockerImagesPath, filepath.Join(appDir, "docker_images"))
		if err != nil {
			return errors.E("provisioner-create", err)
		}

		finalVMsPath, err := resolvePath(provVMsPath, filepath.Join(appDir, "vms"))
		if err != nil {
			return errors.E("provisioner-create", err)
		}

		// If user specifies a tar file, use it. Otherwise, pull from registry.
		if cmd.Flags().Changed("docker-pxeboot-stack-tar") {
			absTarPath, err := filepath.Abs(provPxebootStackTar)
			if err != nil {
				return errors.E("provisioner-create", fmt.Errorf("failed to resolve path for --docker-pxeboot-stack-tar: %w", err))
			}

			if _, err := os.Stat(absTarPath); err == nil {
				if err := os.MkdirAll(finalDockerImagesPath, 0755); err != nil {
					return errors.E("provisioner-create", fmt.Errorf("failed to create docker images directory: %w", err))
				}
				destTarPath := filepath.Join(finalDockerImagesPath, filepath.Base(absTarPath))
				if err := copyFile(absTarPath, destTarPath); err != nil {
					return errors.E("provisioner-create", fmt.Errorf("failed to copy pxeboot stack tar file: %w", err))
				}
				provPxebootStackTar = filepath.Base(absTarPath)
			} else if !os.IsNotExist(err) {
				return errors.E("provisioner-create", fmt.Errorf("error checking --docker-pxeboot-stack-tar path: %w", err))
			}
			provPxebootStackImage = config.GetPxeBootStackImageName()
		} else { // Pull image from registry
			s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
			s.Suffix = fmt.Sprintf(" Pulling docker image %s...", provPxebootStackImage)
			s.Start()

			pullCmd := exec.Command("docker", "pull", provPxebootStackImage)
			if err := runner.Run(pullCmd); err != nil {
				s.FinalMSG = color.RedString("✖ Failed to pull docker image.\n")
				return errors.E("provisioner-create", fmt.Errorf("failed to pull docker image %s: %w", provPxebootStackImage, err))
			}
			s.FinalMSG = color.GreenString("✔ Docker image pulled successfully.\n")
			s.Stop()

			s.Suffix = " Saving docker image to tar..."
			s.Start()

			if err := os.MkdirAll(finalDockerImagesPath, 0755); err != nil {
				s.FinalMSG = color.RedString("✖ Failed to create docker images directory.\n")
				return errors.E("provisioner-create", fmt.Errorf("failed to create docker images directory: %w", err))
			}
			destTarPath := filepath.Join(finalDockerImagesPath, provPxebootStackTar)
			saveCmd := exec.Command("docker", "save", provPxebootStackImage, "-o", destTarPath)
			if err := runner.Run(saveCmd); err != nil {
				s.FinalMSG = color.RedString("✖ Failed to save docker image.\n")
				return errors.E("provisioner-create", fmt.Errorf("failed to save docker image %s to %s: %w", provPxebootStackImage, destTarPath, err))
			}
			provPxebootStackTar = filepath.Base(destTarPath)
			provPxebootStackImage = config.GetPxeBootStackImageName()
			s.FinalMSG = color.GreenString("✔ Docker image saved successfully in %s.\n", destTarPath)
			s.Stop()
		}

		if err := checkExistingVMs(cfg, vmName, provisionerRole); err != nil {
			return err
		}

		macForMetadata, err := getMac(provMAC)
		if err != nil {
			return errors.E("provisioner-create", err)
		}

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		if err := ssh.GenerateKey(sshKeyPath); err != nil {
			return errors.E("provisioner-create", fmt.Errorf("failed to ensure ssh key exists: %w", err))
		}
		sshPubKey, err := os.ReadFile(sshKeyPath + ".pub")
		if err != nil {
			return errors.E("provisioner-create", fmt.Errorf("failed to read ssh public key: %w", err))
		}

		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		imageUrl, imageName := config.GetProvisionerImageURL(provArch)
		imagePath := filepath.Join(appDir, "images", imageName)
		if err := downloader.DownloadImageIfNotExists(imagePath, imageUrl); err != nil {
			return errors.E("provisioner-create", err)
		}
		if err := createDisk(imagePath, vmDiskPath, provDiskSize); err != nil {
			return errors.E("provisioner-create", err)
		}
		isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
		if err := cloudinit.CreateISO(
			vmName, provisionerRole, appDir, isoPath, provIP, provIPv6,
			macForMetadata,
			provPxebootStackTar, provPxebootStackImage,
		); err != nil {
			return errors.E("provisioner-create", err)
		}

		var ipForMetadata, subnetForMetadata string
		if provIP != "" {
			parsedIP, parsedCIDR, err := net.ParseCIDR(provIP)
			if err != nil {
				return fmt.Errorf("internal error: failed to parse already validated IP/CIDR '%s': %w", provIP, err)
			}
			ipForMetadata = parsedIP.String()
			subnetForMetadata = parsedCIDR.String()
		}

		var ipv6ForMetadata, subnetv6ForMetadata string
		if provIPv6 != "" {
			parsedIP, parsedCIDR, err := net.ParseCIDR(provIPv6)
			if err != nil {
				return fmt.Errorf("internal error: failed to parse already validated IPv6/CIDR '%s': %w", provIPv6, err)
			}
			ipv6ForMetadata = parsedIP.String()
			subnetv6ForMetadata = parsedCIDR.String()
		}

		sshPort, err := netutil.FindRandomPort()
		if err != nil {
			return errors.E("provisioner-create", fmt.Errorf("could not find an available SSH port: %w", err))
		}

		if err := metadata.Save(cfg, vmName, provisionerRole, provArch, ipForMetadata, subnetForMetadata, ipv6ForMetadata, subnetv6ForMetadata, macForMetadata, provPxebootStackTar, finalDockerImagesPath, finalVMsPath, string(sshPubKey), sshPort, false, ""); err != nil {
			color.Yellow("Warning: failed to save VM metadata: %v", err)
		}
		color.Green("✔ Provisioner VM '%s' created successfully.", vmName)

		return nil
	},
}

func init() {
	provisionerCmd.AddCommand(provisionerCreateCmd)
	provisionerCreateCmd.Flags().StringVar(&provIP, "ip", "", "The IP address for the provisioner VM in CIDR format (e.g. 192.168.1.1/24)")
	provisionerCreateCmd.Flags().StringVar(&provIPv6, "ipv6", "", "The IPv6 address for the provisioner VM in CIDR format (e.g. fd00:cafe:babe::1/64)")
	provisionerCreateCmd.Flags().StringVar(&provMAC, "mac", "", "The MAC address of the VM")
	provisionerCreateCmd.Flags().StringVar(&provDiskSize, "disk-size", "15G", "The size of the VM disk")
	provisionerCreateCmd.Flags().StringVar(&provArch, "arch", "aarch64", "The architecture of the VM ('aarch64' or 'x86_64')")
	provisionerCreateCmd.Flags().StringVar(&provPxebootStackTar, "docker-pxeboot-stack-tar", "pxeboot_stack.tar", "Path to the pxeboot stack docker tar file")
	provisionerCreateCmd.Flags().StringVar(&provPxebootStackImage, "docker-pxeboot-stack-image", config.GetPxeBootStackImageURL(), "Docker image for the pxeboot stack to pull from a registry")
	provisionerCreateCmd.Flags().StringVar(&provDockerImagesPath, "docker-images-path", "", "Path to docker images to share with the provisioner VM")
	provisionerCreateCmd.Flags().StringVar(&provVMsPath, "vms-path", "", "Path to vms to share with the provisioner VM")
}
