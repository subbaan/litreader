package external

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/subbass/litreader/internal/cache"
)

// MaxSearchFiles is the maximum number of files to return from a search
// to prevent memory exhaustion on very broad searches
const MaxSearchFiles = 50000

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

	// Stream output instead of loading all into memory
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Parse output and count matches per file, with early termination
	matchCounts := make(map[string]int)
	scanner := bufio.NewScanner(stdout)

	// Increase scanner buffer for long lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Format: filename:line_number:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		// Convert the path from ripgrep to absolute
		filePath := parts[0]
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(searchDir, filePath)
		}

		// Clean the path to normalize it
		filePath = filepath.Clean(filePath)

		// Only count if file is in our list
		if fileSet[filePath] {
			// Check if this is a new file and we've hit the limit
			if matchCounts[filePath] == 0 && len(matchCounts) >= MaxSearchFiles {
				// We've hit the file limit, stop processing
				break
			}
			matchCounts[filePath]++
		}
	}

	// Kill ripgrep if still running (we may have stopped early)
	cmd.Process.Kill()
	cmd.Wait()

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
