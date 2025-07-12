UUID := $(shell if [ -f uuidgen ]; then cat uuidgen; else /usr/bin/uuidgen > uuidgen && cat uuidgen; fi)

.PHONY: setup provisioner-vm-disk provisioner-vm-cloudconfig.iso run-provisioner-vm target-vm-disk target-vm-cloudconfig.iso run-target-vm clean create-runtime-dir


setup:
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
	@if [ ! -f /etc/sudoers.d/vm_lab ]; then \
		echo "Creating sudoers file for vm_lab..."; \
		echo "$$USER ALL=(ALL) NOPASSWD: $(shell brew --prefix qemu)/bin/qemu-system-aarch64, $(shell brew --prefix qemu)/bin/qemu-system-x86_64" | sudo tee /etc/sudoers.d/vm_lab; \
	else \
		echo "sudoers file for vm_lab already exists."; \
	fi

download-cloud-images:
	mkdir -p cloud_images
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
	sudo $(shell brew --prefix qemu)/bin/qemu-system-aarch64 -M virt -accel hvf -cpu cortex-a72 -smp 2 -m 2048 \
		-drive if=pflash,format=raw,readonly=on,file=$(shell brew --prefix qemu)/share/qemu/edk2-aarch64-code.fd \
		-drive file=provisioner-vm-aarch64.qcow2,index=0,format=qcow2,media=disk \
		-drive file=provisioner-vm-cloudconfig.iso,index=1,media=cdrom \
		-netdev user,id=net0,hostfwd=tcp::2222-:22 \
		-device virtio-net-pci,netdev=net0,mac=00:11:DE:AD:BE:EF \
		-netdev vmnet-host,id=net1,start-address=192.168.2.1,end-address=192.168.2.254,subnet-mask=255.255.255.0,net-uuid=$(UUID) \
		-device virtio-net-pci,netdev=net1,mac=00:00:DE:AD:BE:EF \
		-nographic

clean-provisioner-vm:
	rm -f provisioner-vm-*.qcow2 provisioner-vm-cloudconfig.iso

######## TARGET VM ########

target-vm-disk: download-cloud-images
	qemu-img create -f qcow2 -b cloud_images/ubuntu-24.04-server-cloudimg-amd64.img -F qcow2 target-vm-x86_64.qcow2
	qemu-img resize target-vm-x86_64.qcow2 10G

target-vm-cloudconfig.iso:
	$(shell brew --prefix cdrtools)/bin/mkisofs -o target-vm-cloudconfig.iso -volid cidata -joliet -rock cloudconfig-target-vm/

run-target-vm: target-vm-disk target-vm-cloudconfig.iso
	sudo $(shell brew --prefix qemu)/bin/qemu-system-x86_64 -m 2048 \
		-drive if=pflash,format=raw,readonly=on,file=$(shell brew --prefix qemu)/share/qemu/edk2-x86_64-code.fd \
		-drive file=target-vm-x86_64.qcow2,index=0,format=qcow2,media=disk \
		-drive file=target-vm-cloudconfig.iso,index=1,media=cdrom \
		-netdev vmnet-host,id=net1,start-address=192.168.2.1,end-address=192.168.2.254,subnet-mask=255.255.255.0,net-uuid=$(UUID) \
		-device virtio-net-pci,netdev=net1,mac=00:33:DE:AD:BE:EF \
		-nographic

clean-target-vm:
	rm -f target-vm-*.qcow2 target-vm-cloudconfig.iso

clean-all:
	rm -f *-vm-*.qcow2 *-vm-cloudconfig.iso cloud_images/*
