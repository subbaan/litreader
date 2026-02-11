package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/models"
	"github.com/subbass/litreader/internal/state"
)

// FavoritesModel represents the favorites list view
type FavoritesModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	favorites []models.Favorite
	cursor    int
	topRow    int

	sortMode int // 0=name, 1=date, 2=rating
}

// NewFavoritesModel creates a new favorites view
func NewFavoritesModel(s *state.State, styles *Styles) *FavoritesModel {
	fm := &FavoritesModel{
		state:     s,
		styles:    styles,
		favorites: s.Config.Favorites,
		cursor:    0,
		topRow:    0,
		sortMode:  0,
	}

	return fm
}

// Init initializes the favorites view
func (fm *FavoritesModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the favorites view
func (fm *FavoritesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Navigation
		case "up", "k":
			if fm.cursor > 0 {
				fm.cursor--
				if fm.cursor < fm.topRow {
					fm.topRow--
				}
			}

		case "down", "j":
			if fm.cursor < len(fm.favorites)-1 {
				fm.cursor++
				availableLines := fm.getAvailableLines()
				if fm.cursor >= fm.topRow+availableLines {
					fm.topRow++
				}
			}

		case "pgdown":
			fm.cursor = min(fm.cursor+10, len(fm.favorites)-1)
			availableLines := fm.getAvailableLines()
			fm.topRow = min(fm.topRow+10, max(0, len(fm.favorites)-availableLines))

		case "pgup":
			fm.cursor = max(fm.cursor-10, 0)
			fm.topRow = max(fm.topRow-10, 0)

		// Open selected favorite
		case "enter", "right":
			if len(fm.favorites) > 0 && fm.cursor < len(fm.favorites) {
				fav := fm.favorites[fm.cursor]
				return fm, func() tea.Msg {
					return openFileMsg{
						filename:   fav.Filename,
						position:   fav.Position,
						searchText: fav.SearchText,
					}
				}
			}

		// Sort mode
		case "t":
			fm.sortMode = (fm.sortMode + 1) % 3
			// Skip rating mode (2) when ShowRatings is false
			if !fm.state.Config.ShowRatings && fm.sortMode == 2 {
				fm.sortMode = 0
			}
			fm.sortFavorites()

		// Delete favorite
		case "backspace", "delete":
			if len(fm.favorites) > 0 && fm.cursor < len(fm.favorites) {
				// Remove from list
				fm.favorites = append(fm.favorites[:fm.cursor], fm.favorites[fm.cursor+1:]...)
				fm.state.Config.Favorites = fm.favorites
				fm.state.FavoritesVersion++
				fm.state.Config.Save()

				// Adjust cursor
				if fm.cursor >= len(fm.favorites) && len(fm.favorites) > 0 {
					fm.cursor = len(fm.favorites) - 1
				}
			}

		// Return to dashboard (handled by app.go navigation)
		case "q":
			// Don't handle here - let app.go handle navigation
			break
		}
	}

	return fm, nil
}

// View renders the favorites view
func (fm *FavoritesModel) View() string {
	if fm.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	sortNames := []string{"Name", "Date", "Rating"}
	sortName := sortNames[fm.sortMode]
	title := fmt.Sprintf("Favorites (%d) - Sort: %s", len(fm.favorites), sortName)
	b.WriteString(fm.styles.RenderTitle(title, fm.width))
	b.WriteString("\n\n")

	if len(fm.favorites) == 0 {
		b.WriteString("  No favorites yet. Press 'q' to go back.\n")
	} else {
		availableLines := fm.getAvailableLines()
		displayCount := min(len(fm.favorites), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := fm.topRow + idx
			if itemIdx >= len(fm.favorites) {
				break
			}

			fav := fm.favorites[itemIdx]
			isSelected := itemIdx == fm.cursor

			// Get file size
			var fileSize float64
			if info, err := os.Stat(fav.Filename); err == nil {
				fileSize = float64(info.Size()) / 1024.0 // KB
			}

			// Get author (parent directory name)
			author := filepath.Base(filepath.Dir(fav.Filename))

			filename := filepath.Base(fav.Filename)
			var line string
			if fm.state.Config.ShowRatings {
				// Format: [4.68] | 123.45 KB | AuthorName | filename.txt
				ratingStr := "N/A "
				if fav.Rating != nil {
					ratingStr = fmt.Sprintf("%.2f", *fav.Rating)
				}
				line = fmt.Sprintf("  [%s] | %10.2f KB | %-20s | %s", ratingStr, fileSize, author, filename)
			} else {
				// Format: 123.45 KB | AuthorName | filename.txt
				line = fmt.Sprintf("  %10.2f KB | %-20s | %s", fileSize, author, filename)
			}

			// Truncate if too long
			if len(line) > fm.width-1 {
				line = line[:fm.width-4] + "..."
			}

			if isSelected {
				b.WriteString(fm.styles.ListCursor.Render(line))
			} else {
				b.WriteString(fm.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	currentLines := strings.Count(b.String(), "\n")
	for currentLines < fm.height-2 {
		b.WriteString("\n")
		currentLines++
	}

	// Help bar
	helpText := "↑↓:Scroll ↵:Open t:Sort Del:Remove q:Back"
	b.WriteString(fm.styles.RenderHelpBar(helpText, fm.width))

	return b.String()
}

// Helper methods

func (fm *FavoritesModel) getAvailableLines() int {
	// Height minus title(2) and help(1)
	return fm.height - 3
}

func (fm *FavoritesModel) sortFavorites() {
	// Simple bubble sort
	items := fm.favorites
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			shouldSwap := false

			switch fm.sortMode {
			case 0: // Name
				shouldSwap = filepath.Base(items[i].Filename) > filepath.Base(items[j].Filename)
			case 1: // Date
				shouldSwap = items[i].DateAdded.Before(items[j].DateAdded)
			case 2: // Rating
				rating1 := 0.0
				rating2 := 0.0
				if items[i].Rating != nil {
					rating1 = *items[i].Rating
				}
				if items[j].Rating != nil {
					rating2 = *items[j].Rating
				}
				shouldSwap = rating1 < rating2
			}

			if shouldSwap {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// Ensure FavoritesModel implements tea.Model
var _ tea.Model = (*FavoritesModel)(nil)
