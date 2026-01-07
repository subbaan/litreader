package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/state"
)

// AuthorsModel represents the author favorites view
type AuthorsModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	authors  []string
	cursor   int
	topRow   int
	sortMode int // 0=order added, 1=alphabetical
}

// NewAuthorsModel creates a new authors view
func NewAuthorsModel(s *state.State, styles *Styles) *AuthorsModel {
	am := &AuthorsModel{
		state:    s,
		styles:   styles,
		authors:  make([]string, len(s.Config.AuthorFavorites)),
		cursor:   0,
		topRow:   0,
		sortMode: 1, // Default to alphabetical
	}

	// Copy authors list so we can sort without modifying config
	copy(am.authors, s.Config.AuthorFavorites)

	// Apply default sort
	am.sortAuthors()

	return am
}

// Init initializes the authors view
func (am *AuthorsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the authors view
func (am *AuthorsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Navigation
		case "up", "k":
			if am.cursor > 0 {
				am.cursor--
				if am.cursor < am.topRow {
					am.topRow--
				}
			}

		case "down", "j":
			if am.cursor < len(am.authors)-1 {
				am.cursor++
				availableLines := am.getAvailableLines()
				if am.cursor >= am.topRow+availableLines {
					am.topRow++
				}
			}

		// Open selected author (browse their files)
		case "enter", "right":
			if len(am.authors) > 0 && am.cursor < len(am.authors) {
				author := am.authors[am.cursor]
				// Find the folder for this author in the search directory
				authorFolder := filepath.Join(am.state.Config.SearchDir, author)
				return am, func() tea.Msg {
					return openAuthorFilesMsg{
						authorFolder: authorFolder,
					}
				}
			}

		// Toggle sort mode
		case "t":
			am.sortMode = (am.sortMode + 1) % 2
			am.sortAuthors()

		// Delete author
		case "backspace", "delete":
			if len(am.authors) > 0 && am.cursor < len(am.authors) {
				authorToRemove := am.authors[am.cursor]
				// Remove from displayed list
				am.authors = append(am.authors[:am.cursor], am.authors[am.cursor+1:]...)

				// Remove from config (find it in the original order)
				for i, author := range am.state.Config.AuthorFavorites {
					if author == authorToRemove {
						am.state.Config.AuthorFavorites = append(
							am.state.Config.AuthorFavorites[:i],
							am.state.Config.AuthorFavorites[i+1:]...,
						)
						break
					}
				}
				am.state.Config.Save()

				// Adjust cursor
				if am.cursor >= len(am.authors) && len(am.authors) > 0 {
					am.cursor = len(am.authors) - 1
				}
			}

		// Return to dashboard (handled by app.go navigation)
		case "q":
			// Don't handle here - let app.go handle navigation
			break
		}
	}

	return am, nil
}

// View renders the authors view
func (am *AuthorsModel) View() string {
	if am.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := fmt.Sprintf("Favorite Authors (%d)", len(am.authors))
	b.WriteString(am.styles.RenderTitle(title, am.width))
	b.WriteString("\n")

	// Sort indicator
	sortModes := []string{"Order Added", "Alphabetical"}
	sortText := sortModes[am.sortMode]
	statusBar := fmt.Sprintf("Sort by: %s", sortText)
	b.WriteString(am.styles.StatusBar.Render(statusBar))
	b.WriteString("\n\n")

	if len(am.authors) == 0 {
		b.WriteString("  No favorite authors yet. Press 'q' to go back.\n")
	} else {
		availableLines := am.getAvailableLines()
		displayCount := min(len(am.authors), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := am.topRow + idx
			if itemIdx >= len(am.authors) {
				break
			}

			author := am.authors[itemIdx]
			isSelected := itemIdx == am.cursor

			line := fmt.Sprintf("  %s", author)

			// Truncate if too long
			if len(line) > am.width-1 {
				line = line[:am.width-4] + "..."
			}

			if isSelected {
				b.WriteString(am.styles.ListCursor.Render(line))
			} else {
				b.WriteString(am.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	currentLines := strings.Count(b.String(), "\n")
	for currentLines < am.height-2 {
		b.WriteString("\n")
		currentLines++
	}

	// Help bar
	helpText := "↑↓:Scroll ↵:Browse t:Sort Del:Remove q:Back"
	b.WriteString(am.styles.RenderHelpBar(helpText, am.width))

	return b.String()
}

// Helper methods

func (am *AuthorsModel) getAvailableLines() int {
	// Height minus title(1), status(1), gap(1), and help(1)
	return am.height - 4
}

func (am *AuthorsModel) sortAuthors() {
	switch am.sortMode {
	case 0: // Order added (restore original order)
		am.authors = make([]string, len(am.state.Config.AuthorFavorites))
		copy(am.authors, am.state.Config.AuthorFavorites)
	case 1: // Alphabetical (case-insensitive)
		sort.Slice(am.authors, func(i, j int) bool {
			return strings.ToLower(am.authors[i]) < strings.ToLower(am.authors[j])
		})
	}
}

// Ensure AuthorsModel implements tea.Model
var _ tea.Model = (*AuthorsModel)(nil)
