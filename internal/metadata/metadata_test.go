package metadata

import (
	"os"
	"path/filepath"
	"provisioning-vm-lab/internal/config"
	"reflect"
	"testing"
)

// setup sets up a temporary directory for testing and mocks GetAppDir.
func setup(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalGetAppDir := config.GetAppDirFunc
	config.GetAppDirFunc = func() (string, error) {
		return tempDir, nil
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
		config.GetAppDirFunc = originalGetAppDir
	}

	return tempDir, cleanup
}

func TestSaveLoad(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	vmName := "test-vm"
	role := "provisioner"
	ip := "192.168.1.1"
	mac := "52:54:00:12:34:56"
	pxeBootStackTar := "pxe-stack.tar"
	dockerImagesPath := "/path/to/docker/images"

	err := Save(vmName, role, ip, mac, pxeBootStackTar, dockerImagesPath)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	meta, err := Load(vmName)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	expected := &Metadata{
		Role:             role,
		IP:               ip,
		MAC:              mac,
		PxeBootStackTar:  pxeBootStackTar,
		DockerImagesPath: dockerImagesPath,
	}

	if !reflect.DeepEqual(meta, expected) {
		t.Errorf("Load() got = %v, want %v", meta, expected)
	}
}

func TestFindProvisioner(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	// Create some dummy metadata files
	Save("vm1", "target", "", "mac1", "", "")
	Save("vm2", "provisioner", "ip2", "mac2", "pxe2", "docker2")
	Save("vm3", "target", "", "mac3", "", "")

	provisionerName, err := FindProvisioner()
	if err != nil {
		t.Fatalf("FindProvisioner() failed: %v", err)
	}

	if provisionerName != "vm2" {
		t.Errorf("FindProvisioner() got = %s, want vm2", provisionerName)
	}
}

func TestDelete(t *testing.T) {
	appDir, cleanup := setup(t)
	defer cleanup()

	vmName := "vm-to-delete"
	Save(vmName, "target", "", "mac", "", "")

	err := Delete(vmName)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	vmsDir := filepath.Join(appDir, "vms")
	metaPath := filepath.Join(vmsDir, vmName+".json")
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Errorf("Metadata file for %s was not deleted", vmName)
	}
}

func TestGetAll(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	// Create some dummy metadata files
	Save("vm1", "target", "", "mac1", "", "")
	Save("vm2", "provisioner", "ip2", "mac2", "pxe2", "docker2")

	allMeta, err := GetAll()
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
