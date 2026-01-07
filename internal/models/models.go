package models

import "time"

// Favorite represents a favorited file with reading progress
type Favorite struct {
	Filename   string     `json:"filename"`
	Position   int        `json:"position"`
	SearchText string     `json:"search_text"`
	DateAdded  time.Time  `json:"date_added"`
	Rating     *float64   `json:"rating,omitempty"` // Nullable for compatibility
}

// Bookmark represents a bookmarked position in a file with a note
type Bookmark struct {
	Filename  string     `json:"filename"`
	Position  int        `json:"position"`
	Note      string     `json:"note"`
	DateAdded time.Time  `json:"date_added"`
	Rating    *float64   `json:"rating,omitempty"` // Nullable for compatibility
}

// FileInfo holds metadata about a file
type FileInfo struct {
	Size   int64    `json:"size"`
	Rating float64  `json:"rating"`
}
