package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// VM represents the structure of the VM's JSON definition file.
type VM struct {
	Name    string `json:"name"`
	Arch    string `json:"arch"`
	Distro  string `json:"distro"`
	MAC     string `json:"mac"`
	ISOName string
}

type httpServer struct {
	vmsDir       string
	templatePath string
}

func (s *httpServer) bootHandler(w http.ResponseWriter, r *http.Request) {
	mac := r.URL.Query().Get("mac")
	if mac == "" {
		http.Error(w, "mac query parameter is required", http.StatusBadRequest)
		return
	}

	vm, err := s.findVMByMAC(mac)
	if err != nil {
		log.Printf("Error finding VM for MAC %s: %v", mac, err)
		http.Error(w, fmt.Sprintf("VM with MAC %s not found", mac), http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles(s.templatePath)
	if err != nil {
		log.Printf("Error parsing template file %s: %v", s.templatePath, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if err := tmpl.Execute(w, vm); err != nil {
		log.Printf("Error executing template for VM %s: %v", vm.Name, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *httpServer) findVMByMAC(mac string) (*VM, error) {
	files, err := os.ReadDir(s.vmsDir)
	if err != nil {
		return nil, fmt.Errorf("could not read vms directory %s: %w", s.vmsDir, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(s.vmsDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Warning: could not read file %s: %v", filePath, err)
			continue
		}

		var vm VM
		if err := json.Unmarshal(data, &vm); err != nil {
			log.Printf("Warning: could not unmarshal JSON from %s: %v", filePath, err)
			continue
		}

		if strings.EqualFold(vm.MAC, mac) {
			if vm.Distro == "ubuntu-24.04" {
				if vm.Arch == "aarch64" {
					vm.ISOName = "ubuntu-24.04.3-live-server-arm64.iso"
				} else if vm.Arch == "x86_64" {
					vm.ISOName = "ubuntu-24.04.3-live-server-amd64.iso"
				}
			}
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("no vm found with mac address %s", mac)
}

func main() {
	// Get default values from environment variables, with a fallback.
	defaultVmsDir := getEnv("PVMLAB_VMS_DIR", "/mnt/host/vms")
	defaultTemplatePath := getEnv("PVMLAB_TEMPLATE_PATH", "boot.ipxe.go.template")

	// Define command-line flags for configuration
	vmsDir := flag.String("vms-dir", defaultVmsDir, "Directory containing VM JSON definitions. Can also be set with PVMLAB_VMS_DIR.")
	templatePath := flag.String("template", defaultTemplatePath, "Path to the iPXE Go template file. Can also be set with PVMLAB_TEMPLATE_PATH.")
	flag.Parse()

	server := &httpServer{
		vmsDir:       *vmsDir,
		templatePath: *templatePath,
	}

	http.HandleFunc("/boot", server.bootHandler)
	log.Printf("Starting PXE boot server on :8080, watching VM definitions in %s", *vmsDir)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
