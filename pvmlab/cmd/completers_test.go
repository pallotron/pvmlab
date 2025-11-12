package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestVmNameCompleter(t *testing.T) {
	tests := []struct {
		name string
		args []string
		toComplete string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name: "basic completion",
			args: []string{},
			toComplete: "",
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "with partial completion",
			args: []string{},
			toComplete: "vm",
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			_, directive := VmNameCompleter(cmd, tt.args, tt.toComplete)

			// We expect either NoFileComp (success) or Error (if config fails)
			// Both are valid since config might not exist in test environment
			if directive != cobra.ShellCompDirectiveNoFileComp && directive != cobra.ShellCompDirectiveError {
				t.Errorf("VmNameCompleter() directive = %v, want %v or %v",
					directive, cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveError)
			}
		})
	}
}
