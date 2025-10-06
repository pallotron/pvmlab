package cmd

import (
	"fmt"
	"os"
	"provisioning-vm-lab/internal/metadata"
	"provisioning-vm-lab/internal/pidfile"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// vmListCmd represents the list command
var vmListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all created VMs and their status",
	Long:  `Lists all VMs that have been created, showing their role, IP, MAC, and running status.`,
	Run: func(cmd *cobra.Command, args []string) {
		allMeta, err := metadata.GetAll()
		if err != nil {
			fmt.Println("Error getting VM list:", err)
			os.Exit(1)
		}

		if len(allMeta) == 0 {
			fmt.Println("No VMs have been created yet.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tROLE\tPRIVATE IP\tSSH ACCESS\tMAC\tSTATUS")

		// Sort VM names for consistent output
		vmNames := make([]string, 0, len(allMeta))
		for name := range allMeta {
			vmNames = append(vmNames, name)
		}
		sort.Strings(vmNames)

		for _, vmName := range vmNames {
			meta := allMeta[vmName]
			status := "Stopped"
			isRunning, err := pidfile.IsRunning(vmName)
			if err != nil {
				fmt.Printf("Warning: could not check status for %s: %v\n", vmName, err)
			}
			if isRunning {
				status = "Running"
			}

			sshAccess := "N/A"
			if meta.Role == "provisioner" {
				sshAccess = "localhost:2222"
			} else if meta.IP != "" {
				sshAccess = fmt.Sprintf("%s (from provisioner)", meta.IP)
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", vmName, meta.Role, meta.IP, sshAccess, meta.MAC, status)
		}
		w.Flush()
	},
}

func init() {
	vmCmd.AddCommand(vmListCmd)
}
