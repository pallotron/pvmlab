package cmd

import (
	"os"
	"path/filepath"
	"pvmlab/internal/config"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var distroLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List available distributions",
	Long:  `List available distributions that have been pulled.`, 
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.New()
		if err != nil {
			return err
		}

		if len(config.Distros) == 0 {
			color.Yellow("No distributions defined in the configuration.")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		header := []string{"DISTRO", "ARCH", "STATUS", "ARTIFACTS"}
		table.Header(header)

		// Sort distro names for consistent output
		distroNames := make([]string, 0, len(config.Distros))
		for name := range config.Distros {
			distroNames = append(distroNames, name)
		}
		sort.Strings(distroNames)

		for _, distroName := range distroNames {
			distro := config.Distros[distroName]
			for archName := range distro.Arch {
				distroPath := filepath.Join(cfg.GetAppDir(), "images", distroName, archName)

				vmlinuzPath := filepath.Join(distroPath, "vmlinuz")
				modulesCpioGzPath := filepath.Join(distroPath, "modules.cpio.gz")

				status := color.RedString("Not Pulled")
				if _, errVmlinuz := os.Stat(vmlinuzPath); errVmlinuz == nil {
					if _, errModules := os.Stat(modulesCpioGzPath); errModules == nil {
						status = color.GreenString("Pulled")
					}
				}

				artifacts, err := os.ReadDir(distroPath)
				var artifactNames []string
				if err == nil {
					for _, artifact := range artifacts {
						artifactNames = append(artifactNames, artifact.Name())
					}
				}

				row := []string{
					distroName,
					archName,
					status,
					strings.Join(artifactNames, ", "),
				}
				table.Append(row)
			}
		}

		table.Render()

		return nil
	},
}

func init() {
	distroCmd.AddCommand(distroLsCmd)
}
