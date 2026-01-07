package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/subbass/litreader/internal/config"
	"github.com/subbass/litreader/internal/models"
)

// Cache holds file metadata with timestamps
type Cache struct {
	Timestamp time.Time                  `json:"timestamp"`
	Files     map[string]*models.FileInfo `json:"files"`
}

// getMainCacheFileName returns the main cache file name based on the app name
func getMainCacheFileName() string {
	if len(os.Args) > 0 {
		appName := filepath.Base(os.Args[0])
		return appName + "_cache.json"
	}
	return "txtreader_cache.json" // Fallback
}

// Load reads the cache from disk
func Load() (*Cache, error) {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(cacheDir, getMainCacheFileName())

	// If cache doesn't exist, return empty cache
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return NewCache(), nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		// If unmarshal fails, return empty cache rather than error
		return NewCache(), nil
	}

	return &cache, nil
}

// Save writes the cache to disk
func (c *Cache) Save() error {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, getMainCacheFileName())

	c.Timestamp = time.Now()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// IsExpired checks if the cache is older than the expiry duration
func (c *Cache) IsExpired(expiryDays int) bool {
	if c.Timestamp.IsZero() {
		return true
	}

	expiryDuration := time.Duration(expiryDays) * 24 * time.Hour
	return time.Since(c.Timestamp) > expiryDuration
}

// Get retrieves file info from cache
func (c *Cache) Get(path string) (*models.FileInfo, bool) {
	info, exists := c.Files[path]
	return info, exists
}

// Set adds or updates file info in cache
func (c *Cache) Set(path string, info *models.FileInfo) {
	c.Files[path] = info
}

// NewCache creates a new empty cache
func NewCache() *Cache {
	return &Cache{
		Timestamp: time.Now(),
		Files:     make(map[string]*models.FileInfo),
	}
}
