package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"
	"strings"

	"github.com/spf13/cobra"
)

var privileged, networkHost bool

var dockerStartCmd = &cobra.Command{
	Use:   "start <vm_name> <path_to_container_tar>",
	Short: "Start a docker container in a VM from a tarball",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]
		tarPath := args[1]

		meta, err := metadata.Load(vmName)
		if err != nil {
			fmt.Println("Error loading VM metadata:", err)
			os.Exit(1)
		}

		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		dockerImagesDir := filepath.Join(appDir, "docker_images")
		if err := os.MkdirAll(dockerImagesDir, 0755); err != nil {
			fmt.Printf("Error creating docker_images directory: %s\n", err)
			os.Exit(1)
		}

		tarFileName := filepath.Base(tarPath)
		destPath := filepath.Join(dockerImagesDir, tarFileName)

		sourceFile, err := os.Open(tarPath)
		if err != nil {
			fmt.Printf("Error opening source tarball: %s\n", err)
			os.Exit(1)
		}
		defer sourceFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			fmt.Printf("Error creating destination tarball: %s\n", err)
			os.Exit(1)
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			fmt.Printf("Error copying tarball: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Copied %s to %s\n", tarPath, destPath)

		containerName := strings.TrimSuffix(tarFileName, ".tar")

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		var sshCmd *exec.Cmd

		if meta.Role == "provisioner" {
			var runFlags []string
			if networkHost {
				runFlags = append(runFlags, "--net=host")
			}
			if privileged {
				runFlags = append(runFlags, "--privileged")
			}

			// This assumes that the image tag in the tarball is the same as the container name.
			remoteCmd := strings.Join([]string{
				fmt.Sprintf("sudo docker stop %s || true", containerName),
				fmt.Sprintf("sudo docker container rm %s || true", containerName),
				fmt.Sprintf("sudo docker load -i /mnt/host/docker_images/%s", tarFileName),
				fmt.Sprintf("sudo docker run -d --name %s %s %s:latest", containerName, strings.Join(runFlags, " "), containerName),
			}, " && ")
			sshCmd = exec.Command(
				"ssh", "-i", sshKeyPath, "-p", "2222",
				"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
				"ubuntu@localhost", "bash", "-c", "'"+remoteCmd+"'",
			)
		} else {
			fmt.Println("Error: Target VM not supported for now. Please ssh from the provisioner VM.")
			os.Exit(1)
		}

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error starting container %s: %s\n", containerName, err)
			fmt.Println(string(output))
			os.Exit(1)
		}
		fmt.Printf("Container %s started successfully.\n", containerName)
		fmt.Println(string(output))
	},
}

func init() {
	dockerCmd.AddCommand(dockerStartCmd)
	dockerStartCmd.Flags().BoolVar(&privileged, "privileged", false, "Run container in privileged mode")
	dockerStartCmd.Flags().BoolVar(&networkHost, "network-host", false, "Use host networking for the container")
}
