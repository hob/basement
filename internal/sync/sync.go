package sync

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/huh/v2"
)

type missingFile struct {
	Path string
	Info os.FileInfo
}

// SyncMissing indexes the local directory and scans the remote directory for missing .mkv files.
// It then prompts the user to select which missing files to copy into targetDir.
func SyncMissing(localDir, networkDir, targetDir string, preserveStructure bool) error {
	fmt.Println("Step 1: Indexing local files...")
	localFiles, err := indexMkvFiles(localDir)
	if err != nil {
		return fmt.Errorf("error indexing local directory '%s': %w", localDir, err)
	}
	fmt.Printf("-> Found %d local .mkv files.\n\n", len(localFiles))

	fmt.Println("Step 2: Scanning network drive for missing files...")
	missingFiles, err := scanAndReportMissingFiles(networkDir, &localFiles)
	if err != nil {
		return fmt.Errorf("error scanning network drive '%s': %w", networkDir, err)
	}

	if len(missingFiles) > 0 {
		selected := promptForSelection(missingFiles)
		if len(selected) > 0 {
			copySelectedToLocal(selected, targetDir, networkDir, preserveStructure)
		}
	}

	fmt.Println("\nScan complete.")
	return nil
}

// indexMkvFiles walks the given directory and creates a set of all .mkv filenames.
func indexMkvFiles(dir string) (map[string]bool, error) {
	return _indexMkvFiles(dir, make(map[string]bool))
}

func _indexMkvFiles(dir string, files map[string]bool) (map[string]bool, error) {
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
// in the localFiles map, it writes its details to missing_files.csv and returns the list.
func scanAndReportMissingFiles(networkDir string, localFiles *map[string]bool) ([]missingFile, error) {
	outputFileName := "missing_files.csv"
	csvFile, err := os.Create(outputFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", outputFileName, err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	// Write CSV Header
	header := []string{"Filename", "SourcePath", "Size(Bytes)", "ModifiedTime"}
	if err := csvWriter.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	missing := make([]missingFile, 0)
	err = scanForMissingFiles(networkDir, localFiles, &missing)
	if err != nil {
		return nil, fmt.Errorf("error scanning for missing files: %w", err)
	}
	if len(missing) == 0 {
		fmt.Println("-> No new files found.")
	} else {
		for _, m := range missing {
			fileName := m.Info.Name()
			record := []string{
				fileName,
				m.Path,
				strconv.FormatInt(m.Info.Size(), 10),
				m.Info.ModTime().UTC().Format(time.RFC3339),
			}

			if err := csvWriter.Write(record); err != nil {
				log.Printf("   Warning: Failed to write record for %s to CSV: %v", fileName, err)
			}

			// Add the file to our map so we don't write duplicate rows if it's found again.
			(*localFiles)[fileName] = true
		}
		fmt.Printf("\nReport generated: %s\n", outputFileName)
	}

	return missing, nil
}

func scanForMissingFiles(networkDir string, localFiles *map[string]bool, missing *[]missingFile) error {
	err := filepath.Walk(networkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Warning: error accessing path %q: %v\n", path, err)
			return nil
		}

		if info.IsDir() {
			if path != networkDir {
				// Recursively scan subdirectories
				return scanForMissingFiles(path, localFiles, missing)
			}
		} else if strings.HasSuffix(strings.ToLower(info.Name()), ".mkv") {
			fileName := info.Name()
			if !(*localFiles)[fileName] {
				fmt.Printf("-> Found missing file: %s\n", fileName)
				*missing = append(*missing, missingFile{Path: path, Info: info})
				// Add the file to our map to prevent duplicates
				(*localFiles)[fileName] = true
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory %s: %w", networkDir, err)
	}
	return nil
}

// promptForSelection shows a checkbox list (whiptail-style) of missing files and returns the selected subset.
// Uses Charm huh for an interactive TUI; falls back to text prompt when not a terminal.
func promptForSelection(missing []missingFile) []missingFile {
	if len(missing) == 0 {
		return nil
	}
	// Prefer checkbox TUI when stdin is a terminal
	if isTerminal(os.Stdin) {
		return promptForSelectionTUI(missing)
	}
	return promptForSelectionText(missing)
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func promptForSelectionTUI(missing []missingFile) []missingFile {
	opts := make([]huh.Option[int], 0, len(missing))
	for i, m := range missing {
		label := fmt.Sprintf("%s  (%s)", m.Info.Name(), formatSize(m.Info.Size()))
		opts = append(opts, huh.NewOption(label, i))
	}
	var selectedIndices []int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select files to copy to local (space to toggle, enter to confirm)").
				Options(opts...).
				Height(min(12, len(missing)+2)).
				Value(&selectedIndices),
		),
	)
	if err := form.Run(); err != nil {
		log.Printf("TUI prompt failed (use a terminal for checkboxes): %v", err)
		return promptForSelectionText(missing)
	}
	result := make([]missingFile, 0, len(selectedIndices))
	for _, i := range selectedIndices {
		result = append(result, missing[i])
	}
	return result
}

func promptForSelectionText(missing []missingFile) []missingFile {
	fmt.Println("\nMissing files - which do you want to copy to local?")
	for i, m := range missing {
		fmt.Printf("  %d) %s (%s)\n", i+1, m.Info.Name(), formatSize(m.Info.Size()))
	}
	fmt.Print("Enter numbers (e.g. 1,3,5), 'all', or 'none': ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read input: %v", err)
		return nil
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" || line == "none" {
		return nil
	}
	if line == "all" {
		return missing
	}
	var selected []missingFile
	for _, s := range strings.Split(line, ",") {
		s = strings.TrimSpace(s)
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > len(missing) {
			log.Printf("Skipping invalid selection: %q", s)
			continue
		}
		selected = append(selected, missing[n-1])
	}
	return selected
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// copySelectedToLocal copies the selected missing files to targetDir.
// If preserveStructure is enabled, it recreates the folder structure relative to sourceDir (--remote).
func copySelectedToLocal(selected []missingFile, targetDir string, sourceDir string, preserveStructure bool) {
	for _, m := range selected {
		dest := filepath.Join(targetDir, m.Info.Name())
		if preserveStructure && sourceDir != "" && m.Path != "" {
			if rel, err := filepath.Rel(sourceDir, m.Path); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				dest = filepath.Join(targetDir, rel)
			}
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			log.Printf("failed to create destination directory for %s: %v", dest, err)
			continue
		}
		fmt.Printf("Copying %s -> %s ... ", m.Info.Name(), dest)
		srcFile, err := os.Open(m.Path)
		if err != nil {
			log.Printf("failed to open source: %v", err)
			continue
		}
		destFile, err := os.Create(dest)
		if err != nil {
			srcFile.Close()
			log.Printf("failed to create destination: %v", err)
			continue
		}
		_, err = io.Copy(destFile, srcFile)
		srcFile.Close()
		destFile.Close()
		if err != nil {
			log.Printf("copy failed: %v", err)
			os.Remove(dest)
			continue
		}
		fmt.Println("done.")
	}
}
