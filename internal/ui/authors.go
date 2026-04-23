package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/subbass/litreader/internal/library"
	"github.com/subbass/litreader/internal/state"
)

const (
	authorInfoMinWidth  = 80
	authorInfoMaxWidth  = 40
	authorInfoListLimit = 5
	authorInfoGap       = 1
	authorInfoMinList   = 20
)

type authorInfo struct {
	authorName    string
	totalStories  int
	totalSize     int64
	avgSize       int64
	longestName   string
	longestSize   int64
	avgRating     float64
	ratingCount   int
	favoriteNames []string
}

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

	info authorInfo
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
	am.refreshAuthorInfo()

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
			if len(am.authors) > 0 {
				if am.cursor > 0 {
					am.cursor--
				} else {
					am.cursor = len(am.authors) - 1
					availableLines := am.getAvailableLines()
					am.topRow = max(0, len(am.authors)-availableLines)
				}
				if am.cursor < am.topRow {
					am.topRow = am.cursor
				}
				am.refreshAuthorInfo()
			}

		case "down", "j":
			if len(am.authors) > 0 {
				if am.cursor < len(am.authors)-1 {
					am.cursor++
				} else {
					am.cursor = 0
					am.topRow = 0
				}
				availableLines := am.getAvailableLines()
				if am.cursor >= am.topRow+availableLines {
					am.topRow++
				}
				am.refreshAuthorInfo()
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
			am.refreshAuthorInfo()

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
				am.refreshAuthorInfo()
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

	body := am.renderBody()
	b.WriteString(body)

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

func (am *AuthorsModel) renderBody() string {
	bodyHeight := am.getAvailableLines()
	if bodyHeight < 1 {
		return ""
	}

	wideEnough := am.width >= authorInfoMinWidth
	listWidth := am.width
	infoWidth := 0
	if wideEnough {
		infoWidth = min(int(float64(authorInfoMaxWidth)*1.5), am.width/2)
		if infoWidth < 25 {
			infoWidth = 25
		}
		required := infoWidth + (authorInfoMinList * 2) + (authorInfoGap * 2)
		if am.width >= required {
			listWidth = (am.width - infoWidth - (authorInfoGap * 2)) / 2
		} else {
			wideEnough = false
		}
	}

	listContent := am.renderAuthorList(listWidth, bodyHeight)
	if !wideEnough {
		return listContent
	}

	infoContent := am.renderAuthorInfo(infoWidth, bodyHeight)
	rightPadWidth := listWidth
	rightPad := lipgloss.NewStyle().Width(rightPadWidth).Height(bodyHeight).Render("")
	gap := strings.Repeat(" ", authorInfoGap)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, listContent, gap, infoContent, gap, rightPad)
	return joined
}

