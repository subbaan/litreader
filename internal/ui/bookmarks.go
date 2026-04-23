package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/models"
	"github.com/subbass/litreader/internal/state"
)

// BookmarksModel represents the bookmarks list view
type BookmarksModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	bookmarks []models.Bookmark
	cursor    int
	topRow    int
}

// NewBookmarksModel creates a new bookmarks view
func NewBookmarksModel(s *state.State, styles *Styles) *BookmarksModel {
	bm := &BookmarksModel{
		state:     s,
		styles:    styles,
		bookmarks: s.Config.Bookmarks,
		cursor:    0,
		topRow:    0,
	}

	return bm
}

// Init initializes the bookmarks view
func (bm *BookmarksModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the bookmarks view
func (bm *BookmarksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Navigation
		case "up", "k":
			if len(bm.bookmarks) > 0 {
				if bm.cursor > 0 {
					bm.cursor--
				} else {
					bm.cursor = len(bm.bookmarks) - 1
					availableLines := bm.getAvailableLines()
					bm.topRow = max(0, len(bm.bookmarks)-availableLines)
				}
				if bm.cursor < bm.topRow {
					bm.topRow = bm.cursor
				}
			}

		case "down", "j":
			if len(bm.bookmarks) > 0 {
				if bm.cursor < len(bm.bookmarks)-1 {
					bm.cursor++
				} else {
					bm.cursor = 0
					bm.topRow = 0
				}
				availableLines := bm.getAvailableLines()
				if bm.cursor >= bm.topRow+availableLines {
					bm.topRow++
				}
			}

		case "pgdown":
			bm.cursor = min(bm.cursor+10, len(bm.bookmarks)-1)
			availableLines := bm.getAvailableLines()
			bm.topRow = min(bm.topRow+10, max(0, len(bm.bookmarks)-availableLines))

		case "pgup":
			bm.cursor = max(bm.cursor-10, 0)
			bm.topRow = max(bm.topRow-10, 0)

		// Open selected bookmark
		case "enter", "right":
			if len(bm.bookmarks) > 0 && bm.cursor < len(bm.bookmarks) {
				bookmark := bm.bookmarks[bm.cursor]
				return bm, func() tea.Msg {
					return openFileMsg{
						filename:   bookmark.Filename,
						position:   bookmark.Position,
						searchText: "",
					}
				}
			}

		// Delete bookmark
		case "backspace", "delete":
			if len(bm.bookmarks) > 0 && bm.cursor < len(bm.bookmarks) {
				// Remove from list
				bm.bookmarks = append(bm.bookmarks[:bm.cursor], bm.bookmarks[bm.cursor+1:]...)
				bm.state.Config.Bookmarks = bm.bookmarks
				bm.state.Config.Save()

				// Adjust cursor
				if bm.cursor >= len(bm.bookmarks) && len(bm.bookmarks) > 0 {
					bm.cursor = len(bm.bookmarks) - 1
				}
			}

		// Return to dashboard (handled by app.go navigation)
		case "q":
			// Don't handle here - let app.go handle navigation
			break
		}
	}

	return bm, nil
}

// View renders the bookmarks view
func (bm *BookmarksModel) View() string {
	if bm.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := fmt.Sprintf("Bookmarks (%d)", len(bm.bookmarks))
	b.WriteString(bm.styles.RenderTitle(title, bm.width))
	b.WriteString("\n\n")

	if len(bm.bookmarks) == 0 {
		b.WriteString("  No bookmarks yet. Press 'q' to go back.\n")
	} else {
		availableLines := bm.getAvailableLines()
		displayCount := min(len(bm.bookmarks), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := bm.topRow + idx
			if itemIdx >= len(bm.bookmarks) {
				break
			}

			bookmark := bm.bookmarks[itemIdx]
			isSelected := itemIdx == bm.cursor

			filename := filepath.Base(bookmark.Filename)
			note := bookmark.Note
			if len(note) > 30 {
				note = note[:27] + "..."
			}

			var line string
			if bm.state.Config.ShowRatings {
				// Format: [4.68] filename.txt: "note text" (Line 123)
				ratingStr := "N/A "
				if bookmark.Rating != nil {
					ratingStr = fmt.Sprintf("%.2f", *bookmark.Rating)
				}
				line = fmt.Sprintf("  [%s] %s: \"%s\" (Line %d)",
					ratingStr, filename, note, bookmark.Position)
			} else {
				// Format: filename.txt: "note text" (Line 123)
				line = fmt.Sprintf("  %s: \"%s\" (Line %d)",
					filename, note, bookmark.Position)
			}

			// Truncate if too long
			if len(line) > bm.width-1 {
				line = line[:bm.width-4] + "..."
			}

			if isSelected {
				b.WriteString(bm.styles.ListCursor.Render(line))
			} else {
				b.WriteString(bm.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	currentLines := strings.Count(b.String(), "\n")
	for currentLines < bm.height-2 {
		b.WriteString("\n")
		currentLines++
	}

	// Help bar
	helpText := "↑↓:Scroll ↵:Open Del:Remove ←/q:Back"
	b.WriteString(bm.styles.RenderHelpBar(helpText, bm.width))

	return b.String()
}

// Helper methods

func (bm *BookmarksModel) getAvailableLines() int {
	// Height minus title(2) and help(1)
	return bm.height - 3
}

// Ensure BookmarksModel implements tea.Model
var _ tea.Model = (*BookmarksModel)(nil)
