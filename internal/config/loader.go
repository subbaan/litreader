package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/subbass/litreader/internal/models"
)

// Load reads and parses the config file, maintaining Python compatibility
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return NewDefaultConfig(), nil
	}

	config := NewDefaultConfig()

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	favoriteMap := make(map[int]*models.Favorite)
	bookmarkMap := make(map[int]*models.Bookmark)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first '=' only
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Parse favorites (favorite_1_filename, favorite_1_position, etc.)
		if strings.HasPrefix(key, "favorite_") {
			parseFavoriteField(key, value, favoriteMap)
			continue
		}

		// Parse bookmarks (bookmark_1_filename, bookmark_1_position, etc.)
		if strings.HasPrefix(key, "bookmark_") {
			parseBookmarkField(key, value, bookmarkMap)
			continue
		}

		// Parse author favorites (author_favorite_1=name)
		if strings.HasPrefix(key, "author_favorite_") {
			config.AuthorFavorites = append(config.AuthorFavorites, value)
			continue
		}

		// Parse regular config fields
		switch key {
		case "SEARCH_DIR":
			config.SearchDir = expandHome(value)
		case "EXPORT_DIR":
			config.ExportDir = expandHome(value)
		case "ui_color":
			config.UIColor = value
		case "selector_color":
			config.SelectorColor = value
		case "selector_reverse":
			config.SelectorReverse = parseBool(value)
		case "selector_bold":
			config.SelectorBold = parseBool(value)
		case "selector_reverse_color":
			config.SelectorReverseColor = value
		case "content_color":
			config.ContentColor = value
		case "content_bold":
			config.ContentBold = parseBool(value)
		case "last_file":
			config.LastFile = value
		case "position":
			config.Position, _ = strconv.Atoi(value)
		case "search_text":
			config.SearchText = value
		case "CACHE_EXPIRY_DAYS":
			if days, err := strconv.Atoi(value); err == nil {
				config.CacheExpiryDays = days
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Convert maps to slices in order
	config.Favorites = favoritesMapToSlice(favoriteMap)
	config.Bookmarks = bookmarksMapToSlice(bookmarkMap)

	return config, nil
}

// parseFavoriteField parses a single favorite field and updates the map
func parseFavoriteField(key, value string, favoriteMap map[int]*models.Favorite) {
	// Extract index from key (e.g., "favorite_1_filename" -> 1)
	parts := strings.Split(key, "_")
	if len(parts) < 3 {
		return
	}

	idx, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}

	// Ensure favorite exists in map
	if favoriteMap[idx] == nil {
		favoriteMap[idx] = &models.Favorite{}
	}

	fav := favoriteMap[idx]
	fieldName := strings.Join(parts[2:], "_")

	switch fieldName {
	case "filename":
		fav.Filename = value
	case "position":
		fav.Position, _ = strconv.Atoi(value)
	case "search_text":
		fav.SearchText = value
	case "date_added":
		if t, err := time.Parse(time.RFC3339, value); err == nil {
			fav.DateAdded = t
		}
	case "rating":
		if value != "" && value != "None" {
			if rating, err := strconv.ParseFloat(value, 64); err == nil {
				fav.Rating = &rating
			}
		}
	}
}

// parseBookmarkField parses a single bookmark field and updates the map
func parseBookmarkField(key, value string, bookmarkMap map[int]*models.Bookmark) {
	// Extract index from key (e.g., "bookmark_1_filename" -> 1)
	parts := strings.Split(key, "_")
	if len(parts) < 3 {
		return
	}

	idx, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}

	// Ensure bookmark exists in map
	if bookmarkMap[idx] == nil {
		bookmarkMap[idx] = &models.Bookmark{}
	}

	bm := bookmarkMap[idx]
	fieldName := strings.Join(parts[2:], "_")

	switch fieldName {
	case "filename":
		bm.Filename = value
	case "position":
		bm.Position, _ = strconv.Atoi(value)
	case "note":
		bm.Note = value
	case "date_added":
		if t, err := time.Parse(time.RFC3339, value); err == nil {
			bm.DateAdded = t
		}
	case "rating":
		if value != "" && value != "None" {
			if rating, err := strconv.ParseFloat(value, 64); err == nil {
				bm.Rating = &rating
			}
		}
	}
}

// favoritesMapToSlice converts the map to a slice, filtering incomplete entries
func favoritesMapToSlice(m map[int]*models.Favorite) []models.Favorite {
	if len(m) == 0 {
		return []models.Favorite{}
	}

	// Find max index
	maxIdx := 0
	for idx := range m {
		if idx > maxIdx {
			maxIdx = idx
		}
	}

	result := make([]models.Favorite, 0, len(m))
	for i := 1; i <= maxIdx; i++ {
		if fav, exists := m[i]; exists && fav.Filename != "" {
			result = append(result, *fav)
		}
	}

	return result
}

// bookmarksMapToSlice converts the map to a slice, filtering incomplete entries
func bookmarksMapToSlice(m map[int]*models.Bookmark) []models.Bookmark {
	if len(m) == 0 {
		return []models.Bookmark{}
	}

	// Find max index
	maxIdx := 0
	for idx := range m {
		if idx > maxIdx {
			maxIdx = idx
		}
	}

	result := make([]models.Bookmark, 0, len(m))
	for i := 1; i <= maxIdx; i++ {
		if bm, exists := m[i]; exists && bm.Filename != "" {
			result = append(result, *bm)
		}
	}

	return result
}

// parseBool converts string to bool (Python-style)
func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes"
}

// expandHome expands ~ to home directory
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	return filepath.Join(home, path[2:])
}
