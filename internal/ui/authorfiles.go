package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/library"
	"github.com/subbass/litreader/internal/state"
)

// AuthorFilesModel represents the author files browser view
type AuthorFilesModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	authorName   string
	authorFolder string
	files        []string
	filtered     []string
	favoriteSet  map[string]bool
	favoritesVer int
	cursor       int
	topRow       int
	sortMode     int // 0=name, 1=size asc, 2=size desc

	// Filtering
	filterInput textinput.Model
	filterMode  bool

	// Scroll burst guard
	lastScrollTime time.Time
}

// NewAuthorFilesModel creates a new author files view
func NewAuthorFilesModel(s *state.State, styles *Styles, authorFolder string) *AuthorFilesModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Width = 40

	afm := &AuthorFilesModel{
		state:        s,
		styles:       styles,
		authorFolder: authorFolder,
		authorName:   filepath.Base(authorFolder),
		favoriteSet:  make(map[string]bool),
		favoritesVer: -1,
		cursor:       0,
		topRow:       0,
		sortMode:     0,
		filterInput:  ti,
		filterMode:   false,
	}

	afm.refreshFavorites()

	// Find all files in this author's folder
	afm.findAuthorFiles()

	return afm
}

// findAuthorFiles lists all files in the author's directory
func (afm *AuthorFilesModel) findAuthorFiles() {
	afm.files = []string{}

	// List all files in the author folder
	for _, file := range afm.state.AllFiles {
		// Check if file is in this author's folder
		if filepath.Dir(file) == afm.authorFolder {
			afm.files = append(afm.files, file)
		}
	}

	// Sort by name initially
	afm.sortFiles()
}

// applyFilter filters files based on current filter text
func (afm *AuthorFilesModel) applyFilter() {
	filterText := strings.ToLower(strings.TrimSpace(afm.filterInput.Value()))

	if filterText == "" {
		afm.filtered = afm.files
	} else {
		afm.filtered = make([]string, 0)
		for _, file := range afm.files {
			if strings.Contains(strings.ToLower(filepath.Base(file)), filterText) {
				afm.filtered = append(afm.filtered, file)
			}
		}
	}

	// Reset cursor if out of bounds
	if afm.cursor >= len(afm.filtered) {
		afm.cursor = 0
		afm.topRow = 0
	}
}

