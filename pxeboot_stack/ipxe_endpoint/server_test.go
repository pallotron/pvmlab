package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBootHandler(t *testing.T) {
	// 1. Setup: Create a temporary directory for VM definitions
	tmpDir, err := os.MkdirTemp("", "pvmlab-test-vms")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock VM JSON file
	vmMAC := "52:54:00:12:34:56"
	vmJSON := fmt.Sprintf(`{
		"name": "test-vm",
		"arch": "aarch64",
		"distro": "ubuntu-24.04",
		"mac": "%s"
	}`, vmMAC)
	vmFile := filepath.Join(tmpDir, "test-vm.json")
	if err := os.WriteFile(vmFile, []byte(vmJSON), 0644); err != nil {
		t.Fatalf("Failed to write mock VM JSON: %v", err)
	}

	// Create a mock iPXE template file
	templateContent := `#!ipxe
echo Booting {{.Name}} ({{.Arch}})
set base-url http://192.168.100.1/images/{{.Distro}}
kernel ${base-url}/vmlinuz quiet autoinstall
initrd ${base-url}/initrd
boot
`
	templateFile := filepath.Join(tmpDir, "boot.ipxe.go.template")
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write mock template: %v", err)
	}

	// Create a server instance with the temp directory
	server := &httpServer{
		vmsDir:       tmpDir,
		templatePath: templateFile,
	}

	// 2. Test Cases
	tests := []struct {
		name           string
		mac            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Successful Boot",
			mac:            vmMAC,
			expectedStatus: http.StatusOK,
			expectedBody: `#!ipxe
echo Booting test-vm (aarch64)
set base-url http://192.168.100.1/images/ubuntu-24.04
kernel ${base-url}/vmlinuz quiet autoinstall
initrd ${base-url}/initrd
boot
`,
		},
		{
			name:           "VM Not Found",
			mac:            "00:00:00:00:00:00",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "VM with MAC 00:00:00:00:00:00 not found\n",
		},
		{
			name:           "MAC Parameter Missing",
			mac:            "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "mac query parameter is required\n",
		},
		{
			name:           "Template Not Found",
			mac:            vmMAC,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 3. Execution: Create a request and record the response
			url := "/boot"
			if tc.mac != "" {
				url = fmt.Sprintf("/boot?mac=%s", tc.mac)
			}
			req := httptest.NewRequest("GET", url, nil)
			rr := httptest.NewRecorder()

			// Create a temporary server for each test run to isolate configs
			testServer := &httpServer{
				vmsDir:       server.vmsDir,
				templatePath: server.templatePath,
			}
			if tc.name == "Template Not Found" {
				testServer.templatePath = "/path/to/non/existent/template.tmpl"
			}
			handler := http.HandlerFunc(testServer.bootHandler)
			handler.ServeHTTP(rr, req)

			// 4. Assertion: Check the status code and body
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			body, err := io.ReadAll(rr.Body)
			if err != nil {
				t.Fatalf("could not read response body: %v", err)
			}

			if string(body) != tc.expectedBody {
				t.Errorf("handler returned unexpected body:\ngot:\n%v\nwant:\n%v",
					string(body), tc.expectedBody)
			}
		})
	}
}

func TestFindVMByMAC(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pvmlab-test-findvm")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// --- Setup ---
	// Valid VM
	vmMAC := "AA:BB:CC:DD:EE:FF"
	vmJSON := fmt.Sprintf(`{"name": "good-vm", "mac": "%s"}`, vmMAC)
	if err := os.WriteFile(filepath.Join(tmpDir, "good-vm.json"), []byte(vmJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Corrupted VM
	corruptedJSON := `{"name": "bad-vm", "mac": "11:22:33:44:55:66",` // Missing closing brace
	if err := os.WriteFile(filepath.Join(tmpDir, "bad-vm.json"), []byte(corruptedJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Non-JSON file
	if err := os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	server := &httpServer{vmsDir: tmpDir}

	// --- Test Cases ---
	t.Run("VM Found", func(t *testing.T) {
		vm, err := server.findVMByMAC(vmMAC)
		if err != nil {
			t.Fatalf("Expected to find VM, but got error: %v", err)
		}
		if vm.Name != "good-vm" {
			t.Errorf("Expected VM name 'good-vm', got '%s'", vm.Name)
		}
	})

	t.Run("VM Not Found", func(t *testing.T) {
		_, err := server.findVMByMAC("00:00:00:00:00:00")
		if err == nil {
			t.Fatal("Expected an error for a non-existent MAC, but got nil")
		}
	})

	t.Run("Case Insensitive MAC", func(t *testing.T) {
		vm, err := server.findVMByMAC(strings.ToLower(vmMAC))
		if err != nil {
			t.Fatalf("Expected to find VM with lowercase MAC, but got error: %v", err)
		}
		if vm.Name != "good-vm" {
			t.Errorf("Expected VM name 'good-vm', got '%s'", vm.Name)
		}
	})

	t.Run("Directory Not Found", func(t *testing.T) {
		badServer := &httpServer{vmsDir: "/path/to/non/existent/dir"}
		_, err := badServer.findVMByMAC(vmMAC)
		if err == nil {
			t.Fatal("Expected an error for a non-existent directory, but got nil")
		}
	})
}

func TestGetEnv(t *testing.T) {
	t.Run("Variable is set", func(t *testing.T) {
		key := fmt.Sprintf("PVMLAB_TEST_VAR_%d", time.Now().UnixNano())
		expectedValue := "my-test-value"

		t.Setenv(key, expectedValue)

		value := getEnv(key, "fallback")
		if value != expectedValue {
			t.Errorf("getEnv() = %q, want %q", value, expectedValue)
		}
	})

	t.Run("Variable is not set", func(t *testing.T) {
		key := fmt.Sprintf("PVMLAB_TEST_VAR_UNSET_%d", time.Now().UnixNano())
		fallbackValue := "my-fallback-value"

		value := getEnv(key, fallbackValue)
		if value != fallbackValue {
			t.Errorf("getEnv() = %q, want %q", value, fallbackValue)
		}
	})

	t.Run("Variable is set to empty string", func(t *testing.T) {
		key := fmt.Sprintf("PVMLAB_TEST_VAR_EMPTY_%d", time.Now().UnixNano())
		expectedValue := ""

		t.Setenv(key, expectedValue)

		value := getEnv(key, "fallback")
		if value != expectedValue {
			t.Errorf("getEnv() = %q, want %q", value, expectedValue)
		}
	})
}
