package metadatascan

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Hit is a directory that directly contains metadata.json.
type Hit struct {
	RelPath string
	// StartDay is the earliest day covered (UTC, truncated to day).
	StartDay time.Time
	// EndDay is the latest day covered (UTC, truncated to day).
	EndDay time.Time
	// DateLines are day-coverage strings, one per output line (e.g. "2006-01-02" or "2006-01-02..2006-01-07").
	DateLines []string
}

type metadataFile struct {
	Date struct {
		Timestamp string `json:"timestamp"`
		Formatted string `json:"formatted"`
	} `json:"date"`
}

type imageMetadataFile struct {
	CreationTime struct {
		Timestamp string `json:"timestamp"`
		Formatted string `json:"formatted"`
	} `json:"creationTime"`
	PhotoTakenTime struct {
		Timestamp string `json:"timestamp"`
		Formatted string `json:"formatted"`
	} `json:"photoTakenTime"`
}

func parseUnixSeconds(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	// Timestamps in the export are UTC seconds.
	return time.Unix(sec, 0).UTC(), true
}

func readAlbumDateTimestamp(path string) (time.Time, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, false
	}
	var m metadataFile
	if err := json.Unmarshal(b, &m); err != nil {
		return time.Time{}, false
	}
	return parseUnixSeconds(m.Date.Timestamp)
}

func readImageLikeTimestamps(path string) []time.Time {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m imageMetadataFile
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	var out []time.Time
	if t, ok := parseUnixSeconds(m.PhotoTakenTime.Timestamp); ok {
		out = append(out, t)
	}
	if t, ok := parseUnixSeconds(m.CreationTime.Timestamp); ok {
		out = append(out, t)
	}
	return out
}

func toDayUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func formatDay(t time.Time) string {
	return t.Format("2006-01-02")
}

func dayRanges(days []time.Time) (start time.Time, end time.Time, lines []string, ok bool) {
	if len(days) == 0 {
		return time.Time{}, time.Time{}, nil, false
	}
	uniq := make(map[time.Time]struct{}, len(days))
	var d []time.Time
	for _, t := range days {
		day := toDayUTC(t)
		if _, exists := uniq[day]; exists {
			continue
		}
		uniq[day] = struct{}{}
		d = append(d, day)
	}
	sort.Slice(d, func(i, j int) bool { return d[i].Before(d[j]) })
	start, end = d[0], d[len(d)-1]

	var parts []string
	runStart := d[0]
	runPrev := d[0]
	oneDay := 24 * time.Hour
	flush := func() {
		if runStart.Equal(runPrev) {
			parts = append(parts, formatDay(runStart))
		} else {
			parts = append(parts, fmt.Sprintf("%s..%s", formatDay(runStart), formatDay(runPrev)))
		}
	}
	for i := 1; i < len(d); i++ {
		if d[i].Sub(runPrev) == oneDay {
			runPrev = d[i]
			continue
		}
		flush()
		runStart = d[i]
		runPrev = d[i]
	}
	flush()

	return start, end, parts, true
}

func scanFolderCoverage(dirAbs string) (start time.Time, end time.Time, lines []string, ok bool) {
	metaPath := filepath.Join(dirAbs, "metadata.json")
	if t, ok := readAlbumDateTimestamp(metaPath); ok {
		day := toDayUTC(t)
		return day, day, []string{formatDay(day)}, true
	}

	entries, err := os.ReadDir(dirAbs)
	if err != nil {
		return time.Time{}, time.Time{}, nil, false
	}

	var ts []time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		// Skip the folder-level metadata.json (already tried).
		if strings.EqualFold(e.Name(), "metadata.json") {
			continue
		}
		ts = append(ts, readImageLikeTimestamps(filepath.Join(dirAbs, e.Name()))...)
	}
	return dayRanges(ts)
}

// Scan finds folders containing metadata.json and returns them with paths
// relative to cwd and the date from that metadata.json. When recursive is true,
// subfolders are scanned; otherwise only root and its direct children are checked.
func Scan(root string, recursive bool) ([]Hit, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve scan root: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}

	var hits []Hit

	addHit := func(dirAbs string) {
		rel, err := filepath.Rel(cwdAbs, dirAbs)
		if err != nil {
			rel = dirAbs
		}
		start, end, lines, ok := scanFolderCoverage(dirAbs)
		if !ok {
			return
		}
		hits = append(hits, Hit{RelPath: rel, StartDay: start, EndDay: end, DateLines: lines})
	}

	if recursive {
		err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(d.Name(), "metadata.json") {
				return nil
			}
			dirAbs := filepath.Dir(path)
			dirAbs, err = filepath.Abs(dirAbs)
			if err != nil {
				return nil
			}
			addHit(dirAbs)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", rootAbs, err)
		}
	} else {
		addHit(rootAbs)
		entries, err := os.ReadDir(rootAbs)
		if err != nil {
			return nil, fmt.Errorf("read dir %s: %w", rootAbs, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			addHit(filepath.Join(rootAbs, e.Name()))
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		return hits[i].StartDay.Before(hits[j].StartDay)
	})
	return hits, nil
}
