package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	localDir := flag.String("local", "", "Path to the local directory to store mkv files.")
	networkDir := flag.String("remote", "", "Path to the network drive to scan for mkv files.")
	flag.Parse()

	if *localDir == "" || *networkDir == "" {
		fmt.Println("Usage: mkv-sync --local <your-local-mkv-directory> --remote <network-drive-path>")
		log.Fatal("Both --local and --remote directory paths are required.")
	}

	fmt.Println("Step 1: Indexing local files...")
	localFiles, err := indexMkvFiles(*localDir)
	if err != nil {
		log.Fatalf("Error indexing local directory '%s': %v", *localDir, err)
	}
	fmt.Printf("-> Found %d local .mkv files.\n\n", len(localFiles))

	fmt.Println("Step 2: Scanning network drive for missing files...")
	err = scanAndReportMissingFiles(*networkDir, localFiles)
	if err != nil {
		log.Fatalf("Error scanning network drive '%s': %v", *networkDir, err)
	}

	fmt.Println("\nScan complete.")
}

// indexMkvFiles walks the given directory and creates a set of all .mkv filenames.
func indexMkvFiles(dir string) (map[string]bool, error) {
	return _indexMkvFiles(dir, make(map[string]bool))
}

func _indexMkvFiles(dir string, files map[string]bool) (map[string]bool, error) {
	log.Printf("Walking directory: %s", dir)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Don't stop for single file errors, just log them.
			log.Printf("Warning: error accessing path %q: %v\n", path, err)
			return nil
		}
		if info.IsDir() {
			if path != dir {
				_indexMkvFiles(path, files)
			}
		} else if strings.HasSuffix(strings.ToLower(info.Name()), ".mkv") {
			files[info.Name()] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", dir, err)
	}
	return files, nil
}

// scanAndReportMissingFiles walks the network directory, and if a .mkv file is not
// in the localFiles map, it writes its details to missing_files.csv.
func scanAndReportMissingFiles(networkDir string, localFiles map[string]bool) error {
	outputFileName := "missing_files.csv"
	csvFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFileName, err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	// Write CSV Header
	header := []string{"Filename", "SourcePath", "Size(Bytes)", "ModifiedTime"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	var filesFound int
	err = filepath.Walk(networkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: error accessing path %q: %v\n", path, err)
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mkv") {
			fileName := info.Name()
			if !localFiles[fileName] {
				filesFound++
				fmt.Printf("-> Found missing file: %s\n", fileName)

				record := []string{
					fileName,
					path,
					strconv.FormatInt(info.Size(), 10),
					info.ModTime().UTC().Format(time.RFC3339),
				}

				if err := csvWriter.Write(record); err != nil {
					log.Printf("   Warning: Failed to write record for %s to CSV: %v", fileName, err)
				}

				// Add the file to our map so we don't write duplicate rows if it's found again.
				localFiles[fileName] = true
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory %s: %w", networkDir, err)
	}

	if filesFound == 0 {
		fmt.Println("-> No new files found.")
	} else {
		fmt.Printf("\nReport generated: %s\n", outputFileName)
	}

	return nil
}
