package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/state"
)

// AuthorInfo holds computed information about an author
type AuthorInfo struct {
	Name       string
	Folder     string // Full path
	FileCount  int
	TotalSize  int64
	AvgRating  float64
	IsFavorite bool
}

// ExploreAuthorsModel represents the explore authors view
type ExploreAuthorsModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	authors  []AuthorInfo // Computed from AllFiles
	filtered []AuthorInfo // After applying filter
	cursor   int
	topRow   int
	sortMode int // 0=Name, 1=Count, 2=Size, 3=Rating

	// Filtering
	filterInput textinput.Model
	filterMode  bool

	// Scroll burst guard
	lastScrollTime time.Time
}

// NewExploreAuthorsModel creates a new explore authors view
func NewExploreAuthorsModel(s *state.State, styles *Styles) *ExploreAuthorsModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Width = 40

	em := &ExploreAuthorsModel{
		state:       s,
		styles:      styles,
		cursor:      0,
		topRow:      0,
		sortMode:    0, // Default to Name
		filterInput: ti,
		filterMode:  false,
	}

	// Compute authors from AllFiles
	em.computeAuthors()

	return em
}

// computeAuthors extracts unique authors from AllFiles
func (em *ExploreAuthorsModel) computeAuthors() {
	// Map to aggregate author info
	authorMap := make(map[string]*AuthorInfo)

	// Create a set of favorite authors for quick lookup
	favSet := make(map[string]bool)
	for _, fav := range em.state.Config.AuthorFavorites {
		favSet[fav] = true
	}

	for _, filePath := range em.state.AllFiles {
		// Get the parent directory (author folder)
		authorFolder := filepath.Dir(filePath)
		authorName := filepath.Base(authorFolder)

		// Skip if it's the root search directory
		if authorFolder == em.state.Config.SearchDir {
			continue
		}

		// Initialize or update author info
		if _, exists := authorMap[authorFolder]; !exists {
			authorMap[authorFolder] = &AuthorInfo{
				Name:       authorName,
				Folder:     authorFolder,
				IsFavorite: favSet[authorName],
			}
		}

		info := authorMap[authorFolder]
		info.FileCount++

		// Use cached metadata if available
		if meta, ok := em.state.FileMetadata[filePath]; ok {
			info.TotalSize += meta.Size
			if meta.Rating > 0 {
				// Running average calculation
				oldTotal := info.AvgRating * float64(info.FileCount-1)
				info.AvgRating = (oldTotal + meta.Rating) / float64(info.FileCount)
			}
		}
	}

	// Convert map to slice
	em.authors = make([]AuthorInfo, 0, len(authorMap))
	for _, info := range authorMap {
		em.authors = append(em.authors, *info)
	}

	// Sort and apply filter
	em.sortAuthors()
	em.applyFilter()
}

// sortAuthors sorts the authors based on current sort mode
func (em *ExploreAuthorsModel) sortAuthors() {
	switch em.sortMode {
	case 0: // Name (case-insensitive)
		sort.Slice(em.authors, func(i, j int) bool {
			return strings.ToLower(em.authors[i].Name) < strings.ToLower(em.authors[j].Name)
		})
	case 1: // File count descending
		sort.Slice(em.authors, func(i, j int) bool {
			return em.authors[i].FileCount > em.authors[j].FileCount
		})
	case 2: // Size descending
		sort.Slice(em.authors, func(i, j int) bool {
			return em.authors[i].TotalSize > em.authors[j].TotalSize
		})
	case 3: // Rating descending
		sort.Slice(em.authors, func(i, j int) bool {
			return em.authors[i].AvgRating > em.authors[j].AvgRating
		})
	}
	em.applyFilter()
}

// applyFilter filters authors based on current filter text
func (em *ExploreAuthorsModel) applyFilter() {
	filterText := strings.ToLower(strings.TrimSpace(em.filterInput.Value()))

	if filterText == "" {
		em.filtered = em.authors
	} else {
		em.filtered = make([]AuthorInfo, 0)
		for _, author := range em.authors {
			if strings.Contains(strings.ToLower(author.Name), filterText) {
				em.filtered = append(em.filtered, author)
			}
		}
	}

	// Reset cursor if out of bounds
	if em.cursor >= len(em.filtered) {
		em.cursor = 0
		em.topRow = 0
	}
}

