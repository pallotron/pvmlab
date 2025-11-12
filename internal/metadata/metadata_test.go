package metadata

import (
	"os"
	"path/filepath"
	"pvmlab/internal/config"
	"reflect"
	"testing"
)

// setup sets up a temporary directory for testing and returns a config object.
func setup(t *testing.T) (*config.Config, func()) {
	tempDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	cfg.SetHomeDir(tempDir)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return cfg, cleanup
}

func TestSaveLoad(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm"
	role := "provisioner"
	ip := "192.168.1.1"
	subnet := "192.168.1.0/24"
	mac := "52:54:00:12:34:56"
	pxeBootStackTar := "pxe-stack.tar"
	dockerImagesPath := "/path/to/docker/images"
	vmsPath := "/path/to/vms"

	err := Save(cfg, vmName, role, "aarch64", ip, subnet, "", "", mac, pxeBootStackTar, dockerImagesPath, vmsPath, "", "", "", 0, false, "")
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	meta, err := Load(cfg, vmName)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	want := &Metadata{
		Name:             vmName,
		Role:             "provisioner",
		Arch:             "aarch64",
		IP:               "192.168.1.1",
		Subnet:           "192.168.1.0/24",
		MAC:              "52:54:00:12:34:56",
		PxeBootStackTar:  "pxe-stack.tar",
		DockerImagesPath: "/path/to/docker/images",
		VMsPath:          "/path/to/vms",
		SSHKey:           "",
	}

	if !reflect.DeepEqual(meta, want) {
		t.Errorf("Load() got = %v, want %v", meta, want)
	}
}

func TestFindProvisioner(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()
	if err := Save(cfg, "vm1", "target", "aarch64", "", "", "", "", "mac1", "", "", "", "", "", "", 0, false, ""); err != nil {
		t.Fatalf("Save() failed for vm1: %v", err)
	}
	if err := Save(cfg, "vm2", "provisioner", "aarch64", "ip2", "subnet2", "", "", "mac2", "pxe2", "docker2", "", "", "", "", 45678, false, ""); err != nil {
		t.Fatalf("Save() failed for vm2: %v", err)
	}
	if err := Save(cfg, "vm3", "target", "aarch64", "", "", "", "", "mac3", "", "", "", "", "", "", 0, false, ""); err != nil {
		t.Fatalf("Save() failed for vm3: %v", err)
	}

	provisionerName, err := FindProvisioner(cfg)
	if err != nil {
		t.Fatalf("FindProvisioner() failed: %v", err)
	}

	if provisionerName != "vm2" {
		t.Errorf("FindProvisioner() got = %s, want vm2", provisionerName)
	}
}

func TestDelete(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	vmName := "vm-to-delete"
	if err := Save(cfg, vmName, "target", "aarch64", "", "", "", "", "mac", "", "", "", "", "", "", 0, false, ""); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	err := Delete(cfg, vmName)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	vmsDir := filepath.Join(cfg.GetAppDir(), "vms")
	metaPath := filepath.Join(vmsDir, vmName+".json")
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Errorf("Metadata file for %s was not deleted", vmName)
	}
}

func TestGetAll(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	if err := Save(cfg, "vm1", "target", "aarch64", "", "", "", "", "mac1", "", "", "", "", "", "", 0, false, ""); err != nil {
		t.Fatalf("Save() failed for vm1: %v", err)
	}
	if err := Save(cfg, "vm2", "provisioner", "aarch64", "ip2", "subnet2", "", "", "mac2", "pxe2", "docker2", "", "", "", "", 45678, false, ""); err != nil {
		t.Fatalf("Save() failed for vm2: %v", err)
	}

	allMeta, err := GetAll(cfg)
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}

	if len(allMeta) != 2 {
		t.Errorf("GetAll() got %d VMs, want 2", len(allMeta))
	}

	if allMeta["vm1"].Role != "target" {
		t.Errorf("GetAll() metadata for vm1 is incorrect")
	}
	if allMeta["vm2"].Role != "provisioner" {
		t.Errorf("GetAll() metadata for vm2 is incorrect")
	}
}

func TestGetProvisioner(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	// Scenario 1: Provisioner exists
	if err := Save(cfg, "vm1", "target", "aarch64", "", "", "", "", "mac1", "", "", "", "", "", "", 0, false, ""); err != nil {
		t.Fatalf("Save() failed for vm1: %v", err)
	}
	if err := Save(cfg, "vm2", "provisioner", "aarch64", "ip2", "subnet2", "", "", "mac2", "pxe2", "docker2", "", "", "", "", 45678, false, ""); err != nil {
		t.Fatalf("Save() failed for vm2: %v", err)
	}

	provisioner, err := GetProvisioner(cfg)
	if err != nil {
		t.Fatalf("GetProvisioner() failed: %v", err)
	}
	if provisioner.Name != "vm2" {
		t.Errorf("GetProvisioner() got = %s, want vm2", provisioner.Name)
	}

	// Scenario 2: No provisioner
	cleanup()
	cfg, cleanup = setup(t)
	defer cleanup()
	if err := Save(cfg, "vm1", "target", "aarch64", "", "", "", "", "mac1", "", "", "", "", "", "", 0, false, ""); err != nil {
		t.Fatalf("Save() failed for vm1: %v", err)
	}

	_, err = GetProvisioner(cfg)
	if err == nil {
		t.Errorf("GetProvisioner() expected an error, but got none")
	}
}

func TestFindVM(t *testing.T) {
	cfg, cleanup := setup(t)
	defer cleanup()

	// Scenario 1: VM exists
	if err := Save(cfg, "vm1", "target", "aarch64", "", "", "", "", "mac1", "", "", "", "", "", "", 0, false, ""); err != nil {
		t.Fatalf("Save() failed for vm1: %v", err)
	}

	vmName, err := FindVM(cfg, "vm1")
	if err != nil {
		t.Fatalf("FindVM() failed: %v", err)
	}
	if vmName != "vm1" {
		t.Errorf("FindVM() got = %s, want vm1", vmName)
	}

	// Scenario 2: VM does not exist
	vmName, err = FindVM(cfg, "non-existent-vm")
	if err != nil {
		t.Fatalf("FindVM() for non-existent vm failed: %v", err)
	}
	if vmName != "" {
		t.Errorf("FindVM() for non-existent vm got = %s, want empty string", vmName)
	}
}