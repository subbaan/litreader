package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/subbass/litreader/internal/external"
	"github.com/subbass/litreader/internal/library"
	"github.com/subbass/litreader/internal/models"
	"github.com/subbass/litreader/internal/state"
)

// ViewerModel represents the file viewer
type ViewerModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	// File content
	filename          string
	lines             []string
	totalLines        int
	originalLineCount int // Line count of original file (before pandoc rendering)

	// Viewing state
	topRow int // Top line currently displayed

	// Search
	searchText    string
	matches       []int // Line numbers containing search text
	currentMatch  int   // Current match index
	searchInput   textinput.Model
	EditingSearch bool // Exported for app navigation check

	// Bookmark note input
	bookmarkInput   textinput.Model
	EditingBookmark bool // Exported for app navigation check

	// Bookmark viewer
	ViewingBookmarks bool // Exported for app navigation check
	fileBookmarks    []models.Bookmark
	bookmarkCursor   int

	// Error
	err error
}

// NewViewerModel creates a new file viewer
func NewViewerModel(s *state.State, styles *Styles, filename string, position int, searchText string) *ViewerModel {
	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Enter search term..."
	ti.Width = 50

	// Initialize bookmark note input
	bi := textinput.New()
	bi.Placeholder = "Enter bookmark note..."
	bi.Width = 50

	vm := &ViewerModel{
		state:            s,
		styles:           styles,
		filename:         filename,
		topRow:           position,
		searchText:       searchText,
		searchInput:      ti,
		EditingSearch:    false,
		bookmarkInput:    bi,
		EditingBookmark:  false,
		ViewingBookmarks: false,
		fileBookmarks:    []models.Bookmark{},
		bookmarkCursor:   0,
		matches:          []int{},
		currentMatch:     0,
	}

	// Load file content
	vm.loadFile()

	// If search text provided, find matches (but don't jump to first match)
	if vm.searchText != "" {
		vm.findMatches()
		// Update currentMatch to reflect the saved position
		vm.updateCurrentMatchFromPosition()
	}

	return vm
}

// loadFile loads and renders the file content
func (vm *ViewerModel) loadFile() {
	data, err := os.ReadFile(vm.filename)
	if err != nil {
		vm.err = err
		return
	}

	// Try to convert to valid UTF-8 string with encoding detection
	content := vm.decodeContent(data)

	// Store original line count before pandoc rendering
	vm.originalLineCount = strings.Count(content, "\n") + 1

	// Render through pandoc if available (with panic recovery)
	func() {
		defer func() {
			if r := recover(); r != nil {
				// If pandoc panics, just use the original content
				vm.err = fmt.Errorf("pandoc rendering failed: %v", r)
			}
		}()

		rendered, err := external.RenderMarkdown(content)
		if err == nil {
			content = rendered
		}
	}()

	vm.lines = strings.Split(content, "\n")
	vm.totalLines = len(vm.lines)

	// Ensure top row is valid
	if vm.topRow >= vm.totalLines {
		vm.topRow = vm.totalLines - 1
	}
	if vm.topRow < 0 {
		vm.topRow = 0
	}
}

// decodeContent tries to decode file content with multiple encoding fallbacks
func (vm *ViewerModel) decodeContent(data []byte) string {
	// First, try UTF-8 (fast path)
	if utf8.Valid(data) {
		return string(data)
	}

	// If not valid UTF-8, try to clean it up by replacing invalid sequences
	// This handles files that are mostly UTF-8 but have some invalid characters
	content := strings.ToValidUTF8(string(data), "�")

	return content
}

// sanitizeLineForDisplay removes or replaces characters that might cause terminal issues
func (vm *ViewerModel) sanitizeLineForDisplay(line string) string {
	// Remove CRLF, keep just LF (already handled by Split, but be safe)
	line = strings.ReplaceAll(line, "\r", "")

	// Ensure the line is valid UTF-8
	if !utf8.ValidString(line) {
		line = strings.ToValidUTF8(line, "�")
	}

	return line
}

// truncateToWidth safely truncates a string to a maximum display width
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Truncate by runes, checking display width as we go
	var result strings.Builder
	currentWidth := 0

	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > maxWidth {
			break
		}
		result.WriteRune(r)
		currentWidth += runeWidth
	}

	return result.String()
}