// toggleFavorite adds or removes the selected author from favorites
func (em *ExploreAuthorsModel) toggleFavorite() {
	if len(em.filtered) == 0 || em.cursor >= len(em.filtered) {
		return
	}

	author := &em.filtered[em.cursor]
	authorName := author.Name

	if author.IsFavorite {
		// Remove from favorites
		for i, fav := range em.state.Config.AuthorFavorites {
			if fav == authorName {
				em.state.Config.AuthorFavorites = append(
					em.state.Config.AuthorFavorites[:i],
					em.state.Config.AuthorFavorites[i+1:]...,
				)
				break
			}
		}
		author.IsFavorite = false
	} else {
		// Add to favorites
		em.state.Config.AuthorFavorites = append(em.state.Config.AuthorFavorites, authorName)
		author.IsFavorite = true
	}

	// Update the main authors list too
	for i := range em.authors {
		if em.authors[i].Name == authorName {
			em.authors[i].IsFavorite = author.IsFavorite
			break
		}
	}

	em.state.Config.Save()
}

// Init initializes the explore authors view
func (em *ExploreAuthorsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the explore authors view
func (em *ExploreAuthorsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if em.filterMode {
			// Handle filter mode keys
			switch msg.String() {
			case "esc":
				// Exit filter mode and clear filter
				em.filterMode = false
				em.filterInput.Blur()
				em.filterInput.SetValue("")
				em.applyFilter()
				return em, nil

			case "enter":
				// Exit filter mode but keep filter
				em.filterMode = false
				em.filterInput.Blur()
				return em, nil

			case "up":
				// Navigate while still in filter mode
				now := time.Now()
				if !em.lastScrollTime.IsZero() && now.Sub(em.lastScrollTime) < 15*time.Millisecond {
					return em, nil // XInput2 burst accumulation — discard
				}
				em.lastScrollTime = now
				if em.cursor > 0 {
					em.cursor--
					if em.cursor < em.topRow {
						em.topRow--
					}
				}
				return em, nil

			case "down":
				// Navigate while still in filter mode
				now := time.Now()
				if !em.lastScrollTime.IsZero() && now.Sub(em.lastScrollTime) < 15*time.Millisecond {
					return em, nil // XInput2 burst accumulation — discard
				}
				em.lastScrollTime = now
				if em.cursor < len(em.filtered)-1 {
					em.cursor++
					availableLines := em.getAvailableLines()
					if em.cursor >= em.topRow+availableLines {
						em.topRow++
					}
				}
				return em, nil

			default:
				// Update text input
				em.filterInput, cmd = em.filterInput.Update(msg)
				em.applyFilter()
				return em, cmd
			}
		}

		// Normal mode keys
		switch msg.String() {
		// Navigation
		case "up", "k":
			now := time.Now()
			if !em.lastScrollTime.IsZero() && now.Sub(em.lastScrollTime) < 15*time.Millisecond {
				break // XInput2 burst accumulation — discard
			}
			em.lastScrollTime = now
			if em.cursor > 0 {
				em.cursor--
				if em.cursor < em.topRow {
					em.topRow--
				}
			}

		case "down", "j":
			now := time.Now()
			if !em.lastScrollTime.IsZero() && now.Sub(em.lastScrollTime) < 15*time.Millisecond {
				break // XInput2 burst accumulation — discard
			}
			em.lastScrollTime = now
			if em.cursor < len(em.filtered)-1 {
				em.cursor++
				availableLines := em.getAvailableLines()
				if em.cursor >= em.topRow+availableLines {
					em.topRow++
				}
			}

		case "pgdown":
			em.cursor = min(em.cursor+10, len(em.filtered)-1)
			if em.cursor < 0 {
				em.cursor = 0
			}
			availableLines := em.getAvailableLines()
			em.topRow = min(em.topRow+10, max(0, len(em.filtered)-availableLines))

		case "pgup":
			em.cursor = max(em.cursor-10, 0)
			em.topRow = max(em.topRow-10, 0)

		// Enter filter mode
		case "/":
			em.filterMode = true
			em.filterInput.Focus()
			return em, textinput.Blink

		// Toggle sort mode
		case "t":
			em.sortMode = (em.sortMode + 1) % 4
			// Skip rating mode (3) when ShowRatings is false
			if !em.state.Config.ShowRatings && em.sortMode == 3 {
				em.sortMode = 0
			}
			em.sortAuthors()

		// Toggle favorite
		case "f":
			em.toggleFavorite()

		// Open selected author (browse their files)
		case "enter", "right":
			if len(em.filtered) > 0 && em.cursor < len(em.filtered) {
				author := em.filtered[em.cursor]
				return em, func() tea.Msg {
					return openAuthorFilesMsg{
						authorFolder: author.Folder,
					}
				}
			}

		// Return to dashboard (handled by app.go navigation)
		case "q", "left":
			// Don't handle here - let app.go handle navigation
			break
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			now := time.Now()
			if !em.lastScrollTime.IsZero() && now.Sub(em.lastScrollTime) < 15*time.Millisecond {
				break
			}
			em.lastScrollTime = now
			availableLines := em.getAvailableLines()
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if em.cursor > 0 {
					em.cursor--
					if em.cursor < em.topRow {
						em.topRow = em.cursor
					}
				}
			case tea.MouseButtonWheelDown:
				if em.cursor < len(em.filtered)-1 {
					em.cursor++
					if em.cursor >= em.topRow+availableLines {
						em.topRow++
					}
				}
			}
		}
	}

	return em, nil
}

