package metadatascan

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"spillane.farm/basement/internal/metadatascan"
)

func Command() *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:   "metadata-scan <path>",
		Short: "List folders with metadata.json and show day-level date coverage (UTC), oldest first.",
		Long: `Scans for directories that contain a metadata.json file.

For each folder, date coverage is computed at day precision (UTC):

  • If metadata.json includes date.timestamp, that day is used.
  • Otherwise, every other *.json file in the folder is read and timestamps
    from photoTakenTime and creationTime are collected (Google Photos export style).

Covered days are merged into ranges (e.g. 2006-01-06..2006-01-07); each range
is printed on its own line in the second column. Folders are listed in the
first column, sorted by the earliest covered day (oldest first).

Without --recursive, only the given path and its immediate subfolders are
checked. With --recursive, all nested folders are scanned.`,
		Example: `  # Current directory and immediate children only
  basement metadata-scan .

  # All nested folders under a library path
  basement metadata-scan "D:\\Photos\\Takeout" -r`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hits, err := metadatascan.Scan(args[0], recursive)
			if err != nil {
				return err
			}
			if len(hits) == 0 {
				fmt.Println("No metadata.json files found.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "FOLDER\tDATES")
			for _, h := range hits {
				if len(h.DateLines) == 0 {
					continue
				}
				fmt.Fprintf(w, "%s\t%s\n", h.RelPath, h.DateLines[0])
				for i := 1; i < len(h.DateLines); i++ {
					fmt.Fprintf(w, "\t%s\n", h.DateLines[i])
				}
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Scan all nested subfolders (default: only <path> and its direct children).")
	return cmd
}
