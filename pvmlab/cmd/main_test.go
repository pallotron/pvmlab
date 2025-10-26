package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"pvmlab/internal/cloudinit"
	"pvmlab/internal/config"
	"pvmlab/internal/downloader"
	"pvmlab/internal/metadata"
	"pvmlab/internal/runner"
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
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	// Redirect color library output to the buffer
	originalColorOutput := color.Output
	color.Output = buf
	defer func() { color.Output = originalColorOutput }()

	c, err := root.ExecuteC()

	return c, buf.String(), err
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
	originalRunnerRun := runner.Run

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
		runner.Run = originalRunnerRun
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
	runner.Run = func(*exec.Cmd) error {
		return nil
	}
}