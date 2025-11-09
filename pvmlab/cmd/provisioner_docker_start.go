package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var dockerTar string
var privileged, networkHost bool

var provisionerDockerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a docker container in the provisioner from a tarball",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.New()
		if err != nil {
			return err
		}

		prov, err := metadata.FindProvisioner(cfg)
		if err != nil {
			return fmt.Errorf("error finding provisioner: %w", err)
		}
		if prov == "" {
			return fmt.Errorf("no provisioner found. Please create one with 'pvmlab provisioner create'")
		}
		vmName := prov
		tarPath := dockerTar
		color.Cyan("i Starting docker container in %s from %s", vmName, tarPath)

		meta, err := metadata.Load(cfg, vmName)
		if err != nil {
			return fmt.Errorf("error loading VM metadata: %w", err)
		}

		appDir := cfg.GetAppDir()

		dockerImagesDir := filepath.Join(appDir, "docker_images")
		if err := os.MkdirAll(dockerImagesDir, 0755); err != nil {
			return fmt.Errorf("error creating docker_images directory: %w", err)
		}

		tarFileName := filepath.Base(tarPath)
		destPath := filepath.Join(dockerImagesDir, tarFileName)

		sourceFile, err := os.Open(tarPath)
		if err != nil {
			return fmt.Errorf("error opening source tarball: %w", err)
		}
		defer sourceFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("error creating destination tarball: %w", err)
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			return fmt.Errorf("error copying tarball: %w", err)
		}
		color.Cyan("i Copied %s to %s", tarPath, destPath)

		containerName := strings.TrimSuffix(tarFileName, ".tar")

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		var sshCmd *exec.Cmd

		if meta.SSHPort == 0 {
			return fmt.Errorf("SSH port not found in metadata, is the VM running?")
		}
		sshPort := fmt.Sprintf("%d", meta.SSHPort)

		var runFlags []string
		if networkHost {
			runFlags = append(runFlags, "--net=host")
		}
		if privileged {
			runFlags = append(runFlags, "--privileged")
		}

		// This assumes that the image tag in the tarball is the same as the container name.
		remoteCmd := fmt.Sprintf(
			"sudo /usr/local/bin/pxeboot_stack_reload.sh %s %s %s",
			tarFileName,
			containerName,
			strings.Join(runFlags, " "),
		)
		sshCmd = exec.Command(
			"ssh", "-i", sshKeyPath, "-p", sshPort,
			"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
			"ubuntu@localhost", "bash", "-c", "'"+remoteCmd+"'",
		)

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error starting container %s: %w\n%s", containerName, err, string(output))
		}
		color.Green("✔ Container %s started successfully.", containerName)
		color.Cyan("Output of docker start:\n%s", string(output))

		color.Cyan("i Pruning dangling docker images")
		pruneCmd := exec.Command(
			"ssh", "-i", sshKeyPath, "-p", sshPort,
			"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
			"ubuntu@localhost", "sudo", "docker", "image", "prune", "-f",
		)
		pruneOutput, err := pruneCmd.CombinedOutput()
		if err != nil {
			color.Yellow("w Warning: failed to prune docker images: %v\n%s", err, string(pruneOutput))
		} else {
			color.Cyan("Output of docker image prune:\n%s", string(pruneOutput))
		}
		return nil
	},
}

func init() {
	provisionerDockerCmd.AddCommand(provisionerDockerStartCmd)
	provisionerDockerStartCmd.Flags().StringVar(&dockerTar, "docker-tar", "", "Path to the container tarball")
	if err := provisionerDockerStartCmd.MarkFlagRequired("docker-tar"); err != nil {
		panic(err)
	}
	provisionerDockerStartCmd.Flags().BoolVar(&privileged, "privileged", false, "Run container in privileged mode")
	provisionerDockerStartCmd.Flags().BoolVar(&networkHost, "network-host", false, "Use host networking for the container")
}
