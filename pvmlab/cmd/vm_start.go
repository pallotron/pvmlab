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

		opts, err := gatherVMInfo(args[0])
		if err != nil {
			return err
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
	imagePath := filepath.Join(opts.appDir, "images", config.UbuntuARMImageName)
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("ubuntu cloud image not found. Please run `pvmlab setup` to download it")
	}
	vmDiskPath := filepath.Join(opts.appDir, "vms", opts.vmName+".qcow2")
	if _, err := os.Stat(vmDiskPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("VM disk not found for '%s'. Please run 'pvmlab vm create %s' first", opts.vmName, opts.vmName)
	}
	isoPath := filepath.Join(opts.appDir, "configs", "cloud-init", opts.vmName+".iso")
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cloud-init ISO for '%s' not found. Please create the VM first", opts.vmName)
	}

	return opts, nil
}

func buildQEMUArgs(opts *vmStartOptions) ([]string, error) {
	pidPath := filepath.Join(opts.appDir, "pids", opts.vmName+".pid")
	monitorPath := filepath.Join(opts.appDir, "monitors", opts.vmName+".sock")
	logPath := filepath.Join(opts.appDir, "logs", opts.vmName+".log")
	vmDiskPath := filepath.Join(opts.appDir, "vms", opts.vmName+".qcow2")
	isoPath := filepath.Join(opts.appDir, "configs", "cloud-init", opts.vmName+".iso")

	qemuArgs := []string{
		"qemu-system-aarch64",
		"-M", "virt",
		"-smp", "2",
		"-drive", "if=pflash,format=raw,readonly=on,file=/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", vmDiskPath),
		"-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", isoPath),
		"-pidfile", pidPath,
		"-monitor", fmt.Sprintf("unix:%s,server,nowait", monitorPath),
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
		if err := metadata.Save(opts.cfg, opts.vmName, opts.meta.Role, opts.meta.IP, opts.meta.Subnet, opts.meta.IPv6, opts.meta.SubnetV6, opts.meta.MAC, opts.meta.PxeBootStackTar, opts.meta.DockerImagesPath, opts.meta.VMsPath, opts.meta.SSHPort); err != nil {
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
			"-device", "virtio-net-pci,netdev=net0",
			"-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22,ipv6=on,ipv4=on,ipv6-net=fd00::/64", opts.meta.SSHPort),
			"-device", fmt.Sprintf("virtio-net-pci,netdev=net1,mac=%s", opts.meta.MAC),
			"-netdev", "socket,id=net1,fd=3",
			"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_docker_images,security_model=passthrough", finalDockerImagesPath),
			"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_vms,security_model=passthrough", finalVMsPath),
		)
	} else { // target
		qemuArgs = append(qemuArgs, "-m", "2048", "-device", fmt.Sprintf("virtio-net-pci,netdev=net0,mac=%s", opts.meta.MAC), "-netdev", "socket,id=net0,fd=3")
	}

	if os.Getenv("CI") != "true" {
		qemuArgs = append(qemuArgs, "-cpu", "host", "-accel", "hvf")
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

	if os.Getenv("PVMLAB_DEBUG") == "true" {
		color.Yellow("--- QEMU Command ---")
		for _, arg := range finalCmd {
			fmt.Println("  " + arg)
		}
		color.Yellow("--------------------\n")
	}

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
}