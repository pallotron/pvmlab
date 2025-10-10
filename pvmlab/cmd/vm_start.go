package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"provisioning-vm-lab/internal/logwatcher"
	"provisioning-vm-lab/internal/metadata"
	"provisioning-vm-lab/internal/pidfile"
	"provisioning-vm-lab/internal/socketvmnet"
	"strings"
	"time"

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
	Run: func(cmd *cobra.Command, args []string) {
		vmName := args[0]

		running, err := pidfile.IsRunning(vmName)
		if err != nil {
			fmt.Println("Error checking VM status:", err)
			os.Exit(1)
		}
		if running {
			fmt.Printf("VM '%s' is already running.\n", vmName)
			os.Exit(1)
		}

		meta, err := metadata.Load(vmName)
		if err != nil {
			fmt.Println("Error loading VM metadata:", err)
			os.Exit(1)
		}

		appDir, err := config.GetAppDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		vmDiskPath := filepath.Join(appDir, "vms", vmName+".qcow2")
		if _, err := os.Stat(vmDiskPath); os.IsNotExist(err) {
			fmt.Printf("VM disk not found for '%s'. Please run 'pvmlab vm create %s' first.\n", vmName, vmName)
			os.Exit(1)
		}

		isoPath := filepath.Join(appDir, "configs", "cloud-init", vmName+".iso")
		if _, err := os.Stat(isoPath); os.IsNotExist(err) {
			fmt.Printf("Cloud-init ISO for '%s' not found. Please create the VM first.\n", vmName)
			os.Exit(1)
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
		var qemuArgs []string
		if meta.Role == "provisioner" {
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

			qemuArgs = []string{
				"qemu-system-aarch64",
				"-M", "virt",
				"-cpu", "host",
				"-accel", "hvf",
				"-m", "4096",
				"-smp", "2",
				"-drive", "if=pflash,format=raw,readonly=on,file=/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
				"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", vmDiskPath),
				"-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", isoPath),
				"-device", "virtio-net-pci,netdev=net0",
				"-netdev", "user,id=net0,hostfwd=tcp::2222-:22",
				"-device", fmt.Sprintf("virtio-net-pci,netdev=net1,mac=%s", meta.MAC),
				"-netdev", "socket,id=net1,fd=3",
				"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_docker_images,security_model=passthrough", finalDockerImagesPath),
				"-virtfs", fmt.Sprintf("local,path=%s,mount_tag=host_share_vms,security_model=passthrough", finalVMsPath),
				"-display", "none",
				"-daemonize",
				"-pidfile", pidPath,
				"-monitor", fmt.Sprintf("unix:%s,server,nowait", monitorPath),
				"-serial", fmt.Sprintf("file:%s", logPath),
			}
		} else { // target
			qemuArgs = []string{
				"qemu-system-aarch64",
				"-M", "virt",
				"-cpu", "host",
				"-accel", "hvf",
				"-m", "2048",
				"-smp", "2",
				"-drive", "if=pflash,format=raw,readonly=on,file=/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
				"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", vmDiskPath),
				"-drive", fmt.Sprintf("file=%s,format=raw,if=virtio", isoPath),
				"-device", fmt.Sprintf("virtio-net-pci,netdev=net0,mac=%s", meta.MAC),
				"-netdev", "socket,id=net0,fd=3",
				"-display", "none",
				"-daemonize",
				"-pidfile", pidPath,
				"-monitor", fmt.Sprintf("unix:%s,server,nowait", monitorPath),
				"-serial", fmt.Sprintf("file:%s", logPath),
			}
		}

		socketPath, err := socketvmnet.GetSocketPath()
		if err != nil {
			fmt.Println("Error getting socket_vmnet path:", err)
			os.Exit(1)
		}

		clientPath, err := getSocketVMNetClientPath()
		if err != nil {
			fmt.Println("Error getting socket_vmnet_client path:", err)
			os.Exit(1)
		}

		finalCmd := []string{clientPath, socketPath}
		finalCmd = append(finalCmd, qemuArgs...)

		fmt.Println("Executing command:", strings.Join(finalCmd, " "))
		cmdRun := exec.Command(finalCmd[0], finalCmd[1:]...)
		output, err := cmdRun.CombinedOutput()
		if err != nil {
			fmt.Printf("Error starting VM '%s': %v\n", vmName, err)
			fmt.Println("QEMU output:")
			fmt.Println(string(output))
			os.Exit(1)
		}

		// The command can succeed even if the daemon fails.
		// A short sleep allows the PID file to be created.
		time.Sleep(1 * time.Second)
		running, pidErr := pidfile.IsRunning(vmName)
		if pidErr != nil || !running {
			fmt.Printf("VM command executed, but VM '%s' is not running.\n", vmName)
			if pidErr != nil {
				fmt.Printf("Error checking PID file: %v\n", pidErr)
			}
			fmt.Println("QEMU output (if any):")
			fmt.Println(string(output))
			os.Exit(1)
		}

		fmt.Printf("%s VM started successfully.\n", vmName)
		// TODO: switch to using QEMU guest agent and unix socket
		if wait {
			if err := logwatcher.WaitForMessage(vmName, "cloud-config.target", 5*time.Minute); err != nil {
				fmt.Println(err)
				// We don't exit here, as the VM is running, but the wait failed.
			} else {
				// The wait was successful, so we can exit cleanly.
				os.Exit(0)
			}
		}
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
