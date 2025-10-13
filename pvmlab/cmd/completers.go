package cmd

import (
	"log"
	"pvmlab/internal/config"
	"pvmlab/internal/metadata"
	"sort"

	"github.com/spf13/cobra"
)

func VmNameCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.New()
	if err != nil {
		log.Println("Error creating config for completion:", err)
		return nil, cobra.ShellCompDirectiveError
	}
	allMeta, err := metadata.GetAll(cfg)
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
