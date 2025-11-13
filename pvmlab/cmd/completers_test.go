package cmd

import (
	"errors"
	"testing"

	"pvmlab/internal/config"
	"pvmlab/internal/metadata"

	"github.com/spf13/cobra"
)

func TestVmNameCompleter(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		toComplete    string
		setupMock     func()
		wantDirective cobra.ShellCompDirective
		checkNames    bool
		wantNames     []string
	}{
		{
			name:       "basic completion with VMs",
			args:       []string{},
			toComplete: "",
			setupMock: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"vm1": {Name: "vm1"},
						"vm2": {Name: "vm2"},
						"vm3": {Name: "vm3"},
					}, nil
				}
			},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
			checkNames:    true,
			wantNames:     []string{"vm1", "vm2", "vm3"},
		},
		{
			name:       "with partial completion",
			args:       []string{},
			toComplete: "vm",
			setupMock: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{
						"vm-test":  {Name: "vm-test"},
						"vm-prod":  {Name: "vm-prod"},
						"provisioner": {Name: "provisioner"},
					}, nil
				}
			},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
			checkNames:    true,
			wantNames:     []string{"provisioner", "vm-prod", "vm-test"}, // Sorted
		},
		{
			name:       "no VMs available",
			args:       []string{},
			toComplete: "",
			setupMock: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return map[string]*metadata.Metadata{}, nil
				}
			},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
			checkNames:    true,
			wantNames:     []string{},
		},
		{
			name:       "error getting VMs",
			args:       []string{},
			toComplete: "",
			setupMock: func() {
				metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
					return nil, errors.New("database error")
				}
			},
			wantDirective: cobra.ShellCompDirectiveError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			if tt.setupMock != nil {
				tt.setupMock()
			}

			cmd := &cobra.Command{}
			names, directive := VmNameCompleter(cmd, tt.args, tt.toComplete)

			if directive != tt.wantDirective {
				t.Errorf("VmNameCompleter() directive = %v, want %v", directive, tt.wantDirective)
			}

			if tt.checkNames {
				if len(names) != len(tt.wantNames) {
					t.Errorf("VmNameCompleter() returned %d names, want %d", len(names), len(tt.wantNames))
				}

				for i, name := range names {
					if i < len(tt.wantNames) && name != tt.wantNames[i] {
						t.Errorf("VmNameCompleter() names[%d] = %q, want %q", i, name, tt.wantNames[i])
					}
				}
			}
		})
	}
}

func TestVmNameCompleter_Sorting(t *testing.T) {
	// Test that VM names are returned in sorted order
	metadata.GetAll = func(c *config.Config) (map[string]*metadata.Metadata, error) {
		return map[string]*metadata.Metadata{
			"zulu":  {Name: "zulu"},
			"alpha": {Name: "alpha"},
			"mike":  {Name: "mike"},
			"bravo": {Name: "bravo"},
		}, nil
	}

	cmd := &cobra.Command{}
	names, directive := VmNameCompleter(cmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("VmNameCompleter() directive = %v, want NoFileComp", directive)
	}

	expected := []string{"alpha", "bravo", "mike", "zulu"}
	if len(names) != len(expected) {
		t.Fatalf("VmNameCompleter() returned %d names, want %d", len(names), len(expected))
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("VmNameCompleter() names[%d] = %q, want %q", i, name, expected[i])
		}
	}
}
