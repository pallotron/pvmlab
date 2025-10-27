package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/metadata"
	"pvmlab/internal/netutil"
	"pvmlab/internal/pidfile"
	"pvmlab/internal/runner"
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
	originalRunnerRun := runner.Run
	originalSSHGenerateKey := ssh.GenerateKey
	originalSocketVmnetIsSocketVmnetRunning := socketvmnet.IsSocketVmnetRunning
	originalPidfileIsRunning := pidfile.IsRunning
	originalNetutilFindRandomPort := netutil.FindRandomPort

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
		runner.Run = originalRunnerRun
		ssh.GenerateKey = originalSSHGenerateKey
		socketvmnet.IsSocketVmnetRunning = originalSocketVmnetIsSocketVmnetRunning
		pidfile.IsRunning = originalPidfileIsRunning
		netutil.FindRandomPort = originalNetutilFindRandomPort
	}()

	// Run tests
	os.Exit(m.Run())
}

// setupMocks resets all mocks to default successful behavior.
func setupMocks(_ *testing.T) {
	config.New = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	downloader.DownloadImageIfNotExists = func(string, string) error {
		return nil
	}
	createDisk = func(string, string, string) error {
		return nil
	}
	createISO = func(string, string, string, string, string, string, string, string, string) error {
		return nil
	}
	cloudinit.CreateISO = func(string, string, string, string, string, string, string, string, string) error {
		return nil
	}
	metadata.Save = func(*config.Config, string, string, string, string, string, string, string, string, string, string, string, int, bool) error {
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
	runner.Run = func(*exec.Cmd) error {
		return nil
	}
	ssh.GenerateKey = func(string) error {
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
}