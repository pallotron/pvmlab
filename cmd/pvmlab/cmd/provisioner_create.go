package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/downloader"

	"github.com/spf13/cobra"
)

// provisionerCreateCmd represents the create command
var provisionerCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates the provisioner VM",
	Long: `Downloads the aarch64 Ubuntu cloud image.
Creates and resizes the provisioner VM disk.
Generates the cloud-config ISO.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Downloading Ubuntu cloud image...")
		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		imagePath := filepath.Join(appDir, "images", "ubuntu-22.04-server-cloudimg-arm64.img")
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			if err := downloader.DownloadFile(imagePath, config.UbuntuARMImageURL); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Ubuntu cloud image downloaded successfully.")
		} else {
			fmt.Println("Ubuntu cloud image already exists.")
		}

		fmt.Println("Creating provisioner VM disk...")
		vmDiskPath := filepath.Join(appDir, "vms", "provisioner.qcow2")
		cmdRun := exec.Command("qemu-img", "create", "-f", "qcow2", "-b", imagePath, vmDiskPath)
		if err := cmdRun.Run(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Resizing provisioner VM disk...")
		cmdRun = exec.Command("qemu-img", "resize", vmDiskPath, "20G")
		if err := cmdRun.Run(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Provisioner VM disk created successfully.")
	},
}

func init() {
	provisionerCmd.AddCommand(provisionerCreateCmd)
}
