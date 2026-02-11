package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	favoriteSet  map[string]bool
	favoritesVer int
	cursor       int
	topRow       int
	sortMode     int // 0=name, 1=size asc, 2=size desc
}

// NewAuthorFilesModel creates a new author files view
func NewAuthorFilesModel(s *state.State, styles *Styles, authorFolder string) *AuthorFilesModel {
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

// Init initializes the author files view
func (afm *AuthorFilesModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the author files view
func (afm *AuthorFilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	afm.refreshFavorites()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Navigation
		case "up", "k":
			if afm.cursor > 0 {
				afm.cursor--
				if afm.cursor < afm.topRow {
					afm.topRow--
				}
			}

		case "down", "j":
			if afm.cursor < len(afm.files)-1 {
				afm.cursor++
				availableLines := afm.getAvailableLines()
				if afm.cursor >= afm.topRow+availableLines {
					afm.topRow++
				}
			}

		case "pgdown":
			afm.cursor = min(afm.cursor+10, len(afm.files)-1)
			availableLines := afm.getAvailableLines()
			afm.topRow = min(afm.topRow+10, max(0, len(afm.files)-availableLines))

		case "pgup":
			afm.cursor = max(afm.cursor-10, 0)
			afm.topRow = max(afm.topRow-10, 0)

		// Sort files
		case "t":
			afm.sortMode = (afm.sortMode + 1) % 3
			afm.sortFiles()

		// Open selected file
		case "enter", "right":
			if len(afm.files) > 0 && afm.cursor < len(afm.files) {
				file := afm.files[afm.cursor]
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
	title := fmt.Sprintf("Files by %s", afm.authorName)
	if len(title) > afm.width-4 {
		title = title[:afm.width-7] + "..."
	}
	b.WriteString(afm.styles.RenderTitle(title, afm.width))
	b.WriteString("\n\n")

	// Status bar showing sort order and file count
	sortModes := []string{"Name", "Size Asc", "Size Desc"}
	sortText := sortModes[afm.sortMode]
	var statusBar string
	if len(afm.files) > 0 {
		statusBar = fmt.Sprintf("Sort by: %s | Result %d/%d", sortText, afm.cursor+1, len(afm.files))
	} else {
		statusBar = "No files found"
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

	// One-line gap
	b.WriteString("\n")

	if len(afm.files) == 0 {
		b.WriteString(fmt.Sprintf("  No files found in '%s'. Press 'q' to go back.\n", afm.authorName))
	} else {

		availableLines := afm.getAvailableLines()
		displayCount := min(len(afm.files), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := afm.topRow + idx
			if itemIdx >= len(afm.files) {
				break
			}

			file := afm.files[itemIdx]
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

	// Help bar
	helpText := "↑↓:Scroll ↵:Open t:Sort q:Back"
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
