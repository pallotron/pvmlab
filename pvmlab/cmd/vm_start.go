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
	"golang.org/x/term"
)

var wait, interactive bool
var bootOverride string

type vmStartOptions struct {
	vmName string
	cfg    *config.Config
	meta   *metadata.Metadata
	appDir string
}

// vmStartCmd represents the start command
var vmStartCmd = &cobra.Command{
	Use:               "start <vm-name>",
	Short:             "Starts a VM",
	Long:              `Starts a VM using qemu.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: VmNameCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		if wait && interactive {
			return fmt.Errorf("the --wait and --interactive flags are mutually exclusive")
		}
		if bootOverride != "" && bootOverride != "disk" && bootOverride != "pxe" {
			return fmt.Errorf("invalid --boot value: %s. Must be 'disk' or 'pxe'", bootOverride)
		}

		opts, err := gatherVMInfo(args[0])
		if err != nil {
			return err
		}

		isPxeBoot := opts.meta.PxeBoot
		switch bootOverride {
		case "pxe":
			isPxeBoot = true
		case "disk":
			isPxeBoot = false
		}

		if wait && isPxeBoot {
			return fmt.Errorf("the --wait flag is not supported for PXE boot VMs")
		}

		qemuArgs, err := buildQEMUArgs(opts)
		if err != nil {
			return err
		}

		if err := runQEMU(opts, qemuArgs); err != nil {
			return err
		}

		if !interactive {
			if err := handlePostStart(opts); err != nil {
				return err
			}
		}

		return nil
	},
}

func gatherVMInfo(vmName string) (*vmStartOptions, error) {
	color.Cyan("i Starting VM: %s", vmName)

	cfg, err := config.New()
	if err != nil {
		return nil, err
	}

	running, err := pidfile.IsRunning(cfg, vmName)
	if err != nil {
		return nil, fmt.Errorf("error checking VM status: %w", err)
	}
	if running {
		return nil, fmt.Errorf("VM '%s' is already running", vmName)
	}

	meta, err := metadata.Load(cfg, vmName)
	if err != nil {
		return nil, fmt.Errorf("error loading VM metadata: %w", err)
	}

	opts := &vmStartOptions{
		vmName: vmName,
		cfg:    cfg,
		meta:   meta,
		appDir: cfg.GetAppDir(),
	}

	// Check for necessary files
	vmDiskPath := filepath.Join(opts.appDir, "vms", opts.vmName+".qcow2")
	if _, err := os.Stat(vmDiskPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("VM disk not found for '%s'. Please run 'pvmlab vm create %s' first", opts.vmName, opts.vmName)
	}
	if !opts.meta.PxeBoot {
		isoPath := filepath.Join(opts.appDir, "configs", "cloud-init", opts.vmName+".iso")
		if _, err := os.Stat(isoPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("cloud-init ISO for '%s' not found. Please create the VM first", opts.vmName)
		}
	}

	return opts, nil
}

var uefiVarsTemplatePath = "/opt/homebrew/share/qemu/edk2-arm-vars.fd"

func buildQEMUArgs(opts *vmStartOptions) ([]string, error) {
	pidPath := filepath.Join(opts.appDir, "pids", opts.vmName+".pid")
	monitorPath := filepath.Join(opts.appDir, "monitors", opts.vmName+".sock")
	logPath := filepath.Join(opts.appDir, "logs", opts.vmName+".log")
	vmDiskPath := filepath.Join(opts.appDir, "vms", opts.vmName+".qcow2")
	isoPath := filepath.Join(opts.appDir, "configs", "cloud-init", opts.vmName+".iso")

	var qemuBinary, codePath string
	if opts.meta.Arch == "aarch64" {
		qemuBinary = "qemu-system-aarch64"
		codePath = "/opt/homebrew/share/qemu/edk2-aarch64-code.fd"
	} else { // x86_64
		qemuBinary = "qemu-system-x86_64"
		codePath = "/opt/homebrew/share/qemu/edk2-x86_64-code.fd"
	}

	machineType := "virt,gic-version=3"
	if opts.meta.Arch == "x86_64" {
		machineType = "q35"
	}

	// Determine the effective boot mode
	isPxeBoot := opts.meta.PxeBoot
	switch bootOverride {
	case "pxe":
		isPxeBoot = true
	case "disk":
		isPxeBoot = false
	}

	// TODO: https://github.com/pallotron/pvmlab/issues/3
	// Use a more compatible NIC for PXE booting, as the EDK II firmware for aarch64
	// does not have a built-in virtio-net driver, and the loadable ROM is x86-64.
	// When x86-64 support is added, we can use the virtio-net device with its ROM.
	netDevice := "virtio-net-pci"
	if isPxeBoot {
		netDevice = "e1000"
	}

	qemuArgs := []string{
		qemuBinary,
		"-M", machineType,
		"-smp", "2",
	}

	if opts.meta.Arch == "aarch64" {
		// AARCH64 requires separate code and vars pflash drives.
		varsTemplatePath := uefiVarsTemplatePath
		vmVarsPath := filepath.Join(opts.appDir, "vms", opts.vmName+"-vars.fd")
		if _, err := os.Stat(vmVarsPath); os.IsNotExist(err) {
			input, err := os.ReadFile(varsTemplatePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read UEFI vars template: %w", err)
			}
			if err := os.WriteFile(vmVarsPath, input, 0644); err != nil {
				return nil, fmt.Errorf("failed to write UEFI vars file: %w", err)
			}
		}
		qemuArgs = append(qemuArgs,
			"-drive", fmt.Sprintf("if=pflash,format=raw,readonly=on,file=%s", codePath),
			"-drive", fmt.Sprintf("if=pflash,format=raw,file=%s", vmVarsPath),
		)
	} else {
		// x86_64 can use a single, unified pflash drive.
		qemuArgs = append(qemuArgs, "-drive", fmt.Sprintf("if=pflash,format=raw,readonly=on,file=%s", codePath))
	}

	qemuArgs = append(qemuArgs,
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", vmDiskPath),
		"-pidfile", pidPath,
		"-monitor", fmt.Sprintf("unix:%s,server,nowait", monitorPath),
	)

	// The ISO drive is only attached if the VM was created with one.
	if !opts.meta.PxeBoot {
		qemuArgs = append(qemuArgs, "-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", isoPath))
	}

	if isPxeBoot {
		qemuArgs = append(qemuArgs, "-boot", "n")
	}

	if interactive {
		qemuArgs = append(qemuArgs, "-nographic", "-chardev", "stdio,id=char0,mux=on,signal=off", "-serial", "chardev:char0")
	} else {
		qemuArgs = append(qemuArgs, "-display", "none", "-daemonize", "-serial", fmt.Sprintf("file:%s", logPath))
	}

	if opts.meta.Role == "provisioner" {
		sshPort, err := netutil.FindRandomPort()
		if err != nil {
			return nil, fmt.Errorf("could not find an available SSH port: %w", err)
		}
		opts.meta.SSHPort = sshPort
		if err := metadata.Save(opts.cfg, opts.vmName, opts.meta.Role, opts.meta.Arch, opts.meta.IP, opts.meta.Subnet, opts.meta.IPv6, opts.meta.SubnetV6, opts.meta.MAC, opts.meta.PxeBootStackTar, opts.meta.DockerImagesPath, opts.meta.VMsPath, opts.meta.SSHPort, opts.meta.PxeBoot); err != nil {
			return nil, fmt.Errorf("failed to save updated metadata with new SSH port: %w", err)
		}

		finalDockerImagesPath := opts.meta.DockerImagesPath
		if finalDockerImagesPath == "" {
			finalDockerImagesPath = filepath.Join(opts.appDir, "docker_images")
		}
		finalVMsPath := opts.meta.VMsPath
		if finalVMsPath == "" {
			finalVMsPath = filepath.Join(opts.appDir, "vms")
		}

		qemuArgs = append(qemuArgs,
			"-m", "4096",
			"-device", fmt.Sprintf("%s,netdev=net0", netDevice),
			"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22,ipv6=on,ipv4=on,ipv6-net=fd00::/64", opts.meta.SSHPort),
			"-device", fmt.Sprintf("%s,netdev=net1,mac=%s", netDevice, opts.meta.MAC),
			"-netdev", "socket,id=net1,fd=3",

			"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_docker_images,security_model=passthrough", finalDockerImagesPath),
			"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_vms,security_model=passthrough", finalVMsPath),
		)
	} else { // target
		qemuArgs = append(qemuArgs, "-m", "2048", "-device", fmt.Sprintf("%s,netdev=net0,mac=%s", netDevice, opts.meta.MAC), "-netdev", "socket,id=net0,fd=3")
	}

	if opts.meta.Arch == "aarch64" {
		accel := os.Getenv("PVMLAB_QEMU_ACCEL")
		if accel == "" {
			accel = "hvf"
		}
		cpu := "host"
		if accel == "tcg" {
			cpu = "max"
		}
		qemuArgs = append(qemuArgs, "-cpu", cpu, "-accel", accel)
	} else {
		qemuArgs = append(qemuArgs, "-cpu", "max")
	}

	return qemuArgs, nil
}

func runQEMU(opts *vmStartOptions, qemuArgs []string) error {
	socketPath, err := socketvmnet.GetSocketPath()
	if err != nil {
		return fmt.Errorf("error getting socket_vmnet path: %w", err)
	}
	clientPath, err := getSocketVMNetClientPath()
	if err != nil {
		return fmt.Errorf("error getting socket_vmnet_client path: %w", err)
	}

	finalCmd := append([]string{clientPath, socketPath}, qemuArgs...)

	cmdRun := exec.Command(finalCmd[0], finalCmd[1:]...)
	if interactive {
		return runInteractiveSession(cmdRun)
	}
	return runDaemonized(cmdRun, opts)
}

func runInteractiveSession(cmdRun *exec.Cmd) error {
	fmt.Print("\033[H\033[J")
	color.Yellow("--- Interactive Console ---")
	fmt.Println()
	color.Cyan("You are about to connect to the VM's serial console.")
	fmt.Println()
	color.Red("To exit the QEMU session, press: Ctrl-a x")
	fmt.Println()
	color.Yellow("---------------------------")
	fmt.Print("Press Enter to continue, or ESC to cancel...")

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to enter raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	buf := make([]byte, 1)
	os.Stdin.Read(buf)
	term.Restore(fd, oldState)
	fmt.Println()

	if buf[0] == 27 || buf[0] == 3 { // 27 is ESC, 3 is Ctrl-C
		color.Red("Operation cancelled.")
		return nil
	}

	cmdRun.Stdin = os.Stdin
	cmdRun.Stdout = os.Stdout
	cmdRun.Stderr = os.Stderr
	return cmdRun.Run()
}

func runDaemonized(cmdRun *exec.Cmd, opts *vmStartOptions) error {
	output, err := cmdRun.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error starting VM '%s': %w\nQEMU output:\n%s", opts.vmName, err, string(output))
	}
	return nil
}

func handlePostStart(opts *vmStartOptions) error {
	time.Sleep(1 * time.Second)
	running, pidErr := pidfile.IsRunning(opts.cfg, opts.vmName)
	if pidErr != nil || !running {
		return fmt.Errorf("VM command executed, but VM '%s' is not running. Error checking PID file: %v", opts.vmName, pidErr)
	}

	if wait {
		return waitForVM(opts)
	}

	color.Green("✔ %s VM has been launched in the background.", opts.vmName)
	color.Yellow("  To check its status, run: pvmlab vm logs %s", opts.vmName)
	return nil
}

func waitForVM(opts *vmStartOptions) error {
	color.Cyan("i Waiting for VM to become ready (this may take a few minutes)...")
	timeoutSeconds := 300 // Default to 5 minutes
	if timeoutStr := os.Getenv("PVMLAB_WAIT_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			timeoutSeconds = timeout
		}
	}
	timeoutDuration := time.Duration(timeoutSeconds) * time.Second
	sshKeyPath := filepath.Join(opts.appDir, "ssh", "vm_rsa")

	if opts.meta.Role == "provisioner" {
		if err := waiter.ForPort("localhost", opts.meta.SSHPort, timeoutDuration); err != nil {
			return err
		}
		if err := waiter.ForCloudInitProvisioner(opts.meta.SSHPort, sshKeyPath, timeoutDuration); err != nil {
			return err
		}
	} else { // target
		provisioner, err := metadata.GetProvisioner(opts.cfg)
		if err != nil {
			return fmt.Errorf("failed to find a running provisioner to wait for target VM: %w", err)
		}
		if provisioner.SSHPort == 0 {
			return fmt.Errorf("provisioner found, but it does not have a forwarded SSH port; is it running?")
		}
		if err := waiter.ForCloudInitTarget(provisioner.SSHPort, opts.meta.IP, sshKeyPath, timeoutDuration); err != nil {
			return err
		}
	}
	color.Green("✔ %s VM is ready.", opts.vmName)
	return nil
}

func getSocketVMNetClientPath() (string, error) {
	// if PVMLAB_SOCKET_VMNET_PATH is set use the client in that directory
	if path := os.Getenv("PVMLAB_SOCKET_VMNET_CLIENT"); path != "" {
		return path, nil
	}

	paths := []string{
		"/opt/socket_vmnet/bin/socket_vmnet_client",
		"/opt/homebrew/opt/socket_vmnet/bin/socket_vmnet_client",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	cmd := exec.Command("which", "socket_vmnet_client")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("socket_vmnet_client not found in standard paths or via 'which'. Please install it")
	}
	return strings.TrimSpace(string(out)), nil
}

func init() {
	vmCmd.AddCommand(vmStartCmd)
	vmStartCmd.Flags().BoolVar(&wait, "wait", false, "Wait for cloud-init to complete before exiting.")
	vmStartCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Attach to the VM's serial console.")
	vmStartCmd.Flags().StringVar(&bootOverride, "boot", "", "Override boot device (disk or pxe)")
}
