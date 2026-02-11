package config

import (
	"github.com/subbass/litreader/internal/models"
)

const (
	DefaultCacheExpiryDays = 7
)

// Config holds all configuration for litreader
type Config struct {
	SearchDir            string
	ExportDir            string
	UIColor              string
	SelectorColor        string
	SelectorReverse      bool
	SelectorBold         bool
	SelectorReverseColor string
	ContentColor         string
	ContentBold          bool
	ShowViewerHelpBar    bool
	ShowRatings          bool
	LastFile             string
	Position             int
	SearchText           string
	CacheExpiryDays      int

	Favorites       []models.Favorite
	Bookmarks       []models.Bookmark
	AuthorFavorites []string
}

// NewDefaultConfig returns a Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		SearchDir:            expandHome("~/Documents/"),
		ExportDir:            expandHome("~/Documents/litreader_faves"),
		UIColor:              "blue",
		SelectorColor:        "yellow",
		SelectorReverse:      true,
		SelectorBold:         true,
		SelectorReverseColor: "black",
		ContentColor:         "white",
		ContentBold:          false,
		ShowViewerHelpBar:    true,
		ShowRatings:          true,
		LastFile:             "",
		Position:             0,
		SearchText:           "",
		CacheExpiryDays:      DefaultCacheExpiryDays,
		Favorites:            []models.Favorite{},
		Bookmarks:            []models.Bookmark{},
		AuthorFavorites:      []string{},
	}
}
