.DEFAULT_GOAL := install

.PHONY: isntall socket_vmnet clean install.socket_vmnet install.socket_vmnet.launchd install.completions test integration.test uninstall uninstall-pvmlab uninstall.socket_vmnet uninstall.launchd uninstall.completions uninstall-pxeboot-stack-container

install: install-pvmlab build.socket_vmnet install.completions install.socket_vmnet install.launchd build-pxeboot-stack-container
	@echo "🚀"
clean: clean.socket_vmnet
	@make -C socket_vmnet clean

install-pvmlab:
	@go install ./pvmlab

build.socket_vmnet:
	logger echo "Building socket_vmnet..."
	@make -C socket_vmnet all

clean.socket_vmnet:
	logger echo "Cleaning socket_vmnet..."
	@make -C socket_vmnet clean

install.socket_vmnet: build.socket_vmnet
	logger "Installing socket_vmnet... sudo access might be required..."
	@sudo make -C socket_vmnet install.bin
	@sudo chown root:staff /opt/socket_vmnet/bin/socket_vmnet_client

build-pxeboot-stack-container:
	@make -C pxeboot_stack all

define load_launchd
	@if pvmlab vm list 2>/dev/null | grep -q Running; then \
		echo "Error: Cannot reload launchd service while VMs are running. Please stop all VMs first."; \
		exit 1; \
	fi
	logger "Stopping launchd service $(1) if running..."
	@sudo launchctl bootout system "/Library/LaunchDaemons/$(1).plist" || true
	@logger "Installing launchd service for socket_vmnet in /Library/LaunchDaemons/$(1).plist"
	@sudo cp launchd/$(1).plist  /Library/LaunchDaemons/$(1).plist
	@sudo launchctl bootstrap system "/Library/LaunchDaemons/$(1).plist" || true
	@sudo launchctl enable system/$(1)
	@sudo launchctl kickstart -kp system/$(1)
	@sudo launchctl list $(1)
endef

define unload_launchd
	@if pvmlab vm list 2>/dev/null | grep -q Running; then \
		echo "Error: Cannot unload launchd service while VMs are running. Please stop all VMs first."; \
		exit 1; \
	fi
	logger "Uninstalling launchd service for ${1}"
	@sudo launchctl bootout system "$(DESTDIR)/Library/LaunchDaemons/$(1).plist" || true
	@sudo rm -f /Library/LaunchDaemons/$(1).plist
endef

install.launchd:
	logger "Installing launchd wrapper script..."
	@sudo mkdir -p /opt/pvmlab/libexec/
	@sudo cp launchd/socket_vmnet_wrapper.sh /opt/pvmlab/libexec/socket_vmnet_wrapper.sh
	@sudo chmod +x /opt/pvmlab/libexec/socket_vmnet_wrapper.sh
	$(call load_launchd,io.github.pallotron.pvmlab.socket_vmnet)

uninstall.launchd:
	logger "Uninstalling launchd wrapper script..."
	@sudo rm -f /opt/pvmlab/libexec/socket_vmnet_wrapper.sh
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


release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release VERSION=vX.Y.Z"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Git working directory is not clean. Please commit or stash your changes."; \
		exit 1; \
	fi
	@git tag -a $(VERSION) -m "Version $(VERSION)"
	@git push origin $(VERSION)
	@echo "🎉"

uninstall: uninstall-pvmlab uninstall.socket_vmnet uninstall.launchd uninstall.completions uninstall-pxeboot-stack-container
	@echo "Uninstall complete."
	@echo "👋"

uninstall-pvmlab:
	@echo "stopping all VMs..."
	@echo "Uninstalling pvmlab binary..."
	@if [ -f "$$(go env GOPATH)/bin/pvmlab" ]; then \
		pvmlab clean --purge \
		rm "$$(go env GOPATH)/bin/pvmlab"; \
		echo "Removed $$(go env GOPATH)/bin/pvmlab"; \
	else \
		echo "pvmlab binary not found, skipping."; \
	fi

uninstall.socket_vmnet:
	@echo "Uninstalling socket_vmnet..."
	@if [ -f "/opt/socket_vmnet/bin/socket_vmnet" ]; then \
		sudo make -C socket_vmnet uninstall; \
	else \
		echo "socket_vmnet not found, skipping."; \
	fi

uninstall.completions:
	@echo "Uninstalling shell completions..."
	@if command -v brew &> /dev/null; then \
		BASH_COMPLETION_DIR=$$(brew --prefix)/etc/bash_completion.d; \
		if [ -f "$$BASH_COMPLETION_DIR/pvmlab" ]; then \
			rm "$$BASH_COMPLETION_DIR/pvmlab"; \
			echo "Bash completion removed from $$BASH_COMPLETION_DIR"; \
		else \
			echo "Bash completion not found, skipping."; \
		fi; \
	else \
		echo "brew not found, skipping bash completion uninstallation"; \
	fi
	@if command -v zsh &> /dev/null; then \
		UNINSTALLED=false; \
		for dir in $$(zsh -i -c 'echo $$fpath'); do \
			if [ -f "$$dir/_pvmlab" ]; then \
				if sudo rm "$$dir/_pvmlab"; then \
					echo "Zsh completion removed from $$dir"; \
					UNINSTALLED=true; \
					break; \
				fi; \
			fi; \
		done; \
		if [ "$$UNINSTALLED" = "false" ]; then \
			echo "Could not find zsh completion file to remove."; \
		fi; \
	else \
		echo "zsh not found, skipping zsh completion uninstallation"; \
	fi

uninstall-pxeboot-stack-container:
	@echo "Uninstalling pxeboot stack container..."
	@make -C pxeboot_stack clean


test: 
	RUN_INTEGRATION_TESTS=false go test -v ./...

integration.test: 
	@make -C pxeboot_stack tar
	@RUN_INTEGRATION_TESTS=true go test -v ./tests/integration/...

integration.test.ssh.provisioner:
	@PVMLAB_HOME=$$(./tests/find-test-pvmlab-home.sh) ./build/pvmlab_test vm shell test-provisioner

integration.test.ssh.client:
	@PVMLAB_HOME=$$(./tests/find-test-pvmlab-home.sh) ./build/pvmlab_test vm shell test-client
