package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/library"
	"github.com/subbass/litreader/internal/models"
	"github.com/subbass/litreader/internal/state"
	"github.com/subbass/litreader/internal/version"
)

// InProgressItem represents a story in progress
type InProgressItem struct {
	Favorite   models.Favorite
	Percentage float64
	TotalLines int
}

// DashboardModel represents the dashboard view
type DashboardModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	// In-progress stories list
	inProgress []InProgressItem
	cursor     int
	topRow     int

	// Statistics
	totalFiles  int
	totalSizeMB float64
	avgRating   float64
	ratedCount  int
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(s *state.State, styles *Styles) *DashboardModel {
	dm := &DashboardModel{
		state:  s,
		styles: styles,
		cursor: 0,
		topRow: 0,
	}

	// Calculate statistics
	dm.calculateStatistics()

	// Calculate in-progress stories
	dm.calculateInProgress()

	return dm
}

// calculateInProgress finds favorites with position > 0 and < 95% complete
func (dm *DashboardModel) calculateInProgress() {
	dm.inProgress = []InProgressItem{}

	for _, fav := range dm.state.Config.Favorites {
		if !fileExists(fav.Filename) {
			continue
		}

		totalLines := countFileLines(fav.Filename)
		if totalLines == 0 {
			continue
		}

		// Consider 95%+ as finished
		percentage := (float64(fav.Position) / float64(totalLines)) * 100
		if fav.Position > 0 && percentage < 95.0 {
			dm.inProgress = append(dm.inProgress, InProgressItem{
				Favorite:   fav,
				Percentage: percentage,
				TotalLines: totalLines,
			})
		}
	}

	// Sort by percentage complete (descending order - highest % first)
	sortInProgressByPercentage(dm.inProgress)

	// Reset cursor if out of bounds
	if dm.cursor >= len(dm.inProgress) {
		dm.cursor = 0
	}
}

// calculateStatistics computes library statistics from cached metadata
func (dm *DashboardModel) calculateStatistics() {
	files := dm.state.AllFiles
	dm.totalFiles = len(files)

	if dm.totalFiles == 0 {
		dm.totalSizeMB = 0
		dm.avgRating = 0
		dm.ratedCount = 0
		return
	}

	// Use cached metadata if available (fast), otherwise fall back to file I/O (slow)
	var totalSize int64
	var ratings []float64

	for _, filepath := range files {
		// Try to use cached metadata first
		if meta, ok := dm.state.FileMetadata[filepath]; ok {
			// Use cached data - no file I/O!
			totalSize += meta.Size
			if meta.Rating > 0 {
				ratings = append(ratings, meta.Rating)
			}
		} else {
			// Fallback: read from disk (only happens if cache wasn't used)
			if info, err := os.Stat(filepath); err == nil {
				totalSize += info.Size()
			}
			if rating, err := library.ExtractRating(filepath); err == nil && rating > 0 {
				ratings = append(ratings, rating)
			}
		}
	}

	dm.totalSizeMB = float64(totalSize) / (1024 * 1024)
	dm.ratedCount = len(ratings)

	// Calculate average rating
	if len(ratings) > 0 {
		sum := 0.0
		for _, r := range ratings {
			sum += r
		}
		dm.avgRating = sum / float64(len(ratings))
	} else {
		dm.avgRating = 0
	}
}

// Init initializes the dashboard (bubbletea interface)
func (dm *DashboardModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the dashboard
func (dm *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Navigation
		case "up", "k":
			if dm.cursor > 0 {
				dm.cursor--
				if dm.cursor < dm.topRow {
					dm.topRow--
				}
			}

		case "down", "j":
			if dm.cursor < len(dm.inProgress)-1 {
				dm.cursor++
				availableLines := dm.getAvailableLines()
				if dm.cursor >= dm.topRow+availableLines {
					dm.topRow++
				}
			}

		case "pgdown":
			dm.cursor = min(dm.cursor+10, len(dm.inProgress)-1)
			availableLines := dm.getAvailableLines()
			dm.topRow = min(dm.topRow+10, max(0, len(dm.inProgress)-availableLines))

		case "pgup":
			dm.cursor = max(dm.cursor-10, 0)
			dm.topRow = max(dm.topRow-10, 0)

		// Open selected in-progress story
		case "enter", "right":
			if len(dm.inProgress) > 0 && dm.cursor < len(dm.inProgress) {
				// TODO: Open file viewer
				// For now, just return
			}

		// Keyboard shortcuts
		case "s":
			// TODO: Navigate to search
			return dm, nil

		case "f":
			// TODO: Navigate to favorites
			return dm, nil

		case "b":
			// TODO: Navigate to bookmarks
			return dm, nil

		case "a":
			// TODO: Navigate to authors
			return dm, nil

		case "e":
			// TODO: Navigate to explore authors
			return dm, nil

		case "c":
			// TODO: Navigate to config
			return dm, nil

		case "l":
			// TODO: Open last read file
			return dm, nil

		case "q":
			// Dashboard is the root - quit the app
			return dm, tea.Quit
		}
	}

	return dm, nil
}

