package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/downloader"
	"provisioning-vm-lab/internal/runner"
	"provisioning-vm-lab/internal/socketvmnet"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Checks for and installs dependencies.",
	Long: `Checks for and installs dependencies (Homebrew, cdrtools, socat, socket_vmnet, qemu).
Creates the ~/.provisioning-vm-lab/ directory structure.
Generates the SSH key pair and saves it to ~/.provisioning-vm-lab/ssh/.
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

		if err := generateSSHKeys(filepath.Join(appDir, "ssh")); err != nil {
			return err
		}

		imagePath := filepath.Join(appDir, "images", config.UbuntuARMImageName)
		if err := downloader.DownloadImageIfNotExists(imagePath, config.UbuntuARMImageURL); err != nil {
			return err
		}

		if err := checkDependencies(); err != nil {
			return err
		}

		if err := checkSocketVmnetStatus(); err != nil {
			return err
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
		filepath.Join(appDir, "vms"),
		filepath.Join(appDir, "configs", "cloud-init"),
		filepath.Join(appDir, "images"),
		filepath.Join(appDir, "logs"),
		filepath.Join(appDir, "pids"),
		filepath.Join(appDir, "monitors"),
		filepath.Join(appDir, "ssh"),
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

func generateSSHKeys(sshDir string) error {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Generating SSH keys..."
	s.Start()
	defer s.Stop()

	privateKeyPath := filepath.Join(sshDir, "vm_rsa")
	publicKeyPath := filepath.Join(sshDir, "vm_rsa.pub")

	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		s.FinalMSG = color.RedString("✖ Failed to generate SSH keys.\n")
		return err
	}

	// Encode the private key to the PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)
	if err := os.WriteFile(privateKeyPath, privateKeyBytes, 0600); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to write private key.\n")
		return err
	}

	// Create the public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		s.FinalMSG = color.RedString("✖ Failed to create public key.\n")
		return err
	}

	// Write the public key to a file
	if err := os.WriteFile(publicKeyPath, ssh.MarshalAuthorizedKey(publicKey), 0644); err != nil {
		s.FinalMSG = color.RedString("✖ Failed to write public key.\n")
		return err
	}
	s.FinalMSG = color.GreenString("✔ SSH keys generated successfully.\n")
	return nil
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
