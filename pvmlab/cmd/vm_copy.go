package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/ssh"

	"github.com/spf13/cobra"
)

var (
	recursiveCopy bool
	copyCmd       = &cobra.Command{
		Use:   "copy <source> <destination>",
		Short: "Copy files to/from a VM",
		Long:  `Copy files to/from a VM using scp. One of source or destination must be a remote path (e.g., vm-name:/path/to/file)`,
		RunE:  copy,
	}
)

func init() {
	vmCmd.AddCommand(copyCmd)
	copyCmd.Flags().BoolVarP(&recursiveCopy, "recursive", "r", false, "Recursively copy entire directories")
}

func copy(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("copy command requires exactly two arguments: source and destination")
	}

	source := args[0]
	destination := args[1]

	sourceIsRemote := strings.Contains(source, ":")
	destIsRemote := strings.Contains(destination, ":")

	if sourceIsRemote && destIsRemote {
		return fmt.Errorf("copying between two VMs is not supported")
	}
	if !sourceIsRemote && !destIsRemote {
		return fmt.Errorf("at least one of source or destination must be a remote path (e.g., vm-name:/path/to/file)")
	}

	var vmName, remotePath, localPath string
	var remoteToLocal bool

	if sourceIsRemote {
		parts := strings.SplitN(source, ":", 2)
		if len(parts) < 2 || parts[1] == "" {
			return fmt.Errorf("invalid remote source format: %s. Must be vm-name:/path", source)
		}
		vmName = parts[0]
		remotePath = parts[1]
		localPath = destination
		remoteToLocal = true
	} else { // destIsRemote
		parts := strings.SplitN(destination, ":", 2)
		if len(parts) < 2 || parts[1] == "" {
			return fmt.Errorf("invalid remote destination format: %s. Must be vm-name:/path", destination)
		}
		vmName = parts[0]
		remotePath = parts[1]
		localPath = source
		remoteToLocal = false
	}

	cfg, err := config.New()
	if err != nil {
		return err
	}
	m, err := metadata.Load(cfg, vmName)
	if err != nil {
		return fmt.Errorf("failed to get metadata for VM %s: %w", vmName, err)
	}

	scpPath, err := exec.LookPath("scp")
	if err != nil {
		return fmt.Errorf("scp command not found in PATH")
	}

	scpConnArgs, err := ssh.GetSSHArgs(cfg, m, true)
	if err != nil {
		return err
	}

	var finalScpArgs []string
	if recursiveCopy {
		finalScpArgs = append(finalScpArgs, "-r")
	}
	finalScpArgs = append(finalScpArgs, scpConnArgs...)

	var remoteSpec string
	if m.Role == "provisioner" {
		remoteSpec = fmt.Sprintf("ubuntu@127.0.0.1:%s", remotePath)
	} else {
		remoteSpec = fmt.Sprintf("ubuntu@%s:%s", m.IP, remotePath)
	}

	if remoteToLocal {
		finalScpArgs = append(finalScpArgs, remoteSpec, localPath)
	} else {
		finalScpArgs = append(finalScpArgs, localPath, remoteSpec)
	}

	scpCmd := exec.Command(scpPath, finalScpArgs...)
	scpCmd.Stdout = os.Stdout
	scpCmd.Stderr = os.Stderr
	scpCmd.Stdin = os.Stdin

	return scpCmd.Run()
}

