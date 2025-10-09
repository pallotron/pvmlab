.DEFAULT_GOAL := all

all: install-pvmlab build.socket_vmnet install.completions install.socket_vmnet install.launchd
clean: clean.socket_vmnet
	make -C socket_vmnet clean

install-pvmlab:
	go install ./pvmlab

build.socket_vmnet:
	logger echo "Building socket_vmnet..."
	make -C socket_vmnet all

clean.socket_vmnet:
	logger echo "Cleaning socket_vmnet..."
	make -C socket_vmnet clean

install.socket_vmnet: build.socket_vmnet
	logger "Installing socket_vmnet... sudo access might be required..."
	sudo make -C socket_vmnet install.bin

build-pxeboot-stack-container:
	make -C pxeboot_stack all

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

install.completions: install-pvmlab
	@echo "Installing shell completions..."
	@if command -v brew &> /dev/null; then \
		BASH_COMPLETION_DIR=$$(brew --prefix)/etc/bash_completion.d; \
		mkdir -p $$BASH_COMPLETION_DIR; \
		pvmlab completion bash > $$BASH_COMPLETION_DIR/pvmlab; \
		echo "Bash completion installed in $$BASH_COMPLETION_DIR"; \
		echo "Run 'source $$BASH_COMPLETION_DIR/pvmlab' to load it."; \
	else \
		echo "brew not found, skipping bash completion installation"; \
	fi
	@if command -v zsh &> /dev/null; then \
		INSTALLED=false; \
		for dir in $$(zsh -i -c 'echo $$fpath'); do \
			echo "Trying $$dir"; \
			if [ -d "$$dir" ]; then \
				if pvmlab completion zsh | sudo tee "$$dir/_pvmlab" >/dev/null ; then \
					echo "Zsh completion installed in $$dir"; \
					echo "Run 'source $$dir/_pvmlab' to load it."; \
					INSTALLED=true; \
					break; \
				fi; \
			fi; \
		done; \
		if [ "$$INSTALLED" = "false" ]; then \
			echo "Could not install zsh completion in any fpath directory."; \
		fi; \
	else \
		echo "zsh not found, skipping zsh completion installation"; \
	fi


.PHONY: all socket_vmnet clean install.socket_vmnet install.socket_vmnet.launchd install.completions