// Init initializes the viewer
func (vm *ViewerModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the viewer
func (vm *ViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if vm.err != nil {
		// On error, allow navigation back
		// q key will be handled by app.go navigation
		return vm, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If viewing bookmarks
		if vm.ViewingBookmarks {
			switch msg.String() {
			case "up", "k":
				if vm.bookmarkCursor > 0 {
					vm.bookmarkCursor--
				}
			case "down", "j":
				if vm.bookmarkCursor < len(vm.fileBookmarks)-1 {
					vm.bookmarkCursor++
				}
			case "enter":
				// Jump to selected bookmark
				if len(vm.fileBookmarks) > 0 && vm.bookmarkCursor < len(vm.fileBookmarks) {
					vm.topRow = vm.fileBookmarks[vm.bookmarkCursor].Position
					vm.ViewingBookmarks = false
				}
			case "esc", "q", "v", "left":
				// Close bookmark view
				vm.ViewingBookmarks = false
				return vm, nil
			case "delete", "backspace":
				// Delete selected bookmark
				if len(vm.fileBookmarks) > 0 && vm.bookmarkCursor < len(vm.fileBookmarks) {
					deletedBookmark := vm.fileBookmarks[vm.bookmarkCursor]
					// Remove from config
					for i, bm := range vm.state.Config.Bookmarks {
						if bm.Filename == deletedBookmark.Filename &&
							bm.Position == deletedBookmark.Position &&
							bm.DateAdded == deletedBookmark.DateAdded {
							vm.state.Config.Bookmarks = append(vm.state.Config.Bookmarks[:i],
								vm.state.Config.Bookmarks[i+1:]...)
							vm.state.Config.Save()
							break
						}
					}
					// Remove from local list
					vm.fileBookmarks = append(vm.fileBookmarks[:vm.bookmarkCursor],
						vm.fileBookmarks[vm.bookmarkCursor+1:]...)
					if vm.bookmarkCursor >= len(vm.fileBookmarks) && len(vm.fileBookmarks) > 0 {
						vm.bookmarkCursor = len(vm.fileBookmarks) - 1
					}
					// Close if no more bookmarks
					if len(vm.fileBookmarks) == 0 {
						vm.ViewingBookmarks = false
					}
				}
			}
			return vm, nil
		}

		// If editing bookmark note
		if vm.EditingBookmark {
			switch msg.String() {
			case "enter":
				// Save bookmark with note
				note := vm.bookmarkInput.Value()
				newBookmark := models.Bookmark{
					Filename:  vm.filename,
					Position:  vm.topRow,
					Note:      note,
					DateAdded: time.Now(),
					Rating:    nil,
				}

				// Try to extract rating
				rating, err := library.ExtractRating(vm.filename)
				if err == nil && rating > 0 {
					newBookmark.Rating = &rating
				}

				vm.state.Config.Bookmarks = append(vm.state.Config.Bookmarks, newBookmark)
				vm.state.Config.Save()

				vm.EditingBookmark = false
				vm.bookmarkInput.Blur()
				vm.bookmarkInput.SetValue("") // Clear for next time
				return vm, nil

			case "esc", "ctrl+c":
				// Cancel bookmark input
				vm.EditingBookmark = false
				vm.bookmarkInput.Blur()
				vm.bookmarkInput.SetValue("")
				return vm, nil
			}

			// Update text input
			var cmd tea.Cmd
			vm.bookmarkInput, cmd = vm.bookmarkInput.Update(msg)
			return vm, cmd
		}

		// If editing search term
		if vm.EditingSearch {
			switch msg.String() {
			case "enter":
				// Apply new search term
				vm.searchText = vm.searchInput.Value()
				if vm.searchText != "" {
					vm.findMatches()
					if len(vm.matches) > 0 {
						vm.currentMatch = 0
						vm.topRow = max(vm.matches[0]-2, 0)
					}
				} else {
					vm.matches = []int{}
				}
				vm.EditingSearch = false
				vm.searchInput.Blur()
				return vm, nil

			case "esc", "ctrl+c":
				// Cancel search input
				vm.EditingSearch = false
				vm.searchInput.Blur()
				return vm, nil
			}

			// Update text input
			var cmd tea.Cmd
			vm.searchInput, cmd = vm.searchInput.Update(msg)
			return vm, cmd
		}

		switch msg.String() {
		// Navigation - scroll viewport directly
		case "down", "j":
			if vm.topRow < vm.totalLines-1 {
				vm.topRow++
			}

		case "up", "k":
			if vm.topRow > 0 {
				vm.topRow--
			}

		case "pgdown", " ":
			// Leave 2 lines visible from previous page for context
			scrollAmount := max(vm.getAvailableLines()-2, 1)
			vm.topRow = min(vm.topRow+scrollAmount, vm.totalLines-1)

		case "pgup":
			// Leave 2 lines visible from previous page for context
			scrollAmount := max(vm.getAvailableLines()-2, 1)
			vm.topRow = max(vm.topRow-scrollAmount, 0)

		case "home", "g":
			vm.topRow = 0

		case "end", "G":
			availableLines := vm.getAvailableLines()
			vm.topRow = max(0, vm.totalLines-availableLines)

		// Percentage jumps (0-9 for 0%-90%)
		case "0":
			vm.jumpToPercentage(0)
		case "1":
			vm.jumpToPercentage(10)
		case "2":
			vm.jumpToPercentage(20)
		case "3":
			vm.jumpToPercentage(30)
		case "4":
			vm.jumpToPercentage(40)
		case "5":
			vm.jumpToPercentage(50)
		case "6":
			vm.jumpToPercentage(60)
		case "7":
			vm.jumpToPercentage(70)
		case "8":
			vm.jumpToPercentage(80)
		case "9":
			vm.jumpToPercentage(90)

		// Add to favorites
		case "f":
			vm.addToFavorites()

		// Add author to author favorites
		case "a":
			vm.addAuthorToFavorites()

		// Add bookmark
		case "b":
			vm.bookmarkInput.SetValue("")
			vm.bookmarkInput.Focus()
			vm.EditingBookmark = true
			return vm, textinput.Blink

		// Export file
		case "c":
			vm.exportFile()

		// Search within file
		case "s":
			vm.searchInput.SetValue(vm.searchText)
			vm.searchInput.Focus()
			vm.EditingSearch = true
			return vm, textinput.Blink

		// Clear search
		case "x":
			vm.searchText = ""
			vm.matches = []int{}
			vm.currentMatch = 0

		// Cycle through search matches (if any exist)
		case "right":
			if len(vm.matches) > 0 {
				// Find the next match after current position
				nextMatch := -1
				for i, matchLine := range vm.matches {
					if matchLine > vm.topRow+2 { // +2 for context
						nextMatch = i
						break
					}
				}
				if nextMatch != -1 {
					vm.currentMatch = nextMatch
					vm.topRow = max(vm.matches[vm.currentMatch]-2, 0)
				}
			}

		case "left":
			if len(vm.matches) > 0 {
				// Find the previous match before current position
				prevMatch := -1
				for i := len(vm.matches) - 1; i >= 0; i-- {
					if vm.matches[i] < vm.topRow {
						prevMatch = i
						break
					}
				}
				if prevMatch != -1 {
					vm.currentMatch = prevMatch
					vm.topRow = max(vm.matches[vm.currentMatch]-2, 0)
				}
			} else {
				// No search matches, use left arrow to go back
				vm.savePosition()
				// Let app.go handle navigation
			}

		// Open author files
		case "o":
			authorFolder := filepath.Dir(vm.filename)
			return vm, func() tea.Msg {
				return openAuthorFilesMsg{
					authorFolder: authorFolder,
				}
			}

		// View bookmarks for current file
		case "v":
			vm.viewFileBookmarks()

		// Edit in external editor
		case "e":
			// Open built-in editor at estimated original line
			lineNum := 1
			if vm.totalLines > 0 && vm.originalLineCount > 0 {
				percentage := float64(vm.topRow) / float64(vm.totalLines)
				lineNum = int(percentage*float64(vm.originalLineCount)) + 1
				if lineNum < 1 {
					lineNum = 1
				}
				if lineNum > vm.originalLineCount {
					lineNum = vm.originalLineCount
				}
			}
			return vm, func() tea.Msg {
				return openEditorMsg{filename: vm.filename, startLine: lineNum}
			}

			// TODO: Implement other features
			// case "/": // Search within file
		}
	}

	return vm, nil
}

// View renders the viewer
func (vm *ViewerModel) View() string {
	if vm.err != nil {
		return fmt.Sprintf("Error loading file: %v\nPress 'q' to quit", vm.err)
	}

	if vm.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Check if file is favorited
	isFileFaved := false
	for _, fav := range vm.state.Config.Favorites {
		if fav.Filename == vm.filename {
			isFileFaved = true
			break
		}
	}

	// Check if author is favorited (check AuthorFavorites list, not Favorites)
	authorDir := filepath.Dir(vm.filename)
	authorName := filepath.Base(authorDir)
	isAuthorFaved := false
	for _, author := range vm.state.Config.AuthorFavorites {
		// Support both old format (full path) and new format (just name)
		if author == authorName || author == authorDir {
			isAuthorFaved = true
			break
		}
	}

	// Check if file has bookmarks
	hasBookmarks := false
	bookmarkCount := 0
	for _, bm := range vm.state.Config.Bookmarks {
		if bm.Filename == vm.filename {
			hasBookmarks = true
			bookmarkCount++
		}
	}

	// Top bar - centered with script name and filename (with heart if faved)
	filename := filepath.Base(vm.filename)
	fileHeart := ""
	if isFileFaved {
		fileHeart = "\uf004 "
	}
	topBar := fmt.Sprintf("  %s : %s%s  ", GetAppName(), fileHeart, filename)
	// Center it using proper width calculation
	topBarWidth := lipgloss.Width(topBar)
	padding := (vm.width - topBarWidth) / 2
	if padding < 0 {
		padding = 0
	}
	rightPadding := vm.width - padding - topBarWidth
	if rightPadding < 0 {
		rightPadding = 0
	}
	centeredTop := strings.Repeat(" ", padding) + topBar + strings.Repeat(" ", rightPadding)
	b.WriteString(vm.styles.Title.Render(centeredTop))
	b.WriteString("\n")

	// Second bar - Author, Size, Percentage (centered, with heart if author faved)
	percentage := 0.0
	if vm.totalLines > 0 {
		percentage = (float64(vm.topRow) / float64(vm.totalLines)) * 100
	}

	// Get file size
	var fileSize float64
	if info, err := os.Stat(vm.filename); err == nil {
		fileSize = float64(info.Size()) / 1024.0 // KB
	}

	// Author heart (authorName already calculated above)
	authorHeart := ""
	if isAuthorFaved {
		authorHeart = "\uf004 "
	}

	// Bookmark indicator
	bookmarkIndicator := ""
	if hasBookmarks {
		bookmarkIndicator = fmt.Sprintf(" | \uf02e %d", bookmarkCount)
	}

	secondBar := fmt.Sprintf("Author: %s%s | Size: %.2f KB | %.2f%% read%s", authorHeart, authorName, fileSize, percentage, bookmarkIndicator)
	// Center it using proper width calculation
	secondBarWidth := lipgloss.Width(secondBar)
	padding2 := (vm.width - secondBarWidth) / 2
	if padding2 < 0 {
		padding2 = 0
	}
	rightPadding2 := vm.width - padding2 - secondBarWidth
	if rightPadding2 < 0 {
		rightPadding2 = 0
	}
	centeredSecond := strings.Repeat(" ", padding2) + secondBar + strings.Repeat(" ", rightPadding2)
	b.WriteString(vm.styles.Title.Render(centeredSecond))
	b.WriteString("\n")

	// Content (with error recovery for problematic lines)
	availableLines := vm.getAvailableLines()
	endRow := min(vm.topRow+availableLines, vm.totalLines)

	for i := vm.topRow; i < endRow; i++ {
		if i >= len(vm.lines) {
			break
		}

		line := vm.lines[i]

		// Clean the line for safe terminal rendering
		line = vm.sanitizeLineForDisplay(line)

		// Highlight search text if present
		if vm.searchText != "" {
			line = vm.highlightSearch(line)
		}

		// Truncate if too long (using display width)
		if lipgloss.Width(line) > vm.width {
			if vm.width > 0 {
				line = truncateToWidth(line, vm.width)
			}
		}

		// Render with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					// If rendering panics, write plain text
					b.WriteString(line)
					b.WriteString("\n")
				}
			}()
			b.WriteString(vm.styles.Content.Render(line))
			b.WriteString("\n")
		}()
	}

	// Pad remaining lines
	for i := endRow - vm.topRow; i < availableLines; i++ {
		b.WriteString("\n")
	}

	// Footer bar - two lines
	// Line 1: Navigation and favorites
	line1Text := "←/q:Back ↑↓/Space:Scroll ⌫:PgUp 0-9:%Jump f:Fav a:FavAuthor b:Bookmark"
	line1Width := lipgloss.Width(line1Text)
	if line1Width < vm.width {
		line1Text = line1Text + strings.Repeat(" ", vm.width-line1Width)
	} else if line1Width > vm.width {
		line1Text = line1Text[:vm.width]
	}

	// Line 2: Actions and optional search info
	line2Text := "e:Edit c:Export o:AuthorFiles v:ViewBookmarks"

	// Add search input if editing (bookmark input now shows in overlay)
	if vm.EditingSearch {
		line2Text += " [" + vm.searchInput.View() + "]"
	} else if vm.searchText != "" && len(vm.matches) > 0 {
		line2Text += fmt.Sprintf(" [s:Search %s  (%d/%d) | ←→:Nav x:Clear]", vm.searchText, vm.currentMatch+1, len(vm.matches))
	} else if vm.searchText != "" {
		line2Text += fmt.Sprintf(" [s:Search %s  (0/0) | x:Clear]", vm.searchText)
	} else {
		line2Text += " [s:Search]"
	}

	// Pad line 2 to full width using proper width calculation
	line2Width := lipgloss.Width(line2Text)
	if line2Width < vm.width {
		line2Text = line2Text + strings.Repeat(" ", vm.width-line2Width)
	} else if line2Width > vm.width {
		line2Text = line2Text[:vm.width]
	}

	// Write both lines
	b.WriteString(vm.styles.HelpBar.Render(line1Text))
	b.WriteString("\n")
	b.WriteString(vm.styles.HelpBar.Render(line2Text))

	result := b.String()

	// Overlay bookmark input if editing
	if vm.EditingBookmark {
		result = vm.renderBookmarkInputOverlay(result)
	}

	// Overlay bookmark viewer if active
	if vm.ViewingBookmarks {
		result = vm.renderBookmarkOverlay(result)
	}

	return result
}

