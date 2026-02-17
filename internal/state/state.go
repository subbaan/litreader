package state

import (
	"github.com/subbass/litreader/internal/cache"
	"github.com/subbass/litreader/internal/config"
	"github.com/subbass/litreader/internal/models"
)

// State holds the global application state shared across all views
type State struct {
	Config *config.Config
	Cache  *cache.Cache

	// Current file being viewed
	CurrentFile string

	// File list cache
	AllFiles     []string
	FileMetadata map[string]*cache.FileMetadata // Cached metadata (size, rating)
	FileInfo     map[string]*models.FileInfo

	// Direct file open mode (litreader somefile.txt)
	DirectFile string

	// Search state
	LastSearchQuery   string
	LastSearchResults []cache.SearchResult

	// Increment when favorites list changes (add/remove)
	FavoritesVersion int
}

// NewState creates a new application state
func NewState(cfg *config.Config) *State {
	return &State{
		Config:       cfg,
		FileInfo:     make(map[string]*models.FileInfo),
		FileMetadata: make(map[string]*cache.FileMetadata),
		AllFiles:     []string{},
	}
}
