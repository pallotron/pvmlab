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
		header := []string{"NAME", "ROLE", "PRIVATE IP", "PRIVATE IPV6", "MAC", "STATUS"}
		boldHeader := make([]string, len(header))
		for i, h := range header {
			boldHeader[i] = color.New(color.Bold).Sprint(h)
		}
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
			if meta.Role == "provisioner" {
				role = color.RedString(role)
			}
			ipv6 := meta.IPv6
			if ipv6 == "" {
				ipv6 = "N/A"
			}
			if err := table.Append([]string{
				vmName,
				meta.Role,
				meta.IP,
				ipv6,
				meta.MAC,
				status,
			}); err != nil {
				return err
			}
		}
		return table.Render()
	},
}

func init() {
	vmCmd.AddCommand(vmListCmd)
}
