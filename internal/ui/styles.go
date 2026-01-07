package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/subbass/litreader/internal/config"
)

// Styles holds all the lipgloss styles for the application
type Styles struct {
	// Base styles
	UI       lipgloss.Style
	Selector lipgloss.Style
	Content  lipgloss.Style

	// Component styles
	Title       lipgloss.Style
	StatusBar   lipgloss.Style
	HelpBar     lipgloss.Style
	ListItem    lipgloss.Style
	ListCursor  lipgloss.Style
	Highlight   lipgloss.Style
}

// NewStyles creates styles from config
func NewStyles(cfg *config.Config) *Styles {
	s := &Styles{}

	// Map color names to lipgloss colors
	uiColor := colorFromString(cfg.UIColor)
	selectorColor := colorFromString(cfg.SelectorColor)
	contentColor := colorFromString(cfg.ContentColor)
	selectorReverseColor := colorFromString(cfg.SelectorReverseColor)

	// Base UI style
	s.UI = lipgloss.NewStyle().Foreground(uiColor)

	// Selector style (for selected items in lists)
	s.Selector = lipgloss.NewStyle().Foreground(selectorColor)
	if cfg.SelectorBold {
		s.Selector = s.Selector.Bold(true)
	}
	if cfg.SelectorReverse {
		s.Selector = s.Selector.Reverse(true).Background(selectorReverseColor)
	}

	// Content style (for file viewer)
	s.Content = lipgloss.NewStyle().Foreground(contentColor)
	if cfg.ContentBold {
		s.Content = s.Content.Bold(true)
	}

	// Title bar style (reverse video, bold, colored)
	s.Title = lipgloss.NewStyle().
		Foreground(uiColor).
		Bold(true).
		Reverse(true)

	// Status bar style (bold, colored - for section headers)
	s.StatusBar = lipgloss.NewStyle().
		Foreground(uiColor).
		Bold(true)

	// Help bar style (reverse video, bold, colored)
	s.HelpBar = lipgloss.NewStyle().
		Foreground(uiColor).
		Bold(true).
		Reverse(true)

	// List item style
	s.ListItem = lipgloss.NewStyle().
		Foreground(contentColor).
		Padding(0, 1)

	// List cursor style (selected item)
	s.ListCursor = s.Selector.Copy().Padding(0, 1)

	// Highlight style (for search matches - reverse video like Python version)
	s.Highlight = lipgloss.NewStyle().
		Reverse(true).
		Bold(true)

	return s
}

// colorFromString maps color names to lipgloss colors
func colorFromString(s string) lipgloss.Color {
	s = strings.ToLower(s)

	colorMap := map[string]string{
		"black":   "0",
		"red":     "1",
		"green":   "2",
		"yellow":  "3",
		"blue":    "4",
		"magenta": "5",
		"cyan":    "6",
		"white":   "7",
	}

	if code, ok := colorMap[s]; ok {
		return lipgloss.Color(code)
	}

	// Default to white if unknown
	return lipgloss.Color("7")
}

// RenderTitle renders a title bar (centered, full width, reverse video)
func (s *Styles) RenderTitle(title string, width int) string {
	// Center the title text
	titleLen := len(title)

	// If title is longer than width, truncate it
	if titleLen > width {
		if width > 3 {
			title = title[:width-3] + "..."
			titleLen = width
		} else if width > 0 {
			title = title[:width]
			titleLen = width
		} else {
			return ""
		}
	}

	padding := (width - titleLen) / 2
	if padding < 0 {
		padding = 0
	}

	rightPadding := width - padding - titleLen
	if rightPadding < 0 {
		rightPadding = 0
	}

	centeredTitle := strings.Repeat(" ", padding) + title + strings.Repeat(" ", rightPadding)
	return s.Title.Render(centeredTitle)
}

// RenderStatusBar renders a status bar
func (s *Styles) RenderStatusBar(left, right string, width int) string {
	leftStr := s.StatusBar.Render(left)
	rightStr := s.StatusBar.Render(right)

	gap := width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if gap < 0 {
		gap = 0
	}

	return leftStr + strings.Repeat(" ", gap) + rightStr
}

// RenderHelpBar renders a help/keybinding bar (centered, full width, reverse video)
func (s *Styles) RenderHelpBar(text string, width int) string {
	// Center the help text
	textLen := len(text)

	// If text is longer than width, truncate it
	if textLen > width {
		if width > 3 {
			text = text[:width-3] + "..."
			textLen = width
		} else if width > 0 {
			text = text[:width]
			textLen = width
		} else {
			return ""
		}
	}

	padding := (width - textLen) / 2
	if padding < 0 {
		padding = 0
	}

	rightPadding := width - padding - textLen
	if rightPadding < 0 {
		rightPadding = 0
	}

	centeredText := strings.Repeat(" ", padding) + text + strings.Repeat(" ", rightPadding)
	return s.HelpBar.Render(centeredText)
}