func (am *AuthorsModel) renderAuthorList(width, height int) string {
	var b strings.Builder

	if len(am.authors) == 0 {
		b.WriteString("  No favorite authors yet. Press 'q' to go back.\n")
	} else {
		displayCount := min(len(am.authors), height)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := am.topRow + idx
			if itemIdx >= len(am.authors) {
				break
			}

			author := am.authors[itemIdx]
			isSelected := itemIdx == am.cursor

			line := fmt.Sprintf("  %s", author)

			if len(line) > width-1 {
				line = line[:width-4] + "..."
			}

			if isSelected {
				b.WriteString(am.styles.ListCursor.Render(line))
			} else {
				b.WriteString(am.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	listStyle := lipgloss.NewStyle().Width(width).Height(height)
	return listStyle.Render(b.String())
}

func (am *AuthorsModel) renderAuthorInfo(width, height int) string {
	var b strings.Builder

	if am.info.authorName == "" {
		b.WriteString("  No author selected.\n")
		infoStyle := lipgloss.NewStyle().Width(width).Height(height)
		return infoStyle.Render(b.String())
	}

	b.WriteString(am.styles.StatusBar.Render(" Author Info "))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Author: %s\n", am.info.authorName))
	b.WriteString(fmt.Sprintf("  Stories: %d\n", am.info.totalStories))
	b.WriteString(fmt.Sprintf("  Total Size: %s\n", formatKB(am.info.totalSize)))
	if am.info.totalStories > 0 {
		b.WriteString(fmt.Sprintf("  Avg Size: %s\n", formatKB(am.info.avgSize)))
	} else {
		b.WriteString("  Avg Size: N/A\n")
	}
	if am.info.longestName != "" {
		b.WriteString(fmt.Sprintf("  Longest: %s (%s)\n",
			am.info.longestName, formatKB(am.info.longestSize)))
	} else {
		b.WriteString("  Longest: N/A\n")
	}
	if am.state.Config.ShowRatings {
		if am.info.ratingCount > 0 {
			avg := am.info.avgRating / float64(am.info.ratingCount)
			b.WriteString(fmt.Sprintf("  Avg Rating: %.2f\n", avg))
		} else {
			b.WriteString("  Avg Rating: N/A\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(am.styles.StatusBar.Render(" Favorite Stories "))
	b.WriteString("\n\n")

	if len(am.info.favoriteNames) == 0 {
		b.WriteString("  None\n")
	} else {
		maxList := min(len(am.info.favoriteNames), authorInfoListLimit)
		for i := 0; i < maxList; i++ {
			title := am.info.favoriteNames[i]
			line := fmt.Sprintf("  - %s", title)
			if len(line) > width-1 {
				line = line[:width-4] + "..."
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		if len(am.info.favoriteNames) > maxList {
			remaining := len(am.info.favoriteNames) - maxList
			b.WriteString(fmt.Sprintf("  +%d more\n", remaining))
		}
	}

	boxWidth := width - 2
	if boxWidth < 18 {
		boxWidth = width
	}
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(boxWidth).
		Render(b.String())

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Top,
		panel,
	)
}

func (am *AuthorsModel) refreshAuthorInfo() {
	if len(am.authors) == 0 || am.cursor >= len(am.authors) {
		am.info = authorInfo{}
		return
	}

	authorName := am.authors[am.cursor]
	authorFolder := filepath.Join(am.state.Config.SearchDir, authorName)
	var totalSize int64
	var longestSize int64
	longestName := ""
	var ratingSum float64
	var ratingCount int
	totalStories := 0

	for _, file := range am.state.AllFiles {
		if filepath.Dir(file) != authorFolder {
			continue
		}
		totalStories++
		if info, err := os.Stat(file); err == nil {
			size := info.Size()
			totalSize += size
			if size > longestSize {
				longestSize = size
				longestName = filepath.Base(file)
			}
		}
		if rating, err := library.ExtractRating(file); err == nil && rating > 0 {
			ratingSum += rating
			ratingCount++
		}
	}

	avgSize := int64(0)
	if totalStories > 0 {
		avgSize = totalSize / int64(totalStories)
	}

	favoriteNames := []string{}
	for _, fav := range am.state.Config.Favorites {
		if filepath.Dir(fav.Filename) == authorFolder {
			favoriteNames = append(favoriteNames, filepath.Base(fav.Filename))
		}
	}
	sort.Slice(favoriteNames, func(i, j int) bool {
		return strings.ToLower(favoriteNames[i]) < strings.ToLower(favoriteNames[j])
	})

	am.info = authorInfo{
		authorName:    authorName,
		totalStories:  totalStories,
		totalSize:     totalSize,
		avgSize:       avgSize,
		longestName:   longestName,
		longestSize:   longestSize,
		avgRating:     ratingSum,
		ratingCount:   ratingCount,
		favoriteNames: favoriteNames,
	}
}

func formatKB(size int64) string {
	if size >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f KB", float64(size)/1024)
}

// Ensure AuthorsModel implements tea.Model
var _ tea.Model = (*AuthorsModel)(nil)
