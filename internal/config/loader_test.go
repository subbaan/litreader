package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/subbass/litreader/internal/models"
)

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"False", false},
		{"0", false},
		{"no", false},
		{"", false},
	}

	for _, tt := range tests {
		result := parseBool(tt.input)
		if result != tt.expected {
			t.Errorf("parseBool(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/Documents", filepath.Join(home, "Documents")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := expandHome(tt.input)
		if result != tt.expected {
			t.Errorf("expandHome(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestLoadConfigWithFavorites(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "litreader.conf")

	configContent := `SEARCH_DIR=/home/user/docs
EXPORT_DIR=/home/user/exports
editor=nano
ui_color=blue
selector_color=yellow
selector_reverse=true
selector_bold=true
selector_reverse_color=black
content_color=white
content_bold=false
last_file=/home/user/test.txt
position=100
search_text=hello
CACHE_EXPIRY_DAYS=7
favorite_1_filename=/path/to/file1.txt
favorite_1_position=50
favorite_1_search_text=test
favorite_1_date_added=2024-01-15T10:30:00Z
favorite_1_rating=4.5
favorite_2_filename=/path/to/file2.txt
favorite_2_position=0
favorite_2_search_text=
favorite_2_date_added=
favorite_2_rating=
bookmark_1_filename=/path/to/bookmark.txt
bookmark_1_position=200
bookmark_1_note=Important passage
bookmark_1_date_added=2024-01-20T15:45:00Z
bookmark_1_rating=4.8
author_favorite_1=John Doe
author_favorite_2=Jane Smith
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return configPath, nil
	}
	defer func() { GetConfigPath = originalGetConfigPath }()

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify basic fields
	if cfg.SearchDir != "/home/user/docs" {
		t.Errorf("SearchDir = %q, expected /home/user/docs", cfg.SearchDir)
	}
	if cfg.Position != 100 {
		t.Errorf("Position = %d, expected 100", cfg.Position)
	}
	if cfg.SearchText != "hello" {
		t.Errorf("SearchText = %q, expected hello", cfg.SearchText)
	}
	if !cfg.SelectorReverse {
		t.Error("SelectorReverse should be true")
	}

	// Verify favorites
	if len(cfg.Favorites) != 2 {
		t.Fatalf("len(Favorites) = %d, expected 2", len(cfg.Favorites))
	}

	fav1 := cfg.Favorites[0]
	if fav1.Filename != "/path/to/file1.txt" {
		t.Errorf("Favorite[0].Filename = %q, expected /path/to/file1.txt", fav1.Filename)
	}
	if fav1.Position != 50 {
		t.Errorf("Favorite[0].Position = %d, expected 50", fav1.Position)
	}
	if fav1.Rating == nil || *fav1.Rating != 4.5 {
		t.Errorf("Favorite[0].Rating = %v, expected 4.5", fav1.Rating)
	}

	fav2 := cfg.Favorites[1]
	if fav2.Filename != "/path/to/file2.txt" {
		t.Errorf("Favorite[1].Filename = %q, expected /path/to/file2.txt", fav2.Filename)
	}
	if fav2.Rating != nil {
		t.Errorf("Favorite[1].Rating = %v, expected nil", fav2.Rating)
	}

	// Verify bookmarks
	if len(cfg.Bookmarks) != 1 {
		t.Fatalf("len(Bookmarks) = %d, expected 1", len(cfg.Bookmarks))
	}

	bm := cfg.Bookmarks[0]
	if bm.Filename != "/path/to/bookmark.txt" {
		t.Errorf("Bookmark[0].Filename = %q, expected /path/to/bookmark.txt", bm.Filename)
	}
	if bm.Note != "Important passage" {
		t.Errorf("Bookmark[0].Note = %q, expected 'Important passage'", bm.Note)
	}

	// Verify author favorites
	if len(cfg.AuthorFavorites) != 2 {
		t.Fatalf("len(AuthorFavorites) = %d, expected 2", len(cfg.AuthorFavorites))
	}
	if cfg.AuthorFavorites[0] != "John Doe" {
		t.Errorf("AuthorFavorites[0] = %q, expected 'John Doe'", cfg.AuthorFavorites[0])
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "litreader.conf")

	// Override GetConfigPath for testing
	originalGetConfigPath := GetConfigPath
	GetConfigPath = func() (string, error) {
		return configPath, nil
	}
	defer func() { GetConfigPath = originalGetConfigPath }()

	// Create a config with data
	rating1 := 4.5
	rating2 := 3.8
	cfg := &Config{
		SearchDir:       "/test/search",
		ExportDir:       "/test/export",
		Editor:          "vim",
		UIColor:         "green",
		SelectorColor:   "red",
		SelectorReverse: true,
		Position:        42,
		Favorites: []models.Favorite{
			{
				Filename:   "/test/fav1.txt",
				Position:   10,
				SearchText: "search1",
				DateAdded:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Rating:     &rating1,
			},
		},
		Bookmarks: []models.Bookmark{
			{
				Filename:  "/test/bm1.txt",
				Position:  20,
				Note:      "Test note",
				DateAdded: time.Date(2024, 1, 20, 15, 45, 0, 0, time.UTC),
				Rating:    &rating2,
			},
		},
		AuthorFavorites: []string{"Author One", "Author Two"},
	}

	// Save
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify key fields
	if loaded.SearchDir != cfg.SearchDir {
		t.Errorf("SearchDir mismatch: got %q, want %q", loaded.SearchDir, cfg.SearchDir)
	}
	if loaded.Position != cfg.Position {
		t.Errorf("Position mismatch: got %d, want %d", loaded.Position, cfg.Position)
	}
	if len(loaded.Favorites) != len(cfg.Favorites) {
		t.Errorf("Favorites count mismatch: got %d, want %d", len(loaded.Favorites), len(cfg.Favorites))
	}
	if len(loaded.Bookmarks) != len(cfg.Bookmarks) {
		t.Errorf("Bookmarks count mismatch: got %d, want %d", len(loaded.Bookmarks), len(cfg.Bookmarks))
	}
	if len(loaded.AuthorFavorites) != len(cfg.AuthorFavorites) {
		t.Errorf("AuthorFavorites count mismatch: got %d, want %d", len(loaded.AuthorFavorites), len(cfg.AuthorFavorites))
	}
}
