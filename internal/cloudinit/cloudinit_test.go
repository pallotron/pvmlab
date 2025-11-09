package cloudinit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestCreateISO(t *testing.T) {
	// Setup temporary directory
	appDir, err := os.MkdirTemp("", "test-app")
	if err != nil {
		t.Fatalf("Failed to create temp app dir: %v", err)
	}
	defer os.RemoveAll(appDir)

	// Setup dummy SSH key
	sshDir := filepath.Join(appDir, "ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatalf("Failed to create ssh dir: %v", err)
	}
	sshKeyPath := filepath.Join(sshDir, "vm_rsa.pub")
	dummyKey := "ssh-rsa AAAA..."
	if err := os.WriteFile(sshKeyPath, []byte(dummyKey), 0644); err != nil {
		t.Fatalf("Failed to write dummy ssh key: %v", err)
	}

	testCases := []struct {
		name   string
		role   string
		vmName string
		ip     string
		ipv6   string
		mac    string
		tar    string
		image  string
	}{
		{
			name:   "Provisioner Role IPv4 only",
			role:   "provisioner",
			vmName: "prov-test",
			ip:     "192.168.1.1/24",
			ipv6:   "",
			mac:    "",
			tar:    "pxe.tar",
			image:  "pxe-image:latest",
		},
		{
			name:   "Target Role",
			role:   "target",
			vmName: "target-test",
			ip:     "192.168.1.1/24", // Provisioner IP is needed for target's DNS
			ipv6:   "",
			mac:    "52:54:00:12:34:56",
			tar:    "",
			image:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock execCommand to avoid actual execution of mkisofs
			execCommand = func(command string, args ...string) *exec.Cmd {
				assert.Equal(t, "mkisofs", command)
				cmd := exec.Command("true") // Use a command that exists and does nothing
				return cmd
			}
			isoPath := filepath.Join(appDir, tc.vmName+".iso")
			err := CreateISO(context.Background(), tc.vmName, tc.role, appDir, isoPath, tc.ip, tc.ipv6, tc.mac, tc.tar, tc.image)
			assert.NoError(t, err)

			configDir := filepath.Join(appDir, "configs", "cloud-init", tc.vmName)

			// Validate that the generated files are valid YAML
			validateYamlFile(t, filepath.Join(configDir, "meta-data"), false)
			validateYamlFile(t, filepath.Join(configDir, "user-data"), true)
			validateYamlFile(t, filepath.Join(configDir, "network-config"), false)
		})
	}
}

func TestCreateISOWithGoldenFiles(t *testing.T) {
	// Setup temporary directory
	appDir, err := os.MkdirTemp("", "test-app")
	if err != nil {
		t.Fatalf("Failed to create temp app dir: %v", err)
	}
	defer os.RemoveAll(appDir)

	// Setup dummy SSH key
	sshDir := filepath.Join(appDir, "ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatalf("Failed to create ssh dir: %v", err)
	}
	sshKeyPath := filepath.Join(sshDir, "vm_rsa.pub")
	dummyKey := "ssh-rsa AAAA..."
	if err := os.WriteFile(sshKeyPath, []byte(dummyKey), 0644); err != nil {
		t.Fatalf("Failed to write dummy ssh key: %v", err)
	}

	tc := struct {
		name   string
		role   string
		vmName string
		ip     string
		ipv6   string
		mac    string
		tar    string
		image  string
	}{
		name:   "Provisioner Role IPv4 and IPv6",
		role:   "provisioner",
		vmName: "provisioner",
		ip:     "192.168.100.1/24",
		ipv6:   "fd00:cafe:babe::1/64",
		mac:    "8a:4d:7c:5c:10:64",
		tar:    "pxeboot_stack.tar",
		image:  "pxeboot_stack:latest",
	}

	t.Run(tc.name, func(t *testing.T) {
		// Mock execCommand to avoid actual execution of mkisofs
		execCommand = func(command string, args ...string) *exec.Cmd {
			assert.Equal(t, "mkisofs", command)
			cmd := exec.Command("true") // Use a command that exists and does nothing
			return cmd
		}
		isoPath := filepath.Join(appDir, tc.vmName+".iso")
		err := CreateISO(context.Background(), tc.vmName, tc.role, appDir, isoPath, tc.ip, tc.ipv6, tc.mac, tc.tar, tc.image)
		assert.NoError(t, err)

		configDir := filepath.Join(appDir, "configs", "cloud-init", tc.vmName)
		goldenDir := filepath.Join("testdata", tc.vmName)

		compareFiles(t, filepath.Join(configDir, "meta-data"), filepath.Join(goldenDir, "meta-data"), true)
		compareFiles(t, filepath.Join(configDir, "user-data"), filepath.Join(goldenDir, "user-data"), false)
		compareFiles(t, filepath.Join(configDir, "network-config"), filepath.Join(goldenDir, "network-config"), false)
	})
}

func compareFiles(t *testing.T, generatedPath, goldenPath string, isMetaData bool) {
	t.Helper()
	generatedBytes, err := os.ReadFile(generatedPath)
	assert.NoError(t, err)

	goldenBytes, err := os.ReadFile(goldenPath)
	assert.NoError(t, err)

	if isMetaData {
		var generated, golden MetaData
		err = yaml.Unmarshal(generatedBytes, &generated)
		assert.NoError(t, err)
		err = yaml.Unmarshal(goldenBytes, &golden)
		assert.NoError(t, err)

		// Ignore comparing the two SSH public keys between generated and golden files...
		generated.PublicKeys = nil
		golden.PublicKeys = nil
		// ... but check that generated starts with "ssh-rsa "
		assert.True(
			t,
			generated.PublicKeys == nil || strings.HasPrefix(generated.PublicKeys[0], "ssh-rsa "),
			"Generated public key should start with 'ssh-rsa '",
		)

		assert.Equal(t, golden, generated)
	} else {
		assert.Equal(t, string(goldenBytes), string(generatedBytes))
	}
}

// validateYamlFile reads a file and checks if it's valid YAML.
// If hasHeader is true, it strips the cloud-config header first.
func validateYamlFile(t *testing.T, path string, hasHeader bool) {
	t.Helper()
	yamlBytes, err := os.ReadFile(path)
	assert.NoError(t, err)

	if hasHeader {
		// We only validate the YAML part, skipping the first two lines (## template: jinja, #cloud-config)
		parts := strings.SplitN(string(yamlBytes), "\n", 3)
		assert.GreaterOrEqual(t, len(parts), 3, "%s should have cloud-config header", path)
		assert.Contains(t, parts[0], "## template: jinja")
		assert.Contains(t, parts[1], "#cloud-config")
		yamlBytes = []byte(parts[2])
	}

	var obj interface{}
	err = yaml.Unmarshal(yamlBytes, &obj)
	assert.NoError(t, err, "%s should be valid YAML", path)
	assert.NotNil(t, obj, "%s should not be empty", path)
}