// View renders the dashboard
func (dm *DashboardModel) View() string {
	if dm.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := GetAppName() + " v" + version.AppVersion + " - Dashboard"
	b.WriteString(dm.styles.RenderTitle(title, dm.width))
	b.WriteString("\n\n")

	// LIBRARY STATISTICS
	b.WriteString("  ")
	b.WriteString(dm.styles.StatusBar.Render("LIBRARY STATISTICS"))
	b.WriteString("\n")

	// Display current library folder name
	libraryName := filepath.Base(dm.state.Config.SearchDir)
	if libraryName == "" || libraryName == "." || libraryName == "/" {
		libraryName = dm.state.Config.SearchDir
	}
	b.WriteString(fmt.Sprintf("    Current Library: %s\n", libraryName))

	// Format with comma separators
	b.WriteString(fmt.Sprintf("    Total Stories: %s\n", formatWithCommas(dm.totalFiles)))
	b.WriteString(fmt.Sprintf("    Total Size: %.1f MB\n", dm.totalSizeMB))

	if dm.state.Config.ShowRatings {
		if dm.avgRating > 0 {
			b.WriteString(fmt.Sprintf("    Average Rating: %.2f/5.0\n", dm.avgRating))
		} else {
			b.WriteString("    Average Rating: N/A\n")
		}
		b.WriteString(fmt.Sprintf("    Rated Stories: %s/%s\n", formatWithCommas(dm.ratedCount), formatWithCommas(dm.totalFiles)))
	}
	b.WriteString("\n")

	// YOUR COLLECTIONS
	b.WriteString("  ")
	b.WriteString(dm.styles.StatusBar.Render("YOUR COLLECTIONS"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("    Favorites: %d\n", len(dm.state.Config.Favorites)))
	b.WriteString(fmt.Sprintf("    Bookmarks: %d\n", len(dm.state.Config.Bookmarks)))
	b.WriteString(fmt.Sprintf("    Favorite Authors: %d\n", len(dm.state.Config.AuthorFavorites)))
	b.WriteString("\n")

	// LAST READ
	if dm.state.Config.LastFile != "" {
		b.WriteString("  ")
		b.WriteString(dm.styles.StatusBar.Render("LAST READ"))
		b.WriteString("\n")
		lastFileName := filepath.Base(dm.state.Config.LastFile)
		if len(lastFileName) > dm.width-8 {
			lastFileName = lastFileName[:dm.width-11] + "..."
		}
		b.WriteString(fmt.Sprintf("    %s\n", lastFileName))

		if fileExists(dm.state.Config.LastFile) {
			totalLines := countFileLines(dm.state.Config.LastFile)
			if totalLines > 0 {
				percentage := (float64(dm.state.Config.Position) / float64(totalLines)) * 100
				b.WriteString(fmt.Sprintf("    Progress: %.1f%% (Line %d)\n", percentage, dm.state.Config.Position))
			}
		}
		b.WriteString("\n")
	}

	// STORIES IN PROGRESS
	b.WriteString("  ")
	b.WriteString(dm.styles.StatusBar.Render("STORIES IN PROGRESS"))
	b.WriteString("\n")

	if len(dm.inProgress) == 0 {
		b.WriteString("    No stories in progress. Press 's' to search!\n")
	} else {
		availableLines := dm.getAvailableLines()
		displayCount := min(len(dm.inProgress), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := dm.topRow + idx
			if itemIdx >= len(dm.inProgress) {
				break
			}

			item := dm.inProgress[itemIdx]
			isSelected := itemIdx == dm.cursor

			progressBar := createProgressBar(item.Percentage, 20)
			filename := filepath.Base(item.Favorite.Filename)

			var line string
			if dm.state.Config.ShowRatings {
				// Format: [=======>-----] 45.3% | 4.68 | filename.txt
				ratingStr := "N/A "
				if item.Favorite.Rating != nil {
					ratingStr = fmt.Sprintf("%.2f", *item.Favorite.Rating)
				}
				line = fmt.Sprintf("  [%s] %5.1f%% | %s | %s",
					progressBar, item.Percentage, ratingStr, filename)
			} else {
				// Format: [=======>-----] 45.3% | filename.txt
				line = fmt.Sprintf("  [%s] %5.1f%% | %s",
					progressBar, item.Percentage, filename)
			}

			// Truncate if too long
			if len(line) > dm.width-1 {
				line = line[:dm.width-4] + "..."
			}

			if isSelected {
				b.WriteString(dm.styles.ListCursor.Render(line))
			} else {
				b.WriteString(dm.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pad to fill space
	currentLines := strings.Count(b.String(), "\n")
	for currentLines < dm.height-2 {
		b.WriteString("\n")
		currentLines++
	}

	// Key bindings bar
	helpText := "↑↓:Scroll ↵:Open s:Search f:Fav b:Bmk a:Auth e:Explore c:Config l:Last q:Quit"
	b.WriteString(dm.styles.RenderHelpBar(helpText, dm.width))

	return b.String()
}

// getAvailableLines calculates how many lines are available for the in-progress list
func (dm *DashboardModel) getAvailableLines() int {
	// Total height minus: title(2) + stats(5) + collections(5) + last read(4) + header(2) + help(1)
	usedLines := 19
	if dm.state.Config.LastFile == "" {
		usedLines -= 4
	}
	available := dm.height - usedLines
	if available < 1 {
		available = 1
	}
	return available
}

// Helper functions

func createProgressBar(percentage float64, width int) string {
	filled := int((percentage / 100.0) * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("=", filled) + strings.Repeat("-", width-filled)
	return bar
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func countFileLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	// Try UTF-8 first, fall back to latin1
	content := string(data)
	lines := strings.Split(content, "\n")
	return len(lines)
}

func sortInProgressByPercentage(items []InProgressItem) {
	// Sort by percentage complete (descending order - highest % first)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].Percentage < items[j].Percentage {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func formatWithCommas(n int) string {
	// Format integer with comma separators (e.g., 72627 -> "72,627")
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}
	return result.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Ensure DashboardModel implements tea.Model
var _ tea.Model = (*DashboardModel)(nil)
