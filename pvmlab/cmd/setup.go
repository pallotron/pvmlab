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
	"provisioning-vm-lab/internal/runner"

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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Creating provisioning-vm-lab directory structure...")
		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

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
				fmt.Println(err)
				os.Exit(1)
			}
		}

		fmt.Println("Directory structure created successfully.")

		fmt.Println("Generating SSH keys...")
		if err := generateSSHKeys(filepath.Join(appDir, "ssh")); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("SSH keys generated successfully.")

		fmt.Println("Checking dependencies...")
		if err := checkDependencies(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Dependencies checked successfully.")

		fmt.Println("Checking socket_vmnet service status...")
	},
}

func checkDependencies() error {
	dependencies := []string{"brew", "mkisofs", "socat", "qemu-system-aarch64"}

	for _, dep := range dependencies {
		cmd := exec.Command("which", dep)
		if err := runner.Run(cmd); err != nil {
			fmt.Printf("%s not found. Please install it.\n", dep)
			return err
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
			fmt.Println("socket_vmnet not found in standard paths or /opt. Please install it.")
			return err
		}
	}

	return nil
}

func generateSSHKeys(sshDir string) error {
	privateKeyPath := filepath.Join(sshDir, "vm_rsa")
	publicKeyPath := filepath.Join(sshDir, "vm_rsa.pub")

	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Encode the private key to the PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)
	if err := os.WriteFile(privateKeyPath, privateKeyBytes, 0600); err != nil {
		return err
	}

	// Create the public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	// Write the public key to a file
	return os.WriteFile(publicKeyPath, ssh.MarshalAuthorizedKey(publicKey), 0644)
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
