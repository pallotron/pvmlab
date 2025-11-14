package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"os/signal"
	"pvmlab/internal/config"
	"pvmlab/internal/distro"
	"pvmlab/internal/errors"
	"strconv"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func checkDockerMemory() error {
	cmd := exec.Command("docker", "info", "--format", "{{.MemTotal}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get docker info: %w", err)
	}

	memTotalStr := strings.TrimSpace(string(output))
	memTotal, err := strconv.ParseInt(memTotalStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse docker memory info: %w", err)
	}

	// 4GB in bytes
	if memTotal < 4095369216 {
		return fmt.Errorf(
			"your docker setup has less than 4GB of memory available to VMs. This may cause issues. " +
				"If you use colima please run `colima stop && colima start --memory 4`")
	}

	return nil
}

// distroPullCmd represents the pull command
var distroPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull a distribution for PXE booting",
	Long:  `Pull a distribution for PXE booting. This will download the ISO and extract the necessary assets.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if distroName == "" {
			return errors.E("distro-pull", fmt.Errorf("--distro is required"))
		}
		if distroPullArch != "aarch64" && distroPullArch != "x86_64" {
			return errors.E("distro-pull", fmt.Errorf("--arch must be either 'aarch64' or 'x86_64'"))
		}

		if err := checkDockerMemory(); err != nil {
			color.Yellow("! Warning: %v", err)
		}

		cfg, err := config.New()
		if err != nil {
			return errors.E("distro-pull", err)
		}

		// Create a context that is cancelled on a SIGINT or SIGTERM.
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		if err := distro.Pull(ctx, cfg, distroName, distroPullArch); err != nil {
			if ctx.Err() == context.Canceled {
				color.Yellow("\nOperation cancelled by user.")
				return nil
			}
			return errors.E("distro-pull", err)
		}

		return nil
	},
}

func init() {
	distroCmd.AddCommand(distroPullCmd)
	distroPullCmd.Flags().StringVar(&distroName, "distro", "ubuntu-24.04", "The distribution to pull (e.g. ubuntu-24.04)")
	distroPullCmd.Flags().StringVar(&distroPullArch, "arch", "aarch64", "The architecture of the distribution ('aarch64' or 'x86_64')")
}
