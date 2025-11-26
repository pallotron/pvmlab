package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"pvmlab/internal/assets"
	"pvmlab/internal/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	osGeteuid      = os.Geteuid
	osMkdirAll     = os.MkdirAll
	utilCopyFile   = util.CopyFile
	utilRunCommand = util.RunCommand
)

// systemSetupLaunchdCmd represents the setup-launchd command
var systemSetupLaunchdCmd = &cobra.Command{
	Use:   "setup-launchd",
	Short: "Installs and configures the socket_vmnet launchd service",
	Long: `Installs the socket_vmnet wrapper script and launchd plist.
This command requires root privileges (sudo).

It performs the following operations:
1. Creates /opt/pvmlab/libexec/
2. Installs socket_vmnet_wrapper.sh to /opt/pvmlab/libexec/
3. Installs io.github.pallotron.pvmlab.socket_vmnet.plist to /Library/LaunchDaemons/
4. Loads and starts the service via launchctl`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if osGeteuid() != 0 {
			return fmt.Errorf("this command must be run as root (use sudo)")
		}

		color.Cyan("i Setting up socket_vmnet launchd service...")

		// Find source files
		wrapperSource, plistSource, err := extractEmbeddedFiles()
		if err != nil {
			return err
		}
		// Clean up temporary files after installation
		defer os.Remove(wrapperSource)
		defer os.Remove(plistSource)

		// 1. Create libexec directory
		libexecDir := "/opt/pvmlab/libexec"
		if err := osMkdirAll(libexecDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", libexecDir, err)
		}

		// 2. Install wrapper script
		destWrapper := filepath.Join(libexecDir, "socket_vmnet_wrapper.sh")
		if err := utilCopyFile(wrapperSource, destWrapper, 0755); err != nil {
			return fmt.Errorf("failed to install wrapper script: %w", err)
		}
		color.Green("✔ Installed wrapper script to %s", destWrapper)

		// 3. Install plist
		plistName := "io.github.pallotron.pvmlab.socket_vmnet.plist"
		destPlist := filepath.Join("/Library/LaunchDaemons", plistName)

		// Stop existing service if running
		// We ignore errors here as it might not be loaded
		_ = utilRunCommand("launchctl", "bootout", "system", destPlist)

		if err := utilCopyFile(plistSource, destPlist, 0644); err != nil {
			return fmt.Errorf("failed to install plist: %w", err)
		}
		color.Green("✔ Installed plist to %s", destPlist)

		// 4. Load and start service
		serviceTarget := "system/io.github.pallotron.pvmlab.socket_vmnet"

		if err := utilRunCommand("launchctl", "enable", serviceTarget); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}

		if err := utilRunCommand("launchctl", "bootstrap", "system", destPlist); err != nil { // Bootstrap might fail if already bootstrapped, try kickstart anyway
			color.Yellow("! launchctl bootstrap returned error (service might already be bootstrapped): %v", err)
		}

		if err := utilRunCommand("launchctl", "kickstart", "-kp", serviceTarget); err != nil {
			return fmt.Errorf("failed to kickstart service: %w", err)
		}

		color.Green("✔ launchd service configured and started successfully.")
		return nil
	},
}

func extractEmbeddedFiles() (string, string, error) {
	wrapperTempFile, err := ioutil.TempFile("", "socket_vmnet_wrapper_*.sh")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temporary wrapper file: %w", err)
	}
	defer wrapperTempFile.Close()

	if _, err := wrapperTempFile.Write(assets.SocketVMNetWrapper); err != nil {
		return "", "", fmt.Errorf("failed to write embedded wrapper to temporary file: %w", err)
	}

	plistTempFile, err := ioutil.TempFile("", "socket_vmnet_plist_*.plist")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temporary plist file: %w", err)
	}
	defer plistTempFile.Close()

	if _, err := plistTempFile.Write(assets.SocketVMNetPlist); err != nil {
		return "", "", fmt.Errorf("failed to write embedded plist to temporary file: %w", err)
	}

	return wrapperTempFile.Name(), plistTempFile.Name(), nil
}

func init() {
	systemCmd.AddCommand(systemSetupLaunchdCmd)
}
