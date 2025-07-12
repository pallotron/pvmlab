UUID := $(shell if [ -f uuidgen ]; then cat uuidgen; else /usr/bin/uuidgen > uuidgen && cat uuidgen; fi)
SOCKET_PATH := $(shell brew --prefix)/var/run/socket_vmnet

.PHONY: setup provisioner-vm-disk provisioner-vm-cloudconfig.iso run-provisioner-vm target-vm-disk target-vm-cloudconfig.iso run-target-vm clean create-runtime-dir start-service stop-service

start-service:
	@echo "Starting socket_vmnet service..."
	@sudo brew services start socket_vmnet

stop-service:
	@echo "Stopping socket_vmnet service..."
	@sudo brew services stop socket_vmnet



setup: generate-ssh-key
	@if ! command -v brew &> /dev/null; then \
		echo "Homebrew not found. Please install Homebrew first: https://brew.sh/"; \
		exit 1; \
	fi
	@if ! brew list cdrtools &> /dev/null; then \
		echo "Installing cdrtools (for genisoimage)..."; \
		brew install cdrtools; \
	else \
		echo "cdrtools is already installed."; \
	fi
	@if ! brew list socat &> /dev/null; then \
		echo "Installing socat..."; \
		brew install socat; \
	else \
		echo "socat is already installed."; \
	fi
	@if ! brew list socket_vmnet &> /dev/null; then \
		echo "Installing socket_vmnet..."; \
		brew install socket_vmnet; \
	else \
		echo "socket_vmnet is already installed."; \
	fi
	mkdir -p logs
	mkdir -p cloud_images

generate-ssh-key:
	@if [ ! -f .ssh/vm_rsa ]; then \
		echo "Generating SSH key..."; \
		mkdir -p .ssh; \
		ssh-keygen -t rsa -b 2048 -f .ssh/vm_rsa -N ""; \
	else \
		echo "SSH key already exists."; \
	fi

download-cloud-images:
	@if [ ! -f cloud_images/ubuntu-24.04-server-cloudimg-arm64.img ]; then \
		echo "Downloading aarch64 Ubuntu cloud image..."; \
		curl -L -o cloud_images/ubuntu-24.04-server-cloudimg-arm64.img https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-arm64.img; \
	else \
		echo "aarch64 Ubuntu cloud image already exists."; \
	fi
	@if [ ! -f cloud_images/ubuntu-24.04-server-cloudimg-amd64.img ]; then \
		echo "Downloading amd64 Ubuntu cloud image..."; \
		curl -L -o cloud_images/ubuntu-24.04-server-cloudimg-amd64.img https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img; \
	else \
		echo "amd64 Ubuntu cloud image already exists."; \
	fi

######## PROVISIONER VM ########

provisioner-vm-disk: download-cloud-images
	qemu-img create -f qcow2 -b cloud_images/ubuntu-24.04-server-cloudimg-arm64.img -F qcow2 provisioner-vm-aarch64.qcow2
	qemu-img resize provisioner-vm-aarch64.qcow2 10G

provisioner-vm-cloudconfig.iso: 
	$(shell brew --prefix cdrtools)/bin/mkisofs -o provisioner-vm-cloudconfig.iso -volid cidata -joliet -rock cloudconfig-provisioner-vm/

