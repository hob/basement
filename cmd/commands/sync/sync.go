package sync

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"spillane.farm/basement/internal/sync"
)

func Command() *cobra.Command {
	var fileType string
	var localDir string
	var networkDir string
	var targetDir string
	var report bool
	var preserveStructure bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize files between a remote network drive and a local library.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if report {
				if strings.TrimSpace(localDir) == "" {
					return fmt.Errorf("when using --report, --local is required")
				}
				return sync.GenerateLocalReport(localDir)
			}

			if strings.TrimSpace(localDir) == "" || strings.TrimSpace(networkDir) == "" {
				return fmt.Errorf("both --local and --remote are required (unless --report is used)")
			}

			effectiveTargetDir := targetDir
			if strings.TrimSpace(effectiveTargetDir) == "" {
				effectiveTargetDir = localDir
			}

			return sync.SyncMissing(localDir, networkDir, effectiveTargetDir, preserveStructure)
		},
	}
	cmd.Flags().StringVarP(&fileType, "file-type", "f", "mkv", "Type of file to sync (e.g. mkv, mp4, etc.).")
	cmd.Flags().StringVarP(&localDir, "local", "l", "", "Path to the local directory to store files.")
	cmd.Flags().StringVarP(&networkDir, "remote", "r", "", "Path to the network drive to scan for files.")
	cmd.Flags().StringVarP(&targetDir, "target", "t", "", "Optional path to store copied files (defaults to --local).")
	cmd.Flags().BoolVarP(&report, "report", "", false, "Generate CSV report of all local files (requires --local, ignores --remote).")
	cmd.Flags().BoolVarP(&preserveStructure, "preserve-structure", "p", false, "Preserve subfolder structure under --target (mirrors paths found under --remote).")

	return cmd
}