// Init initializes the author files view
func (afm *AuthorFilesModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the author files view
func (afm *AuthorFilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	afm.refreshFavorites()

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if afm.filterMode {
			// Handle filter mode keys
			switch msg.String() {
			case "esc":
				// Exit filter mode and clear filter
				afm.filterMode = false
				afm.filterInput.Blur()
				afm.filterInput.SetValue("")
				afm.applyFilter()
				return afm, nil

			case "enter":
				// Exit filter mode but keep filter
				afm.filterMode = false
				afm.filterInput.Blur()
				return afm, nil

			case "up":
				// Navigate while still in filter mode
				now := time.Now()
				if !afm.lastScrollTime.IsZero() && now.Sub(afm.lastScrollTime) < 15*time.Millisecond {
					return afm, nil // XInput2 burst accumulation — discard
				}
				afm.lastScrollTime = now
				if afm.cursor > 0 {
					afm.cursor--
					if afm.cursor < afm.topRow {
						afm.topRow--
					}
				}
				return afm, nil

			case "down":
				// Navigate while still in filter mode
				now := time.Now()
				if !afm.lastScrollTime.IsZero() && now.Sub(afm.lastScrollTime) < 15*time.Millisecond {
					return afm, nil // XInput2 burst accumulation — discard
				}
				afm.lastScrollTime = now
				if afm.cursor < len(afm.filtered)-1 {
					afm.cursor++
					availableLines := afm.getAvailableLines()
					if afm.cursor >= afm.topRow+availableLines {
						afm.topRow++
					}
				}
				return afm, nil

			default:
				// Update text input
				afm.filterInput, cmd = afm.filterInput.Update(msg)
				afm.applyFilter()
				return afm, cmd
			}
		}

		switch msg.String() {
		// Navigation
		case "up", "k":
			now := time.Now()
			if !afm.lastScrollTime.IsZero() && now.Sub(afm.lastScrollTime) < 15*time.Millisecond {
				break // XInput2 burst accumulation — discard
			}
			afm.lastScrollTime = now
			if afm.cursor > 0 {
				afm.cursor--
				if afm.cursor < afm.topRow {
					afm.topRow--
				}
			}

		case "down", "j":
			now := time.Now()
			if !afm.lastScrollTime.IsZero() && now.Sub(afm.lastScrollTime) < 15*time.Millisecond {
				break // XInput2 burst accumulation — discard
			}
			afm.lastScrollTime = now
			if afm.cursor < len(afm.filtered)-1 {
				afm.cursor++
				availableLines := afm.getAvailableLines()
				if afm.cursor >= afm.topRow+availableLines {
					afm.topRow++
				}
			}

		case "pgdown":
			afm.cursor = min(afm.cursor+10, len(afm.filtered)-1)
			availableLines := afm.getAvailableLines()
			afm.topRow = min(afm.topRow+10, max(0, len(afm.filtered)-availableLines))

		case "pgup":
			afm.cursor = max(afm.cursor-10, 0)
			afm.topRow = max(afm.topRow-10, 0)

		// Enter filter mode
		case "/":
			afm.filterMode = true
			afm.filterInput.Focus()
			return afm, textinput.Blink

		// Sort files
		case "t":
			afm.sortMode = (afm.sortMode + 1) % 3
			afm.sortFiles()

		// Open selected file
		case "enter", "right":
			if len(afm.filtered) > 0 && afm.cursor < len(afm.filtered) {
				file := afm.filtered[afm.cursor]
				return afm, func() tea.Msg {
					return openFileMsg{
						filename:   file,
						position:   0,
						searchText: "",
					}
				}
			}

		// Return to authors list (handled by app.go navigation)
		case "q":
			// Don't handle here - let app.go handle navigation
			break
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			now := time.Now()
			if !afm.lastScrollTime.IsZero() && now.Sub(afm.lastScrollTime) < 15*time.Millisecond {
				break
			}
			afm.lastScrollTime = now
			availableLines := afm.getAvailableLines()
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if afm.cursor > 0 {
					afm.cursor--
					if afm.cursor < afm.topRow {
						afm.topRow = afm.cursor
					}
				}
			case tea.MouseButtonWheelDown:
				if afm.cursor < len(afm.filtered)-1 {
					afm.cursor++
					if afm.cursor >= afm.topRow+availableLines {
						afm.topRow++
					}
				}
			}
		}
	}

	return afm, nil
}

