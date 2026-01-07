package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/subbass/litreader/internal/config"
)

const searchCacheTTL = 24 * time.Hour

// SearchResult represents a single search result
type SearchResult struct {
	FilePath   string `json:"file_path"`
	MatchCount int    `json:"match_count"`
}

// SearchCache holds search results with timestamp
type SearchCache struct {
	Timestamp time.Time      `json:"timestamp"`
	Query     string         `json:"query"`
	Results   []SearchResult `json:"results"`
}

// LoadSearchCache loads cached search results for a query
func LoadSearchCache(query string) (*SearchCache, error) {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return nil, err
	}

	searchDir := filepath.Join(cacheDir, "searches")
	cachePath := filepath.Join(searchDir, getCacheFileName(query))

	// If cache doesn't exist, return nil
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache SearchCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is expired
	if time.Since(cache.Timestamp) > searchCacheTTL {
		return nil, nil
	}

	return &cache, nil
}

// SaveSearchCache saves search results to cache
func SaveSearchCache(query string, results []SearchResult) error {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return err
	}

	searchDir := filepath.Join(cacheDir, "searches")
	if err := os.MkdirAll(searchDir, 0755); err != nil {
		return err
	}

	cache := &SearchCache{
		Timestamp: time.Now(),
		Query:     query,
		Results:   results,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(searchDir, getCacheFileName(query))
	return os.WriteFile(cachePath, data, 0644)
}

// getCacheFileName generates a safe filename from a query
func getCacheFileName(query string) string {
	// Use SHA256 hash to create a safe filename
	hash := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x.json", hash[:16])
}

// ClearAllSearchCaches removes all cached search results
func ClearAllSearchCaches() error {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return err
	}

	searchDir := filepath.Join(cacheDir, "searches")

	// Remove the entire searches directory
	if err := os.RemoveAll(searchDir); err != nil {
		// If directory doesn't exist, that's fine
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return nil
}
