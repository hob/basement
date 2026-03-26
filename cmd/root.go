package cmd

import "spillane.farm/basement/cmd/commands/sync"

// Execute runs the CLI.
func Execute() error {
	return sync.Command().Execute()
}
