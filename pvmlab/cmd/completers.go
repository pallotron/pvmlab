package cmd

import (
	"log"
	"provisioning-vm-lab/internal/metadata"
	"sort"

	"github.com/spf13/cobra"
)

func VmNameCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	allMeta, err := metadata.GetAll()
	if err != nil {
		// Log to stderr, which is appropriate for completion scripts
		log.Println("Error getting VM list for completion:", err)
		return nil, cobra.ShellCompDirectiveError
	}

	vmNames := make([]string, 0, len(allMeta))
	for name := range allMeta {
		vmNames = append(vmNames, name)
	}
	sort.Strings(vmNames)

	return vmNames, cobra.ShellCompDirectiveNoFileComp
}
