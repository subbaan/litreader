package config

import (
	"fmt"
	"os"
)

// Save writes the config to disk in Python-compatible format
func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write simple config fields
	fmt.Fprintf(file, "SEARCH_DIR=%s\n", c.SearchDir)
	fmt.Fprintf(file, "EXPORT_DIR=%s\n", c.ExportDir)
	fmt.Fprintf(file, "ui_color=%s\n", c.UIColor)
	fmt.Fprintf(file, "selector_color=%s\n", c.SelectorColor)
	fmt.Fprintf(file, "selector_reverse=%s\n", formatBool(c.SelectorReverse))
	fmt.Fprintf(file, "selector_bold=%s\n", formatBool(c.SelectorBold))
	fmt.Fprintf(file, "selector_reverse_color=%s\n", c.SelectorReverseColor)
	fmt.Fprintf(file, "content_color=%s\n", c.ContentColor)
	fmt.Fprintf(file, "content_bold=%s\n", formatBool(c.ContentBold))
	fmt.Fprintf(file, "viewer_help_bar=%s\n", formatBool(c.ShowViewerHelpBar))
	fmt.Fprintf(file, "show_ratings=%s\n", formatBool(c.ShowRatings))
	fmt.Fprintf(file, "last_file=%s\n", c.LastFile)
	fmt.Fprintf(file, "position=%d\n", c.Position)
	fmt.Fprintf(file, "search_text=%s\n", c.SearchText)
	fmt.Fprintf(file, "CACHE_EXPIRY_DAYS=%d\n", c.CacheExpiryDays)

	// Write favorites
	for i, fav := range c.Favorites {
		idx := i + 1
		fmt.Fprintf(file, "favorite_%d_filename=%s\n", idx, fav.Filename)
		fmt.Fprintf(file, "favorite_%d_position=%d\n", idx, fav.Position)
		fmt.Fprintf(file, "favorite_%d_search_text=%s\n", idx, fav.SearchText)

		dateStr := ""
		if !fav.DateAdded.IsZero() {
			dateStr = fav.DateAdded.Format("2006-01-02T15:04:05Z07:00")
		}
		fmt.Fprintf(file, "favorite_%d_date_added=%s\n", idx, dateStr)

		ratingStr := ""
		if fav.Rating != nil {
			ratingStr = fmt.Sprintf("%.2f", *fav.Rating)
		}
		fmt.Fprintf(file, "favorite_%d_rating=%s\n", idx, ratingStr)
	}

	// Write bookmarks
	for i, bm := range c.Bookmarks {
		idx := i + 1
		fmt.Fprintf(file, "bookmark_%d_filename=%s\n", idx, bm.Filename)
		fmt.Fprintf(file, "bookmark_%d_position=%d\n", idx, bm.Position)
		fmt.Fprintf(file, "bookmark_%d_note=%s\n", idx, bm.Note)

		dateStr := ""
		if !bm.DateAdded.IsZero() {
			dateStr = bm.DateAdded.Format("2006-01-02T15:04:05Z07:00")
		}
		fmt.Fprintf(file, "bookmark_%d_date_added=%s\n", idx, dateStr)

		ratingStr := ""
		if bm.Rating != nil {
			ratingStr = fmt.Sprintf("%.2f", *bm.Rating)
		}
		fmt.Fprintf(file, "bookmark_%d_rating=%s\n", idx, ratingStr)
	}

	// Write author favorites
	for i, author := range c.AuthorFavorites {
		idx := i + 1
		fmt.Fprintf(file, "author_favorite_%d=%s\n", idx, author)
	}

	return nil
}

// formatBool converts bool to Python-style string
func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
