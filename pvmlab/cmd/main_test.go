package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/metadata"
	"pvmlab/internal/netutil"
	"pvmlab/internal/pidfile"
	"pvmlab/internal/socketvmnet"
	"pvmlab/internal/ssh"
	"testing"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// executeCommand is a helper function to execute a cobra command and capture its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	_, output, err := executeCommandC(root, args...)
	return output, "", err
}

func executeCommandC(root *cobra.Command, args ...string) (*cobra.Command, string, error) {
	// Capture Cobra's output
	cobraBuf := new(bytes.Buffer)
	root.SetOut(cobraBuf)
	root.SetErr(cobraBuf)
	root.SetArgs(args)

	// Redirect color library output to the same buffer
	originalColorOutput := color.Output
	color.Output = cobraBuf
	defer func() { color.Output = originalColorOutput }()

	// Capture direct stdout/stderr writes
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	c, err := root.ExecuteC()

	// Restore stdout/stderr and read from the pipe
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	capturedBuf := new(bytes.Buffer)
	io.Copy(capturedBuf, r)

	// Combine outputs
	combinedOutput := cobraBuf.String() + capturedBuf.String()

	return c, combinedOutput, err
}

func TestMain(m *testing.M) {
	// Save original functions
	originalConfigNew := config.New
	originalDownloadImageIfNotExists := downloader.DownloadImageIfNotExists
	originalCreateDisk := createDisk
	originalCreateISO := createISO
	originalCloudInitCreateISO := cloudinit.CreateISO
	originalMetadataSave := metadata.Save
	originalMetadataFindProvisioner := metadata.FindProvisioner
	originalMetadataFindVM := metadata.FindVM
	originalMetadataGetAll := metadata.GetAll
	originalMetadataDelete := metadata.Delete
	originalSSHGenerateKey := ssh.GenerateKey
	originalSocketVmnetIsSocketVmnetRunning := socketvmnet.IsSocketVmnetRunning
	originalPidfileIsRunning := pidfile.IsRunning
	originalNetutilFindRandomPort := netutil.FindRandomPort
	originalPidfileRead := pidfile.Read

	// Defer restoration of original functions
	defer func() {
		config.New = originalConfigNew
		downloader.DownloadImageIfNotExists = originalDownloadImageIfNotExists
		createDisk = originalCreateDisk
		createISO = originalCreateISO
		cloudinit.CreateISO = originalCloudInitCreateISO
		metadata.Save = originalMetadataSave
		metadata.FindProvisioner = originalMetadataFindProvisioner
		metadata.FindVM = originalMetadataFindVM
		metadata.GetAll = originalMetadataGetAll
		metadata.Delete = originalMetadataDelete
		ssh.GenerateKey = originalSSHGenerateKey
		socketvmnet.IsSocketVmnetRunning = originalSocketVmnetIsSocketVmnetRunning
		pidfile.IsRunning = originalPidfileIsRunning
		netutil.FindRandomPort = originalNetutilFindRandomPort
		pidfile.Read = originalPidfileRead
	}()

	// Run tests
	os.Exit(m.Run())
}

// setupMocks resets all mocks to default successful behavior and configures a temporary app directory.
func setupMocks(t *testing.T) {
	tempDir := t.TempDir()
	config.New = func() (*config.Config, error) {
		cfg := &config.Config{}
		cfg.SetHomeDir(tempDir)
		// Manually populate Distros for testing
		config.Distros = map[string]config.Distro{
			"ubuntu-24.04": {
				Name:    "ubuntu",
				Version: "24.04",
				Arch: map[string]config.ArchInfo{
					"aarch64": {
						ISOURL:     "https://example.com/ubuntu-arm.iso",
						ISOName:    "ubuntu-arm.iso",
						KernelFile: "casper/vmlinuz",
					},
					"x86_64": {
						ISOURL:     "https://example.com/ubuntu-amd64.iso",
						ISOName:    "ubuntu-amd64.iso",
						KernelFile: "casper/vmlinuz",
					},
				},
			},
		}
		return cfg, nil
	}
	downloader.DownloadImageIfNotExists = func(string, string) error {
		return nil
	}
	createDisk = func(ctx context.Context, imagePath, vmDiskPath, diskSize string) error {
		return nil
	}
	createISO = func(vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image string) error {
		return nil
	}
	cloudinit.CreateISO = func(ctx context.Context, vmName, role, appDir, isoPath, ip, ipv6, mac, tar, image string) error {
		return nil
	}
	metadata.Save = func(*config.Config, string, string, string, string, string, string, string, string, string, string, string, string, int, bool, string) error {
		return nil
	}
	metadata.FindProvisioner = func(*config.Config) (string, error) {
		return "", nil // No provisioner exists by default
	}
	metadata.FindVM = func(*config.Config, string) (string, error) {
		return "", nil // No VM exists by default
	}
	metadata.GetAll = func(*config.Config) (map[string]*metadata.Metadata, error) {
		return make(map[string]*metadata.Metadata), nil // No VMs exist by default
	}
	metadata.Delete = func(*config.Config, string) error {
		return nil
	}
	ssh.GenerateKey = func(keyPath string) error {
		// Create a dummy public key file in the temp directory
		pubKeyPath := keyPath + ".pub"
		if err := os.WriteFile(pubKeyPath, []byte("ssh-rsa AAAA... test@example.com"), 0644); err != nil {
			return err
		}
		return nil
	}
	socketvmnet.IsSocketVmnetRunning = func() (bool, error) {
		return true, nil
	}
	pidfile.IsRunning = func(c *config.Config, name string) (bool, error) {
		return false, nil
	}
	netutil.FindRandomPort = func() (int, error) {
		return 12345, nil
	}
	pidfile.Read = func(c *config.Config, name string) (int, error) {
		return 0, os.ErrNotExist
	}
}
