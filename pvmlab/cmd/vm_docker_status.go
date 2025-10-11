package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/metadata"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var dockerStatusCmd = &cobra.Command{
	Use:               "status <vm_name>",
	Short:             "Show docker container status in a VM",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Getting docker status for %s", vmName)

		cfg, err := config.New()
		if err != nil {
			return err
		}

		meta, err := metadata.Load(cfg, vmName)
		if err != nil {
			return fmt.Errorf("error loading VM metadata: %w", err)
		}

		appDir := cfg.GetAppDir()

		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		var sshCmd *exec.Cmd

		if meta.Role == "provisioner" {
			if meta.SSHPort == 0 {
				return fmt.Errorf("SSH port not found in metadata, is the VM running?")
			}
			sshPort := fmt.Sprintf("%d", meta.SSHPort)
			remoteCmd := "sudo docker ps -a --format json"
			sshCmd = exec.Command(
				"ssh", "-i", sshKeyPath,
				"-p", sshPort, "-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null", "ubuntu@localhost", remoteCmd,
			)
		} else {
			return fmt.Errorf("target VM not supported for now. Please ssh from the provisioner VM")
		}

		output, err := sshCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error getting docker status: %w\n%s", err, string(output))
		}

		// Parse the JSON output and print a table
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
			color.Yellow("No docker containers found.")
			return nil
		}

		type DockerPsJSON struct {
			ID      string `json:"ID"`
			Image   string `json:"Image"`
			Command string `json:"Command"`
			Status  string `json:"Status"`
			Ports   string `json:"Ports"`
			Names   string `json:"Names"`
		}

		table := tablewriter.NewWriter(os.Stdout)
		header := []string{"CONTAINER ID", "IMAGE", "COMMAND", "STATUS", "PORTS", "NAMES"}

		table.Header(header)

		for _, line := range lines {
			if strings.Contains(line, "Warning: Permanently added") {
				continue
			}
			var container DockerPsJSON
			if err := json.Unmarshal([]byte(line), &container); err != nil {
				// Ignore lines that are not valid JSON
				continue
			}
			if container.Ports == "" {
				container.Ports = "N/A"
			}
			row := []string{container.ID, container.Image, container.Command, container.Status, container.Ports, container.Names}
			table.Append(row)
		}
		table.Render()

		return nil
	},
}

func init() {
	dockerCmd.AddCommand(dockerStatusCmd)
}
