package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func marshal(in any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	err := enc.Encode(in)
	return buf.Bytes(), err
}

// MetaData structs
type MetaData struct {
	InstanceID    string   `yaml:"instance-id"`
	LocalHostname string   `yaml:"local-hostname"`
	PublicKeys    []string `yaml:"public-keys"`
}

// UserData structs
type SSHKeys string

func (s SSHKeys) MarshalYAML() (any, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: string(s),
		Style: yaml.LiteralStyle,
	}, nil
}

type User struct {
	Name              string  `yaml:"name"`
	Sudo              string  `yaml:"sudo"`
	Groups            string  `yaml:"groups"`
	Shell             string  `yaml:"shell"`
	SSHAuthorizedKeys SSHKeys `yaml:"ssh_authorized_keys"`
}

type ChPasswdUser struct {
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
	Type     string `yaml:"type"`
}

type ChPasswd struct {
	Users  []ChPasswdUser `yaml:"users"`
	Expire bool           `yaml:"expire"`
}

type WriteFile struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions"`
	Content     string `yaml:"content"`
}

type RunCmd []string

func (r RunCmd) MarshalYAML() (any, error) {
	node := yaml.Node{
		Kind: yaml.SequenceNode,
	}
	for _, s := range r {
		node.Content = append(node.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: s,
			Style: yaml.SingleQuotedStyle,
		})
	}
	return &node, nil
}

type UserData struct {
	SSHpwauth  bool        `yaml:"ssh_pwauth"`
	Users      []User      `yaml:"users"`
	ChPasswd   ChPasswd    `yaml:"chpasswd"`
	WriteFiles []WriteFile `yaml:"write_files"`
	RunCmd     RunCmd      `yaml:"runcmd"`
}

// NetworkConfig structs
type NetplanMatch struct {
	MacAddress string `yaml:"macaddress"`
}

type NetplanEthernet struct {
	Match *NetplanMatch `yaml:"match,omitempty"`
	DHCP4 bool          `yaml:"dhcp4"`
	DHCP6 bool          `yaml:"dhcp6,omitempty"`
}

type NetplanConfig struct {
	Version   int                        `yaml:"version"`
	Ethernets map[string]NetplanEthernet `yaml:"ethernets"`
}

func buildTargetMetaData(vmName, sshKey string) *MetaData {
	return &MetaData{
		InstanceID:    fmt.Sprintf("iid-cloudimg-%s", vmName),
		LocalHostname: vmName,
		PublicKeys:    []string{sshKey},
	}

}

func buildTargetUserData() *UserData {
	return &UserData{
		SSHpwauth: true,
		Users: []User{
			{
				Name:              "ubuntu",
				Sudo:              "ALL=(ALL) NOPASSWD:ALL",
				Groups:            "sudo",
				Shell:             "/bin/bash",
				SSHAuthorizedKeys: SSHKeys("{{ ds.meta_data.public_keys | join('\n') }}"),
			},
		},
		ChPasswd: ChPasswd{
			Users: []ChPasswdUser{
				{Name: "ubuntu", Password: "pass", Type: "text"},
				{Name: "root", Password: "pass", Type: "text"},
			},
			Expire: false,
		},
		RunCmd: RunCmd{
			"rm /etc/update-motd.d/50-landscape-sysinfo",
			"rm /etc/update-motd.d/10-help-text",
			"rm /etc/update-motd.d/50-motd-news",
			"rm /etc/update-motd.d/90-updates-available",
			"systemctl restart systemd-networkd",
		},
	}

}

func buildTargetNetworkConfig(mac string) *NetplanConfig {
	cfg := &NetplanConfig{}
	cfg.Version = 2
	cfg.Ethernets = map[string]NetplanEthernet{
		"static-interface": {
			Match: &NetplanMatch{MacAddress: mac},
			DHCP4: true,
			DHCP6: true,
		},
	}
	return cfg
}

// VM represents the structure of the VM's JSON definition file.
type VM struct {
	Name    string `json:"name"`
	Arch    string `json:"arch"`
	Distro  string `json:"distro"`
	MAC     string `json:"mac"`
	SSHKey  string `json:"ssh_key"`
	Kernel  string `json:"kernel,omitempty"`
	Initrd  string `json:"initrd,omitempty"`
	PxeBoot bool   `json:"pxeboot,omitempty"`
}

// InstallerConfig is the configuration provided to the installer running in the initrd.
type InstallerConfig struct {
	CloudInitURL    string `json:"cloud_init_url"`
	Distro          string `json:"distro"`
	Arch            string `json:"arch"`
	RootfsURL       string `json:"rootfs_url"`
	KmodsURL        string `json:"kmods_url"`
	KernelURL       string `json:"kernel_url"`
	RebootOnSuccess bool   `json:"reboot_on_success"`
}

