package ui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/subbass/litreader/internal/cache"
	"github.com/subbass/litreader/internal/external"
	"github.com/subbass/litreader/internal/library"
	"github.com/subbass/litreader/internal/state"
)

// SearchModel represents the search view
type SearchModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	// Search input
	textInput textinput.Model
	query     string

	// Results
	results []cache.SearchResult
	cursor  int
	topRow  int

	// State
	searching  bool
	sortMode   int  // 0=rating desc, 1=rating asc, 2=size asc, 3=size desc, 4=matches asc, 5=matches desc, 6=name
	fromCache  bool // Whether results came from cache
	cacheError bool // Whether cache load failed
	animFrame  int  // Animation frame counter
}

// NewSearchModel creates a new search view
func NewSearchModel(s *state.State, styles *Styles, initialQuery string) *SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Enter search term..."
	ti.Width = 50
	ti.SetValue(initialQuery)
	ti.Focus() // Auto-focus so user can start typing immediately

	sm := &SearchModel{
		state:     s,
		styles:    styles,
		textInput: ti,
		query:     initialQuery,
		results:    []cache.SearchResult{},
		cursor:     0,
		topRow:     0,
		searching:  false,
		sortMode:   3, // Default to Size Descending (user's preferred sort)
		fromCache:  false,
		cacheError: false,
	}

	return sm
}

// tickMsg is sent on each animation frame
type tickMsg time.Time

// searchCompleteMsg is sent when search completes
type searchCompleteMsg struct {
	results    []cache.SearchResult
	fromCache  bool
	err        error
}

// Init initializes the search view
func (sm *SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// tickCmd returns a command that sends a tick message for animation
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages for the search view
func (sm *SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		// Update animation frame
		if sm.searching {
			sm.animFrame++
			if sm.animFrame >= 1000 {
				sm.animFrame = 0
			}
			return sm, tickCmd()
		}
		return sm, nil

	case searchCompleteMsg:
		sm.searching = false
		if msg.err != nil {
			sm.results = []cache.SearchResult{}
			sm.cacheError = true
		} else {
			sm.results = msg.results
			sm.fromCache = msg.fromCache
			sm.sortResults()
		}
		return sm, nil

	case tea.KeyMsg:
		// If entering search query
		if sm.textInput.Focused() {
			switch msg.String() {
			case "enter":
				// Execute search
				sm.query = sm.textInput.Value()
				if sm.query != "" {
					sm.searching = true
					sm.animFrame = 0
					sm.textInput.Blur()
					return sm, tea.Batch(sm.performSearch(false), tickCmd())
				}
				sm.textInput.Blur()
				return sm, nil

			case "esc", "ctrl+c":
				// Cancel search input
				sm.textInput.Blur()
				return sm, nil
			}

			// Update text input
			sm.textInput, cmd = sm.textInput.Update(msg)
			return sm, cmd
		}

		// When viewing results
		switch msg.String() {
		// Navigation
		case "up", "k":
			if sm.cursor > 0 {
				sm.cursor--
				if sm.cursor < sm.topRow {
					sm.topRow--
				}
			}

		case "down", "j":
			if sm.cursor < len(sm.results)-1 {
				sm.cursor++
				availableLines := sm.getAvailableLines()
				if sm.cursor >= sm.topRow+availableLines {
					sm.topRow++
				}
			}

		// Open selected result
		case "enter", "right":
			if len(sm.results) > 0 && sm.cursor < len(sm.results) {
				result := sm.results[sm.cursor]
				return sm, func() tea.Msg {
					return openFileMsg{
						filename:   result.FilePath,
						position:   0,
						searchText: sm.query,
					}
				}
			}

		// Sort (cycle through 7 modes)
		case "t":
			sm.sortMode = (sm.sortMode + 1) % 7
			sm.sortResults()

		// Force refresh (bypass cache)
		case "r":
			if sm.query != "" {
				sm.searching = true
				sm.animFrame = 0
				return sm, tea.Batch(sm.performSearch(true), tickCmd())
			}

		// Start new search
		case "/":
			sm.textInput.Focus()
			return sm, textinput.Blink

		// Return to dashboard (handled by app.go navigation)
		case "q":
			// Don't handle here - let app.go handle navigation
			break
		}
	}

	return sm, nil
}

