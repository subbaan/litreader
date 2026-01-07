package external

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/subbass/litreader/internal/cache"
)

// SearchFiles searches for a pattern in files using ripgrep
func SearchFiles(pattern, searchDir string, fileList []string) ([]cache.SearchResult, error) {
	// Check if ripgrep is available
	if _, err := exec.LookPath("rg"); err != nil {
		return nil, err
	}

	// Create a set of valid files for filtering (use base directory matching)
	fileSet := make(map[string]bool)
	for _, f := range fileList {
		fileSet[f] = true
	}

	cmd := exec.Command("rg",
		"-i",              // Case insensitive
		"--no-heading",    // Don't show file headers
		"--line-number",   // Show line numbers
		"--with-filename", // Show filenames
		pattern,
		searchDir,
	)

	output, err := cmd.Output()
	if err != nil {
		// Exit code 1 means no matches found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []cache.SearchResult{}, nil
		}
		return nil, err
	}

	// Parse output and count matches per file
	matchCounts := make(map[string]int)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Format: filename:line_number:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		// Convert the path from ripgrep to absolute
		// If it's already absolute, use it; otherwise join with searchDir
		filePath := parts[0]
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(searchDir, filePath)
		}

		// Clean the path to normalize it
		filePath = filepath.Clean(filePath)

		// Only count if file is in our list
		if fileSet[filePath] {
			matchCounts[filePath]++
		}
	}

	// Convert to results
	results := make([]cache.SearchResult, 0, len(matchCounts))
	for path, count := range matchCounts {
		results = append(results, cache.SearchResult{
			FilePath:   path,
			MatchCount: count,
		})
	}

	return results, nil
}

// IsRipgrepAvailable checks if ripgrep is installed
func IsRipgrepAvailable() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}
