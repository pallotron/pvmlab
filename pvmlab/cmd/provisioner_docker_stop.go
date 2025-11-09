package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var provisionerDockerStopCmd = &cobra.Command{
	Use:   "stop <container-name>",
	Short: "Stops a docker container in the provisioner VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerName := args[0]

		cfg, err := config.New()
		if err != nil {
			return fmt.Errorf("provisioner-docker-stop: %w", err)
		}
		prov, err := metadata.FindProvisioner(cfg)
		if err != nil {
			return fmt.Errorf("provisioner-docker-stop: %w", err)
		}
		if prov == "" {
			return fmt.Errorf("no provisioner found. Please create one with 'pvmlab provisioner create'")
		}

		meta, err := metadata.Load(cfg, prov)
		if err != nil {
			return fmt.Errorf("error loading provisioner metadata: %w", err)
		}

		if meta.SSHPort == 0 {
			return fmt.Errorf("SSH port not found in metadata, is the provisioner running?")
		}
		sshPort := fmt.Sprintf("%d", meta.SSHPort)
		sshKeyPath := filepath.Join(cfg.GetAppDir(), "ssh", "vm_rsa")

		color.Cyan("i Stopping docker container '%s' in provisioner '%s'", containerName, prov)

		remoteCmd := fmt.Sprintf("sudo docker stop %s", containerName)
		sshCmd := exec.Command(
			"ssh", "-i", sshKeyPath,
			"-p", sshPort, "-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost", remoteCmd,
		)

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error stopping container %s: %w\n%s", containerName, err, string(output))
		}

		color.Green("✔ Container %s stopped successfully.", containerName)
		fmt.Println(string(output))

		return nil
	},
}

func init() {
	provisionerDockerCmd.AddCommand(provisionerDockerStopCmd)
}