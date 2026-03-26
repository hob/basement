package sync

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GenerateLocalReport walks the given local directory and writes a CSV report of all .mkv files found.
//
// Output: all_mkv_files.csv (in the current working directory).
func GenerateLocalReport(localDir string) error {
	outputFileName := "all_mkv_files.csv"
	csvFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFileName, err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	header := []string{"Series", "Title", "Year", "Extra"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	count := 0
	err = filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: error accessing path %q: %v\n", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".mkv") {
			return nil
		}

		meta := parseMovieMetadataFromFilename(info.Name())
		record := []string{meta.Series, meta.Title, meta.Year, meta.Extra}
		if err := csvWriter.Write(record); err != nil {
			log.Printf("Warning: failed to write record for %s to CSV: %v", info.Name(), err)
			return nil
		}
		count++
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking directory %s: %w", localDir, err)
	}

	if count == 0 {
		fmt.Println("No .mkv files found in local directory.")
	} else {
		fmt.Printf("Report generated: %s (%d files)\n", outputFileName, count)
	}
	return nil
}
