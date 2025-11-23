package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"pvmlab/internal/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	osGeteuid      = os.Geteuid
	osMkdirAll     = os.MkdirAll
	osExecutable   = os.Executable
	osGetwd        = os.Getwd
	utilCopyFile   = util.CopyFile
	utilRunCommand = util.RunCommand
	utilFileExists = util.FileExists
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
		wrapperSource, plistSource, err := findSourceFiles()
		if err != nil {
			return err
		}

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

func findSourceFiles() (string, string, error) {
	exePath, err := osExecutable()
	if err != nil {
		return "", "", err
	}
	exeDir := filepath.Dir(exePath)

	// Possible locations for wrapper and plist
	// 1. Homebrew layout:
	//    bin/pvmlab
	//    libexec/socket_vmnet_wrapper.sh
	//    io.github.pallotron.pvmlab.socket_vmnet.plist (in prefix, parent of bin)

	// Check Homebrew layout
	brewWrapper := filepath.Join(exeDir, "..", "libexec", "socket_vmnet_wrapper.sh")
	brewPlist := filepath.Join(exeDir, "..", "io.github.pallotron.pvmlab.socket_vmnet.plist")

	if utilFileExists(brewWrapper) && utilFileExists(brewPlist) {
		return brewWrapper, brewPlist, nil
	}
	// 2. Dev/Source layout (cwd or relative to binary if built in-place)
	//    pvmlab/cmd/../../launchd/...
	//    or just ./launchd if run from root

	// Try finding relative to executable (assuming binary is in root or build/)
	// If binary is in build/, launchd is in ../launchd
	devWrapper := filepath.Join(exeDir, "..", "launchd", "socket_vmnet_wrapper.sh")
	devPlist := filepath.Join(exeDir, "..", "launchd", "io.github.pallotron.pvmlab.socket_vmnet.plist")
	if utilFileExists(devWrapper) && utilFileExists(devPlist) {
		return devWrapper, devPlist, nil
	}

	// Try relative to CWD
	cwd, _ := osGetwd()
	localWrapper := filepath.Join(cwd, "launchd", "socket_vmnet_wrapper.sh")
	localPlist := filepath.Join(cwd, "launchd", "io.github.pallotron.pvmlab.socket_vmnet.plist")
	if utilFileExists(localWrapper) && utilFileExists(localPlist) {
		return localWrapper, localPlist, nil
	}

	return "", "", fmt.Errorf("could not locate installation files (wrapper script and plist). Ensure you are running this from a Homebrew installation or source root")
}

func init() {
	systemCmd.AddCommand(systemSetupLaunchdCmd)
}
