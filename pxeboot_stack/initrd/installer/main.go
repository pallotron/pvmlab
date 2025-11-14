package main

import (
	"encoding/json"
	"installer/log"
	"os"
)

func main() {
	// Catch any panics and drop to shell
	defer func() {
		if r := recover(); r != nil {
			log.Panic("%v", r)
			log.Title("Dropping to debug shell...")
			dropToShell()
		}
	}()

	log.Title("Go OS Installer started!")

	log.Step("Phase 1: Network Setup")
	netConfig, err := setupNetworking()
	if err != nil {
		log.Error("Failed to setup networking: %v", err)
		dropToShell()
		return
	}

	log.Step("Phase 2: Fetch Installer Configuration")
	if netConfig.ConfigURL == "" {
		log.Error("config_url not found in kernel command line")
		dropToShell()
		return
	}
	log.Title("Config URL: %s", netConfig.ConfigURL)
	configBytes, err := fetchURL(netConfig.ConfigURL)
	if err != nil {
		log.Error("Failed to fetch installer config: %v", err)
		dropToShell()
		return
	}
	var installerConfig InstallerConfig
	if err := json.Unmarshal(configBytes, &installerConfig); err != nil {
		log.Error("Failed to parse installer config: %v", err)
		dropToShell()
		return
	}

	log.Step("Phase 3: Fetch Cloud-Init Configuration")
	cloudInit, err := fetchCloudInitData(installerConfig.CloudInitURL)
	if err != nil {
		log.Error("Failed to fetch cloud-init data: %v", err)
		dropToShell()
		return
	}

	log.Step("Phase 4: Disk Preparation")
	diskPath, err := prepareDisk()
	if err != nil {
		log.Error("Failed to prepare disk: %v", err)
		dropToShell()
		return
	}

	log.Step("Phase 5: OS Installation")
	if err := installOS(&installerConfig); err != nil {
		log.Error("Failed to install OS: %v", err)
		dropToShell()
		return
	}

	log.Step("Phase 6: System Configuration")
	if err := configureSystem(cloudInit); err != nil {
		log.Error("Failed to configure system: %v", err)
		dropToShell()
		return
	}

	log.Step("Phase 7: Finalization")
	if err := finalize(installerConfig.RebootOnSuccess, installerConfig.Arch, installerConfig.Distro, diskPath); err != nil {
		log.Error("Failed to finalize: %v", err)
		dropToShell()
		return
	}

	os.Exit(0)
}