run-provisioner-vm: provisioner-vm-disk provisioner-vm-cloudconfig.iso
	$(shell brew --prefix socket_vmnet)/bin/socket_vmnet_client $(SOCKET_PATH) \
	$(shell brew --prefix qemu)/bin/qemu-system-aarch64 -M virt -accel hvf -cpu cortex-a72 -smp 2 -m 2048 \
		-daemonize \
		-display none \
		-pidfile provisioner-vm.pid \
		-monitor unix:provisioner-vm.monitor,server,nowait \
		-drive if=pflash,format=raw,readonly=on,file=$(shell brew --prefix qemu)/share/qemu/edk2-aarch64-code.fd \
		-drive file=provisioner-vm-aarch64.qcow2,index=0,format=qcow2,media=disk \
		-drive file=provisioner-vm-cloudconfig.iso,index=1,media=cdrom \
		-chardev file,id=charlog,path=logs/provisioner-vm-console.log \
		-serial chardev:charlog \
		-netdev user,id=net0,hostfwd=tcp::2222-:22 \
		-device virtio-net-pci,netdev=net0,mac=00:11:DE:AD:BE:EF \
		-netdev socket,id=net1,fd=3 \
		-device virtio-net-pci,netdev=net1,mac=00:00:DE:AD:BE:EF

stop-provisioner-vm:
	@if [ -e provisioner-vm.monitor ]; then \
		echo "Stopping provisioner VM gracefully..."; \
		echo "system_powerdown" | socat -T 1 - unix-connect:provisioner-vm.monitor; \
		rm provisioner-vm.monitor; \
	else \
		echo "Provisioner VM not running or monitor socket not found."; \
	fi

tail-provisioner-vm-logs:
	tail -f logs/provisioner-vm-console.log

shell-provisioner-vm:
	ssh -i .ssh/vm_rsa -p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null ubuntu@localhost

clean-provisioner-vm: stop-provisioner-vm
	rm -f provisioner-vm-*.qcow2 provisioner-vm-cloudconfig.iso provisioner-vm.pid provisioner-vm.monitor

######## TARGET VM ########

target-vm-disk: download-cloud-images
	qemu-img create -f qcow2 -b cloud_images/ubuntu-24.04-server-cloudimg-amd64.img -F qcow2 target-vm-x86_64.qcow2
	qemu-img resize target-vm-x86_64.qcow2 10G

target-vm-cloudconfig.iso:
	$(shell brew --prefix cdrtools)/bin/mkisofs -o target-vm-cloudconfig.iso -volid cidata -joliet -rock cloudconfig-target-vm/

run-target-vm: target-vm-disk target-vm-cloudconfig.iso
	$(shell brew --prefix socket_vmnet)/bin/socket_vmnet_client $(SOCKET_PATH) \
	$(shell brew --prefix qemu)/bin/qemu-system-x86_64 -m 2048 \
		-daemonize \
		-display none \
		-pidfile target-vm.pid \
		-monitor unix:target-vm.monitor,server,nowait \
		-drive if=pflash,format=raw,readonly=on,file=$(shell brew --prefix qemu)/share/qemu/edk2-x86_64-code.fd \
		-drive file=target-vm-x86_64.qcow2,index=0,format=qcow2,media=disk \
		-drive file=target-vm-cloudconfig.iso,index=1,media=cdrom \
		-chardev file,id=charlog,path=logs/target-vm-console.log \
		-serial chardev:charlog \
		-netdev socket,id=net1,fd=3 \
		-device virtio-net-pci,netdev=net1,mac=00:33:DE:AD:BE:EF
		
stop-target-vm:
	@if [ -e target-vm.monitor ]; then \
		echo "Stopping target VM gracefully..."; \
		echo "system_powerdown" | socat -T 1 - unix-connect:target-vm.monitor; \
		rm target-vm.monitor; \
	else \
		echo "Target VM not running or monitor socket not found."; \
	fi

tail-target-vm-logs:
	tail -f logs/target-vm-console.log

shell-target-vm:
	ssh -i .ssh/vm_rsa -p 2233 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null ubuntu@localhost
		
clean-target-vm: stop-target-vm
	rm -f target-vm-*.qcow2 target-vm-cloudconfig.iso target-vm.pid target-vm.monitorclean-all: clean-provisioner-vm clean-target-vm

clean-all: clean-provisioner-vm clean-target-vm stop-service
	rm -f *-vm-*.qcow2 *-vm-cloudconfig.iso cloud_images/*
