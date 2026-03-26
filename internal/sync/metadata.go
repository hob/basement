package sync

import (
	"path/filepath"
	"regexp"
	"strings"
)

type MovieMetadata struct {
	Series string
	Title  string
	Year   string
	Extra  string
}

func parseMovieMetadataFromFilename(filename string) MovieMetadata {
	// Strip extension
	base := strings.TrimSpace(strings.TrimSuffix(filename, filepath.Ext(filename)))
	meta := MovieMetadata{}

	// Parse "Series - Title (Year) Extra" style using " - " separators.
	// Keeps hyphens inside titles, e.g. "Extra-Terrestrial".
	parts := strings.Split(base, " - ")
	primary := base
	if len(parts) >= 2 {
		meta.Series = strings.TrimSpace(parts[0])
		primary = strings.TrimSpace(strings.Join(parts[1:], " - "))
	}

	yearRE := regexp.MustCompile(`\((\d{4})\)`)
	if m := yearRE.FindStringSubmatchIndex(primary); len(m) > 0 {
		meta.Year = primary[m[2]:m[3]]
		before := strings.TrimSpace(primary[:m[0]])
		after := strings.TrimSpace(primary[m[1]:])
		meta.Title = before
		meta.Extra = strings.TrimSpace(strings.TrimPrefix(after, "-"))
	} else {
		meta.Title = strings.TrimSpace(primary)
	}

	// Final fallback
	if meta.Title == "" {
		meta.Title = base
	}

	return meta
}
