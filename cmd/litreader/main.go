package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/cache"
	"github.com/subbass/litreader/internal/config"
	"github.com/subbass/litreader/internal/state"
	"github.com/subbass/litreader/internal/ui"
	"github.com/subbass/litreader/internal/version"
)

func main() {
	// Get the app name for display
	appName := filepath.Base(os.Args[0])

	// Parse command-line flags
	forceRebuild := flag.Bool("u", false, "Force rebuild library cache")
	showVersion := flag.Bool("v", false, "Show version information")
	showVersionLong := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("h", false, "Show help information")
	showHelpLong := flag.Bool("help", false, "Show help information")

	flag.Parse()

	// Handle version flag
	if *showVersion || *showVersionLong {
		fmt.Printf("%s version %s\n", appName, version.AppVersion)
		fmt.Println("A terminal-based text file reader and library manager")
		fmt.Println("\nConfig location: ~/.config/" + appName + "/" + appName + ".conf")
		fmt.Println("Cache location:  ~/.cache/" + appName + "/")
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp || *showHelpLong {
		fmt.Printf("%s - Terminal-based text file reader and library manager\n\n", appName)
		fmt.Println("Usage:")
		fmt.Printf("  %s [options] [file]\n\n", appName)
		fmt.Println("Options:")
		fmt.Println("  -h, --help       Show this help message")
		fmt.Println("  -v, --version    Show version information")
		fmt.Println("  -u               Force rebuild library cache")
		fmt.Println("  [file]           Open a file directly in the viewer (bypasses library)")
		fmt.Println("\nFeatures:")
		fmt.Println("  - Browse and search your text file library")
		fmt.Println("  - Favorites and bookmarks management")
		fmt.Println("  - Author favorites tracking")
		fmt.Println("  - File viewer with pandoc rendering")
		fmt.Println("  - Rating system for stories")
		fmt.Println("\nKeyboard Shortcuts (in Dashboard):")
		fmt.Println("  s - Search library")
		fmt.Println("  f - View favorites")
		fmt.Println("  b - View bookmarks")
		fmt.Println("  a - View favorite authors")
		fmt.Println("  e - Explore all authors")
		fmt.Println("  c - Edit configuration")
		fmt.Println("  l - Open last read file")
		fmt.Println("  q - Quit")
		fmt.Println("\nConfiguration:")
		fmt.Printf("  Config: ~/.config/%s/%s.conf\n", appName, appName)
		fmt.Printf("  Cache:  ~/.cache/%s/\n", appName)
		fmt.Println("\nFor more information, see the README.md file")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Clear file list cache if force rebuild requested
	if *forceRebuild {
		cache.ClearFileListCache()
	}

	// Load cache
	cacheData, err := cache.Load()
	if err != nil {
		// Non-fatal, just create new cache
		cacheData = cache.NewCache()
	}

	// Create application state
	appState := state.NewState(cfg)
	appState.Cache = cacheData

	// Check for direct file argument
	if args := flag.Args(); len(args) > 0 {
		filePath, err := filepath.Abs(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving file path: %v\n", err)
			os.Exit(1)
		}
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "File not found: %s\n", args[0])
			os.Exit(1)
		}
		appState.DirectFile = filePath
	}

	// Create and run TUI
	app := ui.NewApp(appState)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}
