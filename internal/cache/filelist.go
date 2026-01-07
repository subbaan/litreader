package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/subbass/litreader/internal/config"
)

// FileMetadata holds cached metadata for a single file
type FileMetadata struct {
	Path   string  `json:"path"`
	Size   int64   `json:"size"`
	Rating float64 `json:"rating"`
}

// FileListCache holds the library file list with timestamp and metadata
type FileListCache struct {
	Timestamp time.Time      `json:"timestamp"`
	SearchDir string         `json:"search_dir"`
	Files     []FileMetadata `json:"files"`
}

// LoadFileListCache loads cached file list with metadata
func LoadFileListCache(searchDir string, expiryDays int) ([]FileMetadata, error) {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(cacheDir, "filelist.json")

	// If cache doesn't exist, return nil
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache FileListCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is for the same directory
	if cache.SearchDir != searchDir {
		return nil, nil
	}

	// Check if cache is expired
	cacheTTL := time.Duration(expiryDays) * 24 * time.Hour
	if time.Since(cache.Timestamp) > cacheTTL {
		return nil, nil
	}

	return cache.Files, nil
}

// SaveFileListCache saves file list with metadata to cache
func SaveFileListCache(searchDir string, files []FileMetadata) error {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return err
	}

	cache := &FileListCache{
		Timestamp: time.Now(),
		SearchDir: searchDir,
		Files:     files,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, "filelist.json")
	return os.WriteFile(cachePath, data, 0644)
}

// ClearFileListCache removes the cached file list
func ClearFileListCache() error {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, "filelist.json")

	// If file doesn't exist, that's fine
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(cachePath)
}
