package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"
	"pvmlab/internal/runner"
	"pvmlab/internal/socketvmnet"
	"pvmlab/internal/ssh"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// used in integration tests to skip dependency checks as they are installed by the Github Actions runner
var assetsOnly bool

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Checks for and installs dependencies.",
	Long: `Checks for and installs dependencies (Homebrew, cdrtools, socat, socket_vmnet, qemu).
Creates the ~/.pvmlab/ directory structure.
Generates the SSH key pair and saves it to ~/.pvmlab/ssh/.
Make sure launchd is configured to launch the socket_vmnet service.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("i Setting up pvmlab...")

		cfg, err := config.New()
		if err != nil {
			return err
		}

		appDir := cfg.GetAppDir()

		if err := createDirectories(appDir); err != nil {
			return err
		}

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		s.Suffix = " Generating SSH keys..."
		s.Start()
		sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")
		if err := ssh.GenerateKey(sshKeyPath); err != nil {
			s.FinalMSG = color.RedString("✖ Failed to generate SSH keys.\n")
			return err
		}
		s.FinalMSG = color.GreenString("✔ SSH keys generated successfully.\n")
		s.Stop()

		if !assetsOnly {
			if err := checkDependencies(); err != nil {
				return err
			}

			if err := checkSocketVmnetStatus(); err != nil {
				return err
			}
		}

		color.Green("✔ Setup completed successfully.")
		return nil
	},
}

func createDirectories(appDir string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Creating directory structure..."
	s.Start()
	defer s.Stop()

	dirs := []string{
		filepath.Join(appDir, "images"),
		filepath.Join(appDir, "vms"),
		filepath.Join(appDir, "pids"),
		filepath.Join(appDir, "logs"),
		filepath.Join(appDir, "monitors"),
		filepath.Join(appDir, "ssh"),
		filepath.Join(appDir, "configs", "cloud-init"),
		filepath.Join(appDir, "docker_images"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			s.FinalMSG = color.RedString("✖ Failed to create directory structure.\n")
			return err
		}
	}
	s.FinalMSG = color.GreenString("✔ Directory structure created successfully.\n")
	return nil
}

func checkSocketVmnetStatus() error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Checking socket_vmnet service status..."
	s.Start()
	defer s.Stop()

	running, err := socketvmnet.IsSocketVmnetRunning()
	if err != nil {
		s.FinalMSG = color.RedString("✖ Error checking socket_vmnet status.\n")
		return err
	}

	if running {
		s.FinalMSG = color.GreenString("✔ %s service is already running.\n", socketvmnet.ServiceName)
	} else {
		s.FinalMSG = color.YellowString("i %s service is stopped. Run `pvmlab socket_vmnet start` to start it.\n", socketvmnet.ServiceName)
	}
	return nil
}

func checkDependencies() error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Checking dependencies..."
	s.Start()
	defer s.Stop()

	dependencies := []string{"brew", "mkisofs", "socat", "qemu-system-aarch64", "docker"}

	for _, dep := range dependencies {
		cmd := exec.Command("which", dep)
		if err := runner.Run(cmd); err != nil {
			s.FinalMSG = color.RedString("✖ Dependency check failed.\n")
			return fmt.Errorf("%s not found. Please install it", dep)
		}
	}

	// Special check for socket_vmnet
	socketVmnetPaths := []string{
		"/opt/socket_vmnet/bin/socket_vmnet",
		"/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet",
	}
	foundSocketVmnet := false
	for _, p := range socketVmnetPaths {
		if _, err := os.Stat(p); err == nil {
			foundSocketVmnet = true
			break
		}
	}

	if !foundSocketVmnet {
		cmd := exec.Command("which", "socket_vmnet")
		if err := runner.Run(cmd); err != nil {
			s.FinalMSG = color.RedString("✖ Dependency check failed.\n")
			return fmt.Errorf("socket_vmnet not found in standard paths or /opt. Please install it")
		}
	}
	s.FinalMSG = color.GreenString("✔ Dependencies checked successfully.\n")
	return nil
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVar(&assetsOnly, "assets-only", false, "Only download assets, skip dependency checks and system setup.")
	setupCmd.Flags().MarkHidden("assets-only")
}
