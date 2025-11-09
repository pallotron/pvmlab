package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"pvmlab/internal/config"
	"pvmlab/internal/distro"
	"pvmlab/internal/errors"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

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