// View renders the search view
func (sm *SearchModel) View() string {
	if sm.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar - centered with script name, version, and search dir
	scriptName := GetAppName()
	version := "2.4.9"
	searchDir := sm.state.Config.SearchDir
	topBar := fmt.Sprintf("  %s (version: %s) : %s  ", scriptName, version, searchDir)
	// Center it
	padding := (sm.width - len(topBar)) / 2
	if padding < 0 {
		padding = 0
	}
	rightPad := sm.width - padding - len(topBar)
	if rightPad < 0 {
		rightPad = 0
	}
	centeredTop := strings.Repeat(" ", padding) + topBar + strings.Repeat(" ", rightPad)
	if len(centeredTop) > sm.width {
		centeredTop = centeredTop[:sm.width]
	}
	b.WriteString(sm.styles.Title.Render(centeredTop))
	b.WriteString("\n")

	// Search input or status bar (second header line)
	if sm.textInput.Focused() {
		b.WriteString("  Search: ")
		b.WriteString(sm.textInput.View())
		b.WriteString("\n")
		b.WriteString("  Press Enter to search, Esc to cancel\n\n")
	} else {
		// Status bar showing search query, sort mode, and result position
		sortModes := []string{"Rating Desc", "Rating Asc", "Size Asc", "Size Desc", "Matches Asc", "Matches Desc", "Name"}
		sortText := sortModes[sm.sortMode]
		cacheIndicator := ""
		if sm.query != "" && len(sm.results) > 0 {
			if sm.fromCache {
				cacheIndicator = " " + sm.styles.Highlight.Render(" CACHED ")
			} else {
				cacheIndicator = " [FRESH]"
			}
		}
		resultPos := ""
		if len(sm.results) > 0 {
			resultPos = fmt.Sprintf(" | Result %d/%d", sm.cursor+1, len(sm.results))
		}
		// Build the status bar with styled cache indicator
		leftPart := fmt.Sprintf("Search: %s | Sort by: %s%s", sm.query, sortText, resultPos)
		secondBar := leftPart + cacheIndicator

		// Calculate padding for centering (need to account for ANSI codes in styled text)
		// Get the visible length (without ANSI codes)
		visibleLen := len(leftPart) + (len(cacheIndicator) - len(sm.styles.Highlight.Render(" CACHED "))) + 8 // 8 = " CACHED " length
		if !sm.fromCache || sm.query == "" || len(sm.results) == 0 {
			visibleLen = len(secondBar)
		}

		padding2 := (sm.width - visibleLen) / 2
		if padding2 < 0 {
			padding2 = 0
		}

		rightPadding := sm.width - padding2 - visibleLen
		if rightPadding < 0 {
			rightPadding = 0
		}

		// Apply status bar styling to the non-highlighted parts only
		styledLeftPart := sm.styles.StatusBar.Render(strings.Repeat(" ", padding2) + leftPart)
		styledRightPart := sm.styles.StatusBar.Render(strings.Repeat(" ", rightPadding))

		// Combine with the highlighted cache indicator in the middle
		b.WriteString(styledLeftPart)
		if sm.fromCache && sm.query != "" && len(sm.results) > 0 {
			b.WriteString(sm.styles.Highlight.Render(" CACHED "))
		} else if sm.query != "" && len(sm.results) > 0 {
			b.WriteString(sm.styles.StatusBar.Render(" [FRESH]"))
		}
		b.WriteString(styledRightPart)
		b.WriteString("\n")

		// One-line gap
		b.WriteString("\n")
	}

	// Results
	if sm.searching {
		b.WriteString(sm.renderSearchAnimation())
		b.WriteString("\n")
	} else if len(sm.results) == 0 {
		if sm.query == "" {
			b.WriteString("  Press '/' to enter search term\n")
		} else {
			b.WriteString("  No results found\n")
		}
	} else {
		availableLines := sm.getAvailableLines()
		displayCount := min(len(sm.results), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := sm.topRow + idx
			if itemIdx >= len(sm.results) {
				break
			}

			result := sm.results[itemIdx]
			isSelected := itemIdx == sm.cursor

			// Get file info
			var fileSize float64
			if info, err := os.Stat(result.FilePath); err == nil {
				fileSize = float64(info.Size()) / 1024.0 // KB
			}

			// Get rating
			rating, err := library.ExtractRating(result.FilePath)
			ratingStr := "0"
			if err == nil && rating > 0 {
				ratingStr = fmt.Sprintf("%.2f", rating)
			}

			// Format match count with padding (e.g., "015")
			countDisplay := fmt.Sprintf("%03d", result.MatchCount)

			// Remove base search dir from path
			displayPath := strings.TrimPrefix(result.FilePath, sm.state.Config.SearchDir)
			displayPath = strings.TrimPrefix(displayPath, "/")

			// Format: count | file_size | rating | path/to/file.txt (matching Python version)
			line := fmt.Sprintf("%s | %10.2f KB | %4s | %s", countDisplay, fileSize, ratingStr, displayPath)

			// Truncate if too long
			if len(line) > sm.width-1 {
				line = line[:sm.width-4] + "..."
			}

			if isSelected {
				b.WriteString(sm.styles.ListCursor.Render(line))
			} else {
				b.WriteString(sm.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	currentLines := strings.Count(b.String(), "\n")
	targetLines := sm.height - 2
	for currentLines < targetLines {
		b.WriteString("\n")
		currentLines++
	}

	// Help bar - context-aware based on whether input is focused
	var helpText string
	if sm.textInput.Focused() {
		helpText = "↵:Search Esc:Cancel"
	} else {
		helpText = "/:Search ↑↓:Navigate ↵:Open t:Sort r:Refresh q:Back"
	}
	b.WriteString(sm.styles.RenderHelpBar(helpText, sm.width))

	return b.String()
}

// Helper methods

func (sm *SearchModel) getAvailableLines() int {
	// Height minus title(1), status(1), gap(1), help(1)
	return sm.height - 4
}

// renderSearchAnimation creates a cool demoscene-style animation
func (sm *SearchModel) renderSearchAnimation() string {
	frame := sm.animFrame
	width := sm.width - 4 // Leave some margin
	if width < 40 {
		width = 40
	}

	// Just use the awesome plasma effect
	lines := sm.renderPlasmaEffect(frame, width)
	return strings.Join(lines, "\n")
}

// renderPlasmaEffect creates a plasma/tunnel effect
func (sm *SearchModel) renderPlasmaEffect(frame, width int) []string {
	lines := []string{}
	t := float64(frame) * 0.15

	chars := []rune{'·', '░', '▒', '▓', '█'}

	for y := 0; y < 10; y++ {
		line := strings.Builder{}
		for x := 0; x < width; x++ {
			// Create plasma effect using sine waves
			xf := float64(x) * 0.1
			yf := float64(y) * 0.4

			v1 := math.Sin(xf + t)
			v2 := math.Sin(yf + t*1.3)
			v3 := math.Sin((xf+yf) + t*0.7)
			v4 := math.Sin(math.Sqrt(xf*xf+yf*yf) + t)

			value := (v1 + v2 + v3 + v4) / 4.0
			idx := int((value+1)*2.49) % len(chars)
			line.WriteRune(chars[idx])
		}
		lines = append(lines, "  "+sm.styles.Highlight.Render(line.String()))
	}

	lines = append(lines, "")
	lines = append(lines, sm.styles.StatusBar.Render("  [ SCANNING FILES... ]"))
	return lines
}

func (sm *SearchModel) performSearch(forceRefresh bool) tea.Cmd {
	query := sm.query
	searchDir := sm.state.Config.SearchDir
	allFiles := sm.state.AllFiles

	return func() tea.Msg {
		// Check cache first (unless forcing refresh)
		if !forceRefresh {
			cached, err := cache.LoadSearchCache(query)
			if err == nil && cached != nil {
				return searchCompleteMsg{
					results:   cached.Results,
					fromCache: true,
					err:       nil,
				}
			}
		}

		// Execute ripgrep search
		results, err := external.SearchFiles(query, searchDir, allFiles)
		if err != nil {
			return searchCompleteMsg{
				results:   []cache.SearchResult{},
				fromCache: false,
				err:       err,
			}
		}

		// Save to cache
		cache.SaveSearchCache(query, results)

		return searchCompleteMsg{
			results:   results,
			fromCache: false,
			err:       nil,
		}
	}
}

func (sm *SearchModel) sortResults() {
	items := sm.results

	// Pre-calculate file sizes and ratings once (not on every comparison)
	type cachedInfo struct {
		size   int64
		rating float64
		name   string
	}

	cache := make(map[string]cachedInfo, len(items))
	for _, item := range items {
		info := cachedInfo{
			name: filepath.Base(item.FilePath),
		}

		// Get metadata from pre-loaded cache (MUCH faster than file I/O!)
		if meta, ok := sm.state.FileMetadata[item.FilePath]; ok {
			info.size = meta.Size
			info.rating = meta.Rating
		} else {
			// Fallback to file system if not in cache (rare case for new files)
			if stat, err := os.Stat(item.FilePath); err == nil {
				info.size = stat.Size()
			}
			if rating, err := library.ExtractRating(item.FilePath); err == nil {
				info.rating = rating
			}
		}

		cache[item.FilePath] = info
	}

	// Use Go's efficient sort instead of bubble sort
	sort.Slice(items, func(i, j int) bool {
		iInfo := cache[items[i].FilePath]
		jInfo := cache[items[j].FilePath]

		switch sm.sortMode {
		case 0: // Rating descending
			return iInfo.rating > jInfo.rating
		case 1: // Rating ascending
			return iInfo.rating < jInfo.rating
		case 2: // Size ascending
			return iInfo.size < jInfo.size
		case 3: // Size descending
			return iInfo.size > jInfo.size
		case 4: // Matches ascending
			return items[i].MatchCount < items[j].MatchCount
		case 5: // Matches descending
			return items[i].MatchCount > items[j].MatchCount
		case 6: // Name
			return iInfo.name < jInfo.name
		default:
			return false
		}
	})

	sm.results = items
}

// Ensure SearchModel implements tea.Model
var _ tea.Model = (*SearchModel)(nil)
