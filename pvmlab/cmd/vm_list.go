package cmd

import (
	"fmt"
	"os"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/pidfile"
	"sort"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// vmListCmd represents the list command
var vmListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all created VMs and their status",
	Long:  `Lists all VMs that have been created, showing their role, IP, MAC, and running status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.New()
		if err != nil {
			return err
		}

		allMeta, err := metadata.GetAll(cfg)
		if err != nil {
			return fmt.Errorf("error getting VM list: %w", err)
		}

		if len(allMeta) == 0 {
			color.Yellow("No VMs have been created yet.")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		header := []string{"NAME", "ARCH", "BOOT TYPE", "PRIVATE IP", "PRIVATE IPV6", "MAC", "DISTRO", "STATUS"}
		table.Header(header)
		// Sort VM names for consistent output
		vmNames := make([]string, 0, len(allMeta))
		for name := range allMeta {
			vmNames = append(vmNames, name)
		}
		sort.Strings(vmNames)
		for _, vmName := range vmNames {
			meta := allMeta[vmName]
			status := color.RedString("Stopped")
			isRunning, err := pidfile.IsRunning(cfg, vmName)
			if err != nil {
				color.Yellow("! Warning: could not check status for %s: %v", vmName, err)
			}
			if isRunning {
				status = color.GreenString("Running")
			}
			displayName := vmName
			if meta.Role == "provisioner" {
				displayName = color.RedString(vmName)
			}
			ipv6 := meta.IPv6
			if ipv6 == "" {
				ipv6 = "N/A"
			}
			bootType := "disk"
			if meta.PxeBoot {
				bootType = "pxe"
			}
			arch := "aarch64"
			if meta.Arch == "x86_64" {
				arch = "x86_64"
			}

			var distroToDisplay string
			if meta.Role == "provisioner" {
				distroToDisplay = "N/A (Provisioner)"
			} else {
				distroToDisplay = meta.Distro
			}

			row := []string{
				displayName,
				arch,
				bootType,
				meta.IP,
				ipv6,
				meta.MAC,
				distroToDisplay,
				status,
			}
			table.Append(row)
		}
		table.Render()
		return nil
	},
}

func init() {
	vmCmd.AddCommand(vmListCmd)
}