// Helper methods

func (vm *ViewerModel) renderBookmarkInputOverlay(baseView string) string {
	// Create a centered popup box for bookmark input
	boxWidth := min(vm.width-8, 60)

	var content strings.Builder

	// Title
	content.WriteString("┌")
	content.WriteString(strings.Repeat("─", boxWidth-2))
	content.WriteString("┐\n")

	titleText := " Add Bookmark "
	padding := (boxWidth - len(titleText)) / 2
	if padding < 0 {
		padding = 1
	}
	content.WriteString("│")
	content.WriteString(strings.Repeat(" ", padding))
	content.WriteString(titleText)
	content.WriteString(strings.Repeat(" ", boxWidth-padding-len(titleText)-2))
	content.WriteString("│\n")

	// Separator
	content.WriteString("├")
	content.WriteString(strings.Repeat("─", boxWidth-2))
	content.WriteString("┤\n")

	// Empty line
	content.WriteString("│")
	content.WriteString(strings.Repeat(" ", boxWidth-2))
	content.WriteString("│\n")

	// Input label and field
	label := "  Note: "
	inputView := vm.bookmarkInput.View()
	inputWidth := boxWidth - 4 - len(label)
	if len(inputView) > inputWidth {
		inputView = inputView[:inputWidth]
	}

	content.WriteString("│")
	content.WriteString(label)
	content.WriteString(inputView)
	content.WriteString(strings.Repeat(" ", boxWidth-2-len(label)-len(inputView)))
	content.WriteString("│\n")

	// Empty line
	content.WriteString("│")
	content.WriteString(strings.Repeat(" ", boxWidth-2))
	content.WriteString("│\n")

	// Help text
	helpText := " ↵:Save  Esc:Cancel "
	helpPadding := (boxWidth - len(helpText)) / 2
	if helpPadding < 0 {
		helpPadding = 1
	}
	content.WriteString("│")
	content.WriteString(strings.Repeat(" ", helpPadding))
	content.WriteString(helpText)
	content.WriteString(strings.Repeat(" ", boxWidth-helpPadding-len(helpText)-2))
	content.WriteString("│\n")

	// Bottom border
	content.WriteString("└")
	content.WriteString(strings.Repeat("─", boxWidth-2))
	content.WriteString("┘")

	// Style the box
	styledBox := vm.styles.Title.
		Width(boxWidth).
		Render(content.String())

	// Place overlay in center using lipgloss
	return lipgloss.Place(vm.width, vm.height,
		lipgloss.Center, lipgloss.Center,
		styledBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
}

func (vm *ViewerModel) renderBookmarkOverlay(baseView string) string {
	// Create a centered overlay box for bookmarks
	overlayWidth := min(vm.width-4, 80)
	overlayHeight := min(vm.height-4, len(vm.fileBookmarks)+4)

	var overlay strings.Builder

	// Title
	title := fmt.Sprintf(" Bookmarks for this file (%d) ", len(vm.fileBookmarks))
	padding := (overlayWidth - len(title)) / 2
	if padding < 0 {
		padding = 0
	}
	overlay.WriteString(strings.Repeat("─", padding))
	overlay.WriteString(title)
	overlay.WriteString(strings.Repeat("─", overlayWidth-padding-len(title)))
	overlay.WriteString("\n")

	// Bookmarks list
	for i, bm := range vm.fileBookmarks {
		isSelected := i == vm.bookmarkCursor

		note := bm.Note
		if note == "" {
			note = "(no note)"
		}
		if len(note) > overlayWidth-20 {
			note = note[:overlayWidth-23] + "..."
		}

		line := fmt.Sprintf("Line %d: %s", bm.Position, note)
		if len(line) > overlayWidth-4 {
			line = line[:overlayWidth-7] + "..."
		}

		if isSelected {
			overlay.WriteString(vm.styles.ListCursor.Render("  " + line))
		} else {
			overlay.WriteString("  " + line)
		}
		overlay.WriteString("\n")
	}

	// Help text
	overlay.WriteString(strings.Repeat("─", overlayWidth))
	overlay.WriteString("\n")
	helpText := "↑↓:Select ↵:Jump Del:Delete Esc/v:Close"
	overlay.WriteString(helpText)

	// Center the overlay on the screen
	overlayBox := vm.styles.Title.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(overlay.String())

	// Place overlay in center using lipgloss
	return lipgloss.Place(vm.width, vm.height,
		lipgloss.Center, lipgloss.Center,
		overlayBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
}

func (vm *ViewerModel) getAvailableLines() int {
	// Height minus title(1), second bar(1), footer(2)
	return vm.height - 4
}

func (vm *ViewerModel) jumpToPercentage(pct int) {
	targetLine := (vm.totalLines * pct) / 100
	vm.topRow = targetLine
}

func (vm *ViewerModel) highlightSearch(line string) string {
	if vm.searchText == "" {
		return line
	}

	// Simple case-insensitive highlighting
	lower := strings.ToLower(line)
	lowerSearch := strings.ToLower(vm.searchText)

	idx := strings.Index(lower, lowerSearch)
	if idx == -1 {
		return line
	}

	before := line[:idx]
	match := line[idx : idx+len(vm.searchText)]
	after := line[idx+len(vm.searchText):]

	return before + vm.styles.Highlight.Render(match) + after
}

func (vm *ViewerModel) findMatches() {
	vm.matches = []int{}
	if vm.searchText == "" {
		return
	}

	lowerSearch := strings.ToLower(vm.searchText)
	for i, line := range vm.lines {
		if strings.Contains(strings.ToLower(line), lowerSearch) {
			vm.matches = append(vm.matches, i)
		}
	}
	vm.currentMatch = 0
}

func (vm *ViewerModel) updateCurrentMatchFromPosition() {
	if len(vm.matches) == 0 {
		vm.currentMatch = 0
		return
	}

	// Find the match closest to current position
	for i, matchLine := range vm.matches {
		if matchLine >= vm.topRow {
			vm.currentMatch = i
			return
		}
	}
	// If we're past all matches, set to last match
	vm.currentMatch = len(vm.matches) - 1
}

func (vm *ViewerModel) savePosition() {
	// Update config with current position
	vm.state.Config.LastFile = vm.filename
	vm.state.Config.Position = vm.topRow
	vm.state.Config.SearchText = vm.searchText

	// Also update the favorite's position if this file is in favorites
	for i := range vm.state.Config.Favorites {
		if vm.state.Config.Favorites[i].Filename == vm.filename {
			vm.state.Config.Favorites[i].Position = vm.topRow
			vm.state.Config.Favorites[i].SearchText = vm.searchText
			break
		}
	}

	// Save config
	vm.state.Config.Save()
}

func (vm *ViewerModel) addToFavorites() {
	// Check if already in favorites
	for i := range vm.state.Config.Favorites {
		if vm.state.Config.Favorites[i].Filename == vm.filename {
			// Already exists, update position
			vm.state.Config.Favorites[i].Position = vm.topRow
			vm.state.Config.Favorites[i].SearchText = vm.searchText
			vm.state.Config.Save()
			return
		}
	}

	// Add new favorite
	newFav := models.Favorite{
		Filename:   vm.filename,
		Position:   vm.topRow,
		SearchText: vm.searchText,
		DateAdded:  time.Now(),
		Rating:     nil, // Rating will be extracted if available
	}

	// Try to extract rating
	rating, err := library.ExtractRating(vm.filename)
	if err == nil && rating > 0 {
		newFav.Rating = &rating
	}

	vm.state.Config.Favorites = append(vm.state.Config.Favorites, newFav)
	vm.state.FavoritesVersion++
	vm.state.Config.Save()
}

func (vm *ViewerModel) viewFileBookmarks() {
	// Find all bookmarks for current file
	vm.fileBookmarks = []models.Bookmark{}
	for _, bm := range vm.state.Config.Bookmarks {
		if bm.Filename == vm.filename {
			vm.fileBookmarks = append(vm.fileBookmarks, bm)
		}
	}

	// Only show if there are bookmarks
	if len(vm.fileBookmarks) > 0 {
		vm.ViewingBookmarks = true
		vm.bookmarkCursor = 0
	}
}

func (vm *ViewerModel) addAuthorToFavorites() {
	// Get the author name (just the folder name, relative to search directory)
	authorDir := filepath.Dir(vm.filename)
	authorName := filepath.Base(authorDir)

	// Check if author is already in author favorites
	for _, author := range vm.state.Config.AuthorFavorites {
		if author == authorName {
			// Already in favorites, don't add again
			return
		}
	}

	// Add author to favorites (just the name, not full path)
	vm.state.Config.AuthorFavorites = append(vm.state.Config.AuthorFavorites, authorName)
	vm.state.Config.Save()
}

func (vm *ViewerModel) exportFile() {
	// Get export directory from config
	exportDir := vm.state.Config.ExportDir
	if exportDir == "" {
		return
	}

	// Ensure export directory exists
	os.MkdirAll(exportDir, 0755)

	// Extract author and title from path
	author := filepath.Base(filepath.Dir(vm.filename))
	filename := filepath.Base(vm.filename)

	// Extract title (everything before -litero_ or .txt)
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	if idx := strings.Index(title, "-litero_"); idx != -1 {
		title = title[:idx]
	}

	// Create export filename: author-title.txt
	exportFilename := fmt.Sprintf("%s-%s.txt", author, title)
	exportPath := filepath.Join(exportDir, exportFilename)

	// Copy file
	input, err := os.ReadFile(vm.filename)
	if err != nil {
		return
	}
	os.WriteFile(exportPath, input, 0644)
}

// Ensure ViewerModel implements tea.Model
var _ tea.Model = (*ViewerModel)(nil)