type httpServer struct {
	vmsDir       string
	templatePath string
}

func (s *httpServer) configHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	mac := parts[2]

	vm, err := s.findVMByMAC(mac)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Determine the base URL from the request host
	// This makes the configuration independent of the provisioner's IP
	baseURL := fmt.Sprintf("http://%s", r.Host)

	// Check for a .noreboot file to disable reboot on success
	noRebootFilePath := filepath.Join(s.vmsDir, vm.Name+".noreboot")
	rebootOnSuccess := true
	if _, err := os.Stat(noRebootFilePath); err == nil {
		log.Printf("Found .noreboot file for %s, disabling reboot on success.", vm.Name)
		rebootOnSuccess = false
	}

	config := &InstallerConfig{
		CloudInitURL:    fmt.Sprintf("%s/cloud-init/%s", baseURL, vm.Name),
		Distro:          vm.Distro,
		Arch:            vm.Arch,
		RootfsURL:       fmt.Sprintf("%s/images/%s/%s/rootfs.tar.gz", baseURL, vm.Distro, vm.Arch),
		KmodsURL:        fmt.Sprintf("%s/images/%s/%s/modules.cpio.gz", baseURL, vm.Distro, vm.Arch),
		KernelURL:       fmt.Sprintf("%s/images/%s/%s/%s", baseURL, vm.Distro, vm.Arch, vm.Kernel),
		RebootOnSuccess: rebootOnSuccess,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (s *httpServer) cloudInitHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		errorMsg := fmt.Sprintf("Invalid cloud-init request path: %s. Path should be /cloud-init/<vm_name>/(meta-data|user-data|network-config)", r.URL.Path)
		log.Print(errorMsg)
		w.Write([]byte(errorMsg))
		http.NotFound(w, r)
		return
	}
	vmName := parts[2]
	fileType := parts[3]

	vm, err := s.findVMByName(vmName)
	if err != nil {
		errorMsg := fmt.Sprintf("Error finding VM %s: %v", vmName, err)
		log.Print(errorMsg)
		w.Write([]byte(errorMsg))
		http.NotFound(w, r)
		return
	}

	var data any
	switch fileType {
	case "meta-data":
		data = buildTargetMetaData(vm.Name, vm.SSHKey)
	case "user-data":
		data = buildTargetUserData()
	case "network-config":
		data = buildTargetNetworkConfig(vm.MAC)
	default:
		http.Error(w, "Invalid file type requested. Please use /cloud-init/<vm_name>/(meta-data|user-data|network-config)", http.StatusBadRequest)
		return
	}

	yamlData, err := marshal(data)
	if err != nil {
		log.Printf("Error marshalling %s for %s: %v", fileType, vmName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if fileType == "user-data" {
		userData := append([]byte("## template: jinja\n#cloud-config\n"), yamlData...)
		w.Header().Set("Content-Type", "text/yaml")
		w.Write(userData)
	} else {
		w.Header().Set("Content-Type", "text/yaml")
		w.Write(yamlData)
	}
}

func (s *httpServer) ipxeHandler(w http.ResponseWriter, r *http.Request) {
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
	return s.findVM(func(vm *VM) bool {
		return strings.EqualFold(vm.MAC, mac)
	})
}

func (s *httpServer) findVMByName(name string) (*VM, error) {
	return s.findVM(func(vm *VM) bool {
		return vm.Name == name
	})
}

func (s *httpServer) findVM(predicate func(*VM) bool) (*VM, error) {
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

		if predicate(&vm) {
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("no vm found")
}

func main() {
	// Get default values from environment variables, with a fallback.
	defaultVmsDir := getEnv("PVMLOAB_VMS_DIR", "/mnt/host/vms")
	defaultTemplatePath := getEnv("PVMLAB_TEMPLATE_PATH", "boot.ipxe.go.template")

	// Define command-line flags for configuration
	vmsDir := flag.String("vms-dir", defaultVmsDir, "Directory containing VM JSON definitions. Can also be set with PVMLAB_VMS_DIR.")
	templatePath := flag.String("template", defaultTemplatePath, "Path to the iPXE Go template file. Can also be set with PVMLAB_TEMPLATE_PATH.")
	flag.Parse()

	server := &httpServer{
		vmsDir:       *vmsDir,
		templatePath: *templatePath,
	}

	http.HandleFunc("/ipxe", server.ipxeHandler)
	http.HandleFunc("/cloud-init/", server.cloudInitHandler)
	http.HandleFunc("/config/", server.configHandler)
	log.Printf("Starting PXE boot server on :8080, watching VM definitions in %s", *vmsDir)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