// View renders the explore authors view
func (em *ExploreAuthorsModel) View() string {
	if em.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := fmt.Sprintf("Explore Authors (%d)", len(em.filtered))
	if len(em.authors) != len(em.filtered) {
		title = fmt.Sprintf("Explore Authors (%d/%d)", len(em.filtered), len(em.authors))
	}
	b.WriteString(em.styles.RenderTitle(title, em.width))
	b.WriteString("\n")

	// Filter input or sort indicator
	if em.filterMode {
		b.WriteString("  Filter: ")
		b.WriteString(em.filterInput.View())
		b.WriteString("\n")
	} else {
		// Sort indicator
		allSortModes := []string{"Name", "Count", "Size", "Rating"}
		sortText := allSortModes[em.sortMode]
		filterText := ""
		if em.filterInput.Value() != "" {
			filterText = fmt.Sprintf(" | Filter: \"%s\"", em.filterInput.Value())
		}
		statusBar := fmt.Sprintf("Sort by: %s%s", sortText, filterText)
		b.WriteString(em.styles.StatusBar.Render(statusBar))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(em.filtered) == 0 {
		if len(em.authors) == 0 {
			b.WriteString("  No authors found. Press 'q' to go back.\n")
		} else {
			b.WriteString("  No authors match the filter. Press 'Esc' to clear.\n")
		}
	} else {
		availableLines := em.getAvailableLines()
		displayCount := min(len(em.filtered), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := em.topRow + idx
			if itemIdx >= len(em.filtered) {
				break
			}

			author := em.filtered[itemIdx]
			isSelected := itemIdx == em.cursor

			// Format file count
			countStr := fmt.Sprintf("%3d", author.FileCount)

			// Format size (KB or MB)
			var sizeStr string
			if author.TotalSize >= 1024*1024 {
				sizeStr = fmt.Sprintf("%6.1f MB", float64(author.TotalSize)/(1024*1024))
			} else {
				sizeStr = fmt.Sprintf("%6.1f KB", float64(author.TotalSize)/1024)
			}

			// Favorite indicator
			favIndicator := "  "
			if author.IsFavorite {
				favIndicator = "* "
			}

			var line string
			if em.state.Config.ShowRatings {
				// Format rating
				ratingStr := " N/A"
				if author.AvgRating > 0 {
					ratingStr = fmt.Sprintf("%.2f", author.AvgRating)
				}
				// Format: [count] [size] [rating] [fav] AuthorName
				line = fmt.Sprintf("  %s | %s | %s | %s%s", countStr, sizeStr, ratingStr, favIndicator, author.Name)
			} else {
				// Format: [count] [size] [fav] AuthorName
				line = fmt.Sprintf("  %s | %s | %s%s", countStr, sizeStr, favIndicator, author.Name)
			}

			// Truncate if too long
			if len(line) > em.width-1 {
				line = line[:em.width-4] + "..."
			}

			if isSelected {
				b.WriteString(em.styles.ListCursor.Render(line))
			} else {
				b.WriteString(em.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	currentLines := strings.Count(b.String(), "\n")
	for currentLines < em.height-2 {
		b.WriteString("\n")
		currentLines++
	}

	// Help bar - context-aware
	var helpText string
	if em.filterMode {
		helpText = "Type to filter | ↑↓:Navigate | ↵:Keep filter | Esc:Clear"
	} else {
		helpText = "/:Filter ↑↓:Scroll ↵:Browse t:Sort f:Fav q:Back"
	}
	b.WriteString(em.styles.RenderHelpBar(helpText, em.width))

	return b.String()
}

// Helper methods

func (em *ExploreAuthorsModel) getAvailableLines() int {
	// Height minus title(1), status/filter(1), gap(1), help(1)
	return em.height - 4
}

// Ensure ExploreAuthorsModel implements tea.Model
var _ tea.Model = (*ExploreAuthorsModel)(nil)
