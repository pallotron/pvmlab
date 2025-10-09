.DEFAULT_GOAL := all

all: build.socket_vmnet install.socket_vmnet install.launchd
clean: clean.socket_vmnet
	make -C socket_vmnet clean

build.socket_vmnet:
	logger echo "Building socket_vmnet..."
	make -C socket_vmnet all

clean.socket_vmnet:
	logger echo "Cleaning socket_vmnet..."
	make -C socket_vmnet clean

install.socket_vmnet: build.socket_vmnet
	logger "Installing socket_vmnet... sudo access might be required..."
	sudo make -C socket_vmnet install.bin

define load_launchd
	# Hint: try `launchctl enable system/$(1)` if the `launchctl bootstrap` command below fails
	logger "Stopping launchd service $(1) if running..."
	sudo launchctl bootout system "/Library/LaunchDaemons/$(1).plist" || true
	logger "Installing launchd service for socket_vmnet in /Library/LaunchDaemons/$(1).plist"
	sudo cp launchd/$(1).plist  /Library/LaunchDaemons/$(1).plist
	sudo launchctl bootstrap system "/Library/LaunchDaemons/$(1).plist" || true
	sudo launchctl enable system/$(1)
	sudo launchctl kickstart -kp system/$(1)
	sudo launchctl list $(1)
endef

define unload_launchd
	logger "Uninstalling launchd service for ${1}"
	sudo launchctl bootout system "$(DESTDIR)/Library/LaunchDaemons/$(1).plist" || true
	sudo rm -f /Library/LaunchDaemons/$(1).plist
endef

install.launchd:
	$(call load_launchd,io.github.pallotron.pvmlab.socket_vmnet)

uninstall.launchd:
	$(call unload_launchd,io.github.pallotron.pvmlab.socket_vmnet)


.PHONY: all socket_vmnet clean install.socket_vmnet install.socket_vmnet.launchd
