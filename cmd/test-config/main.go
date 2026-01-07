package main

import (
	"fmt"
	"log"

	"github.com/subbass/litreader/internal/config"
)

func main() {
	fmt.Println("Testing config loading from Python config file...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("\n=== Config Loaded Successfully ===")
	fmt.Printf("Search Dir: %s\n", cfg.SearchDir)
	fmt.Printf("Export Dir: %s\n", cfg.ExportDir)
	fmt.Printf("Editor: %s\n", cfg.Editor)
	fmt.Printf("UI Color: %s\n", cfg.UIColor)
	fmt.Printf("Selector Color: %s\n", cfg.SelectorColor)
	fmt.Printf("Last File: %s\n", cfg.LastFile)
	fmt.Printf("Position: %d\n", cfg.Position)
	fmt.Printf("Search Text: %s\n", cfg.SearchText)
	fmt.Printf("Cache Expiry Days: %d\n", cfg.CacheExpiryDays)

	fmt.Printf("\nFavorites: %d\n", len(cfg.Favorites))
	for i, fav := range cfg.Favorites {
		fmt.Printf("  [%d] %s (pos: %d", i+1, fav.Filename, fav.Position)
		if fav.Rating != nil {
			fmt.Printf(", rating: %.2f", *fav.Rating)
		}
		fmt.Println(")")
	}

	fmt.Printf("\nBookmarks: %d\n", len(cfg.Bookmarks))
	for i, bm := range cfg.Bookmarks {
		fmt.Printf("  [%d] %s (pos: %d, note: %s", i+1, bm.Filename, bm.Position, bm.Note)
		if bm.Rating != nil {
			fmt.Printf(", rating: %.2f", *bm.Rating)
		}
		fmt.Println(")")
	}

	fmt.Printf("\nAuthor Favorites: %d\n", len(cfg.AuthorFavorites))
	for i, author := range cfg.AuthorFavorites {
		fmt.Printf("  [%d] %s\n", i+1, author)
	}

	fmt.Println("\n=== Test Complete ===")
}
