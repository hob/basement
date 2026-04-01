package cmd

import (
	"github.com/spf13/cobra"
	"spillane.farm/basement/cmd/commands/metadatascan"
	"spillane.farm/basement/cmd/commands/sync"
)

// Execute runs the CLI.
func Execute() error {
	root := &cobra.Command{
		Use:   "basement",
		Short: "Local media library tools.",
	}
	root.AddCommand(sync.Command())
	root.AddCommand(metadatascan.Command())
	return root.Execute()
}
