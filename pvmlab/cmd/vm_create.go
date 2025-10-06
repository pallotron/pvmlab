package cmd

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/cloudinit"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/downloader"
	"provisioning-vm-lab/internal/metadata"
	"provisioning-vm-lab/internal/runner"

	"github.com/spf13/cobra"
)

var role, mac string

// vmCreateCmd represents the create command
var vmCreateCmd = &cobra.Command{
	Use:   "create <vm-name>",
	Short: "Creates a new VM",
	Long: `Creates a new VM.
The --role flag determines the type of VM to create.
- provisioner: creates an ARM64 VM with a static IP (192.168.100.1).
- target: creates an AMD64 VM with a static IP (192.168.100.2).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]
		fmt.Printf("Creating VM: %s with role: %s\n", vmName, role)

		var ip, macForMetadata string

		if role == "provisioner" {
			ip = "192.168.100.1"
			macForMetadata = "00:00:DE:AD:BE:EF" // Static MAC for the private interface
			existingProvisioner, err := metadata.FindProvisioner()
			if err != nil {
				fmt.Printf("Error checking for existing provisioner: %v\n", err)
				os.Exit(1)
			}
			if existingProvisioner != "" {
				fmt.Printf("Error: A provisioner VM named '%s' already exists. Only one provisioner is allowed.\n", existingProvisioner)
				os.Exit(1)
			}
		} else { // target
			ip = "192.168.100.2"
			// Check if a VM with this IP already exists
			allMeta, err := metadata.GetAll()
			if err != nil {
				fmt.Printf("Error checking for existing VMs: %v\n", err)
				os.Exit(1)
			}
			for name, meta := range allMeta {
				if meta.IP == ip {
					fmt.Printf("Error: A target VM named '%s' already exists with the IP %s. Only one target is supported for now.\n", name, ip)
					os.Exit(1)
				}
			}

			if mac == "" {
				buf := make([]byte, 6)
				_, err := rand.Read(buf)
				if err != nil {
					fmt.Println("failed to generate random mac address", err)
					os.Exit(1)
				}
				buf[0] = (buf[0] | 2) & 0xfe
				macForMetadata = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
				fmt.Printf("Generated random MAC address: %s\n", macForMetadata)
			} else {
				macForMetadata = mac
			}
		}

		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		var imageUrl string
		var imageFileName string

		if role == "provisioner" {
			imageUrl = config.UbuntuARMImageURL
			imageFileName = "ubuntu-24.04-server-cloudimg-arm64.img"
		} else {
			imageUrl = config.UbuntuAMDImageURL
			imageFileName = "ubuntu-24.04-server-cloudimg-amd64.img"
		}

		fmt.Println("Downloading Ubuntu cloud image...")
		imagePath := filepath.Join(appDir, "images", imageFileName)
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			if err := downloader.DownloadFile(imagePath, imageUrl); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Ubuntu cloud image downloaded successfully.")
		} else {
			fmt.Println("Ubuntu cloud image already exists.")
		}

		fmt.Printf("Creating %s VM disk...\n", vmName)
		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		cmdRun := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", imagePath, vmDiskPath)
		if err := runner.Run(cmdRun); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("Resizing %s VM disk...\n", vmName)
		cmdRun = exec.Command("qemu-img", "resize", vmDiskPath, "10G")
		if err := runner.Run(cmdRun); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("%s VM disk created successfully.\n", vmName)

		fmt.Println("Generating cloud-config ISO...")
		isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
		if err := cloudinit.CreateISO(vmName, role, appDir, isoPath, ip, macForMetadata); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Cloud-config ISO generated successfully.")

		if err := metadata.Save(vmName, role, ip, macForMetadata); err != nil {
			fmt.Printf("Warning: failed to save VM metadata: %v\n", err)
		}
	},
}

func init() {
	vmCmd.AddCommand(vmCreateCmd)
	vmCreateCmd.Flags().StringVar(&role, "role", "target", "The role of the VM (provisioner or target)")
	vmCreateCmd.Flags().StringVar(&mac, "mac", "", "The MAC address of the VM (optional for targets)")
}
