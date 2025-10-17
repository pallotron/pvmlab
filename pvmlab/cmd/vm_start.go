package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"pvmlab/internal/netutil"
	"pvmlab/internal/pidfile"
	"pvmlab/internal/socketvmnet"
	"pvmlab/internal/waiter"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var wait bool

// vmStartCmd represents the start command
var vmStartCmd = &cobra.Command{
	Use:               "start <vm-name>",
	Short:             "Starts a VM",
	Long:              `Starts a VM using qemu.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		color.Cyan("i Starting VM: %s", vmName)

		cfg, err := config.New()
		if err != nil {
			return err
		}

		running, err := pidfile.IsRunning(cfg, vmName)
		if err != nil {
			return fmt.Errorf("error checking VM status: %w", err)
		}
		if running {
			return fmt.Errorf("VM '%s' is already running", vmName)
		}

		meta, err := metadata.Load(cfg, vmName)
		if err != nil {
			return fmt.Errorf("error loading VM metadata: %w", err)
		}

		appDir := cfg.GetAppDir()

		// Check if the base image exists
		imagePath := filepath.Join(appDir, "images", config.UbuntuARMImageName)
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			return fmt.Errorf("ubuntu cloud image not found. Please run `pvmlab setup` to download it")
		}

		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		if _, err := os.Stat(vmDiskPath); os.IsNotExist(err) {
			return fmt.Errorf("VM disk not found for '%s'. Please run 'pvmlab vm create %s' first", vmName, vmName)
		}

		isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
		if _, err := os.Stat(isoPath); os.IsNotExist(err) {
			return fmt.Errorf("cloud-init ISO for '%s' not found. Please create the VM first", vmName)
		}
		pidPath := filepath.Join(appDir, "pids", vmName+".pid")
		monitorPath := filepath.Join(appDir, "monitors", vmName+".sock")
		logPath := filepath.Join(appDir, "logs", vmName+".log")

		// TODO:
		// we should have options to not run things daemonized, this should go in the vm start command.
		// e.g. vm start --no-daemonize <vm-name>
		// This can be done by using:
		// 	-nographic
		// 	-chardev stdio,id=char0,mux=on,logfile=path/to/file.log,signal=off
		// 	-serial chardev:char0
		// and removing -daemonize
		// Base QEMU arguments common to all roles
		qemuArgs := []string{
			"qemu-system-aarch64",
			"-M", "virt",
			"-smp", "2",
			"-drive", "if=pflash,format=raw,readonly=on,file=/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
			"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", vmDiskPath),
			"-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", isoPath),
			"-display", "none",
			"-daemonize",
			"-pidfile", pidPath,
			"-monitor", fmt.Sprintf("unix:%s,server,nowait", monitorPath),
			"-serial", fmt.Sprintf("file:%s", logPath),
		}

		if meta.Role == "provisioner" {
			// Find and assign an SSH port at the last possible moment.
			sshPort, err := netutil.FindRandomPort()
			if err != nil {
				return fmt.Errorf("could not find an available SSH port: %w", err)
			}
			meta.SSHPort = sshPort
			if err := metadata.Save(cfg, vmName, meta.Role, meta.IP, meta.Subnet, meta.IPv6, meta.SubnetV6, meta.MAC, meta.PxeBootStackTar, meta.DockerImagesPath, meta.VMsPath, meta.SSHPort); err != nil {
				return fmt.Errorf("failed to save updated metadata with new SSH port: %w", err)
			}

			if !netutil.IsPortAvailable(meta.SSHPort) {
				return fmt.Errorf("TCP port %d is already in use", meta.SSHPort)
			}

			if meta.PxeBootStackTar == "" {
				return fmt.Errorf("PxeBootStackTar not set in metadata for provisioner")
			}
			var pxeBootStackPath string
			if filepath.IsAbs(meta.PxeBootStackTar) {
				pxeBootStackPath = meta.PxeBootStackTar
			} else {
				pxeBootStackPath = filepath.Join(appDir, "docker_images", meta.PxeBootStackTar)
			}
			if _, err := os.Stat(pxeBootStackPath); os.IsNotExist(err) {
				return fmt.Errorf("pxeboot stack tarball not found at %s. Please run `make -C pxeboot_stack save` from the project git root", pxeBootStackPath)
			}

			var finalDockerImagesPath string
			if meta.DockerImagesPath != "" {
				finalDockerImagesPath = meta.DockerImagesPath
			} else {
				finalDockerImagesPath = filepath.Join(appDir, "docker_images")
			}

			var finalVMsPath string
			if meta.VMsPath != "" {
				finalVMsPath = meta.VMsPath
			} else {
				finalVMsPath = filepath.Join(appDir, "vms")
			}

			// Append provisioner-specific arguments
			qemuArgs = append(qemuArgs,
				"-m", "4096",
				"-device", "virtio-net-pci,netdev=net0",
				"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22,ipv6=on,ipv4=on,ipv6-net=fd00::/64", meta.SSHPort),
				"-device", fmt.Sprintf("virtio-net-pci,netdev=net1,mac=%s", meta.MAC),
				"-netdev", "socket,id=net1,fd=3",
				"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_docker_images,security_model=passthrough", finalDockerImagesPath),
				"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_vms,security_model=passthrough", finalVMsPath),
			)
		} else { // target
			// Append target-specific arguments
			qemuArgs = append(qemuArgs,
				"-m", "2048",
				"-device", fmt.Sprintf("virtio-net-pci,netdev=net0,mac=%s", meta.MAC),
				"-netdev", "socket,id=net0,fd=3",
			)
		}

		// Disable hardware acceleration in CI environments due to lack of nested virtualization
		if os.Getenv("CI") != "true" {
			qemuArgs = append(qemuArgs, "-cpu", "host", "-accel", "hvf")
		}
		socketPath, err := socketvmnet.GetSocketPath()
		if err != nil {
			return fmt.Errorf("error getting socket_vmnet path: %w", err)
		}

		clientPath, err := getSocketVMNetClientPath()
		if err != nil {
			return fmt.Errorf("error getting socket_vmnet_client path: %w", err)
		}

		finalCmd := []string{clientPath, socketPath}
		finalCmd = append(finalCmd, qemuArgs...)

		if os.Getenv("PVMLAB_DEBUG") == "true" {
			color.Yellow("--- QEMU Command ---")
			for _, arg := range finalCmd {
				fmt.Println("  " + arg)
			}
			color.Yellow("--------------------")
		}

		cmdRun := exec.Command(finalCmd[0], finalCmd[1:]...)
		output, err := cmdRun.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error starting VM '%s': %w\nQEMU output:\n%s", vmName, err, string(output))
		}

		// The command can succeed even if the daemon fails.
		// A short sleep allows the PID file to be created.
		time.Sleep(1 * time.Second)
		running, pidErr := pidfile.IsRunning(cfg, vmName)
		if pidErr != nil || !running {
			return fmt.Errorf("VM command executed, but VM '%s' is not running.\nError checking PID file: %v\nQEMU output (if any):\n%s", vmName, pidErr, string(output))
		}

		color.Green("✔ %s VM started successfully.", vmName)
		// TODO: switch to using QEMU guest agent and unix socket
		if wait {
			timeoutSeconds := 300 // Default to 5 minutes
			if timeoutStr := os.Getenv("PVMLAB_WAIT_TIMEOUT"); timeoutStr != "" {
				if timeout, err := strconv.Atoi(timeoutStr); err == nil {
					timeoutSeconds = timeout
				}
			}
			timeoutDuration := time.Duration(timeoutSeconds) * time.Second
			sshKeyPath := filepath.Join(appDir, "ssh", "vm_rsa")

			if meta.Role == "provisioner" {
				if err := waiter.ForPort("localhost", meta.SSHPort, timeoutDuration); err != nil {
					return err
				}
				if err := waiter.ForCloudInitProvisioner(meta.SSHPort, sshKeyPath, timeoutDuration); err != nil {
					return err
				}
			} else { // target
				provisioner, err := metadata.GetProvisioner(cfg)
				if err != nil {
					return fmt.Errorf("failed to find a running provisioner to wait for target VM: %w", err)
				}
				if provisioner.SSHPort == 0 {
					return fmt.Errorf("provisioner found, but it does not have a forwarded SSH port; is it running?")
				}
				if err := waiter.ForCloudInitTarget(provisioner.SSHPort, meta.IP, sshKeyPath, timeoutDuration); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func getSocketVMNetClientPath() (string, error) {
	// Check standard installation paths for socket_vmnet_client
	paths := []string{
		"/opt/socket_vmnet/bin/socket_vmnet_client",
		"/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet_client",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Fallback to `which` if not found in standard paths
	cmd := exec.Command("which", "socket_vmnet_client")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("socket_vmnet_client not found in standard paths or via 'which'. Please install it")
	}
	return strings.TrimSpace(string(out)), nil
}

func init() {
	vmCmd.AddCommand(vmStartCmd)
	vmStartCmd.Flags().BoolVar(&wait, "wait", true, "Wait for cloud-init to complete before exiting.")
}
