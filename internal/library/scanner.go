package library

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/subbass/litreader/internal/cache"
)

// ScanFiles recursively scans a directory for .txt and .md files
func ScanFiles(searchDir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't read
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".txt" || ext == ".md" {
			absPath, err := filepath.Abs(path)
			if err == nil {
				files = append(files, absPath)
			}
		}

		return nil
	})

	return files, err
}

// ScanFilesWithMetadata scans directory and collects file metadata (size, rating)
func ScanFilesWithMetadata(searchDir string) ([]cache.FileMetadata, error) {
	var files []cache.FileMetadata

	err := filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't read
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".txt" || ext == ".md" {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil
			}

			// Get file size
			var size int64
			if info, err := os.Stat(absPath); err == nil {
				size = info.Size()
			}

			// Get rating
			var rating float64
			if r, err := ExtractRating(absPath); err == nil {
				rating = r
			}

			files = append(files, cache.FileMetadata{
				Path:   absPath,
				Size:   size,
				Rating: rating,
			})
		}

		return nil
	})

	return files, err
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