// View renders the author files view
func (afm *AuthorFilesModel) View() string {
	if afm.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := fmt.Sprintf("Files by %s (%d)", afm.authorName, len(afm.filtered))
	if len(afm.files) != len(afm.filtered) {
		title = fmt.Sprintf("Files by %s (%d/%d)", afm.authorName, len(afm.filtered), len(afm.files))
	}
	if len(title) > afm.width-4 {
		title = title[:afm.width-7] + "..."
	}
	b.WriteString(afm.styles.RenderTitle(title, afm.width))
	b.WriteString("\n\n")

	// Filter input or status bar showing sort order and file count
	if afm.filterMode {
		b.WriteString("  Filter: ")
		b.WriteString(afm.filterInput.View())
		b.WriteString("\n")
	} else {
		sortModes := []string{"Name", "Size Asc", "Size Desc"}
		sortText := sortModes[afm.sortMode]
		var statusBar string
		if len(afm.filtered) > 0 {
			statusBar = fmt.Sprintf("Sort by: %s | Result %d/%d", sortText, afm.cursor+1, len(afm.filtered))
		} else if len(afm.files) > 0 {
			statusBar = "No files match the filter"
		} else {
			statusBar = "No files found"
		}
		if afm.filterInput.Value() != "" {
			statusBar = fmt.Sprintf("%s | Filter: \"%s\"", statusBar, afm.filterInput.Value())
		}
		// Center it
		padding := (afm.width - len(statusBar)) / 2
		if padding < 0 {
			padding = 0
		}
		centeredStatus := strings.Repeat(" ", padding) + statusBar + strings.Repeat(" ", afm.width-padding-len(statusBar))
		if len(centeredStatus) > afm.width {
			centeredStatus = centeredStatus[:afm.width]
		}
		b.WriteString(afm.styles.StatusBar.Render(centeredStatus))
		b.WriteString("\n")
	}

	// One-line gap
	b.WriteString("\n")

	if len(afm.filtered) == 0 {
		if len(afm.files) == 0 {
			b.WriteString(fmt.Sprintf("  No files found in '%s'. Press 'q' to go back.\n", afm.authorName))
		} else {
			b.WriteString("  No files match the filter. Press 'Esc' to clear.\n")
		}
	} else {

		availableLines := afm.getAvailableLines()
		displayCount := min(len(afm.filtered), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := afm.topRow + idx
			if itemIdx >= len(afm.filtered) {
				break
			}

			file := afm.filtered[itemIdx]
			isSelected := itemIdx == afm.cursor

			// Get file size
			var fileSize float64
			if info, err := os.Stat(file); err == nil {
				fileSize = float64(info.Size()) / 1024.0 // KB
			}

			// Format: favorite | size | [rating |] filename
			filename := filepath.Base(file)
			heart := " "
			if afm.favoriteSet[file] {
				heart = "\uf004"
			}

			var line string
			if afm.state.Config.ShowRatings {
				rating, err := library.ExtractRating(file)
				ratingStr := "N/A"
				if err == nil && rating > 0 {
					ratingStr = fmt.Sprintf("%.2f", rating)
				}
				line = fmt.Sprintf("  %s %10.2f KB | %4s | %s", heart, fileSize, ratingStr, filename)
			} else {
				line = fmt.Sprintf("  %s %10.2f KB | %s", heart, fileSize, filename)
			}

			// Truncate if too long
			if len(line) > afm.width-1 {
				line = line[:afm.width-4] + "..."
			}

			if isSelected {
				b.WriteString(afm.styles.ListCursor.Render(line))
			} else {
				b.WriteString(afm.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	currentLines := strings.Count(b.String(), "\n")
	for currentLines < afm.height-2 {
		b.WriteString("\n")
		currentLines++
	}

	// Help bar - context-aware
	var helpText string
	if afm.filterMode {
		helpText = "Type to filter | ↑↓:Navigate | ↵:Keep filter | Esc:Clear"
	} else {
		helpText = "/:Filter ↑↓:Scroll ↵:Open t:Sort q:Back"
	}
	b.WriteString(afm.styles.RenderHelpBar(helpText, afm.width))

	return b.String()
}

// Helper methods

func (afm *AuthorFilesModel) getAvailableLines() int {
	// Height minus title(1), status(1), gap(1), help(1)
	return afm.height - 4
}

// sortFiles sorts the files based on current sort mode
func (afm *AuthorFilesModel) sortFiles() {
	switch afm.sortMode {
	case 0: // Name
		sort.Slice(afm.files, func(i, j int) bool {
			return filepath.Base(afm.files[i]) < filepath.Base(afm.files[j])
		})
	case 1: // Size ascending
		sort.Slice(afm.files, func(i, j int) bool {
			sizeI, _ := afm.getFileSize(afm.files[i])
			sizeJ, _ := afm.getFileSize(afm.files[j])
			return sizeI < sizeJ
		})
	case 2: // Size descending
		sort.Slice(afm.files, func(i, j int) bool {
			sizeI, _ := afm.getFileSize(afm.files[i])
			sizeJ, _ := afm.getFileSize(afm.files[j])
			return sizeI > sizeJ
		})
	}
	afm.applyFilter()
}

func (afm *AuthorFilesModel) getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (afm *AuthorFilesModel) refreshFavorites() {
	if afm.favoritesVer == afm.state.FavoritesVersion && afm.favoriteSet != nil {
		return
	}
	afm.favoriteSet = make(map[string]bool)
	for _, fav := range afm.state.Config.Favorites {
		afm.favoriteSet[fav.Filename] = true
	}
	afm.favoritesVer = afm.state.FavoritesVersion
}

// Ensure AuthorFilesModel implements tea.Model
var _ tea.Model = (*AuthorFilesModel)(nil)
