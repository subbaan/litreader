package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/subbass/litreader/internal/cache"
	"github.com/subbass/litreader/internal/state"
)

// ConfigItem represents a single configurable setting
type ConfigItem struct {
	Key         string
	DisplayName string
	Value       string
	Type        string // "string", "bool", "int", "color", "action"
}

// Available color choices
var availableColors = []string{
	"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
}

// ConfigEditorModel represents the configuration editor view
type ConfigEditorModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	// Config items
	items  []ConfigItem
	cursor int
	topRow int

	// Editing state
	editing   bool
	textInput textinput.Model

	// Status message
	statusMsg string
}

// NewConfigEditorModel creates a new config editor view
func NewConfigEditorModel(s *state.State, styles *Styles) *ConfigEditorModel {
	ti := textinput.New()
	ti.Width = 50

	cem := &ConfigEditorModel{
		state:     s,
		styles:    styles,
		textInput: ti,
		cursor:    0,
		topRow:    0,
		editing:   false,
	}

	// Build config items list
	cem.buildConfigItems()

	return cem
}

// buildConfigItems creates the list of editable config items
func (cem *ConfigEditorModel) buildConfigItems() {
	c := cem.state.Config

	cem.items = []ConfigItem{
		{Key: "SearchDir", DisplayName: "Search Directory", Value: c.SearchDir, Type: "string"},
		{Key: "ExportDir", DisplayName: "Export Directory", Value: c.ExportDir, Type: "string"},
		{Key: "Editor", DisplayName: "Editor", Value: c.Editor, Type: "string"},
		{Key: "UIColor", DisplayName: "UI Color", Value: c.UIColor, Type: "color"},
		{Key: "SelectorColor", DisplayName: "Selector Color", Value: c.SelectorColor, Type: "color"},
		{Key: "SelectorReverse", DisplayName: "Selector Reverse", Value: fmt.Sprintf("%t", c.SelectorReverse), Type: "bool"},
		{Key: "SelectorBold", DisplayName: "Selector Bold", Value: fmt.Sprintf("%t", c.SelectorBold), Type: "bool"},
		{Key: "SelectorReverseColor", DisplayName: "Selector Reverse Color", Value: c.SelectorReverseColor, Type: "color"},
		{Key: "ContentColor", DisplayName: "Content Color", Value: c.ContentColor, Type: "color"},
		{Key: "ContentBold", DisplayName: "Content Bold", Value: fmt.Sprintf("%t", c.ContentBold), Type: "bool"},
		{Key: "CacheExpiryDays", DisplayName: "Cache Expiry (days)", Value: fmt.Sprintf("%d", c.CacheExpiryDays), Type: "int"},
		{Key: "ClearSearchCache", DisplayName: "Clear Search Cache", Value: "Press Enter", Type: "action"},
		{Key: "ClearFileListCache", DisplayName: "Clear File List Cache", Value: "Press Enter", Type: "action"},
	}
}

// Init initializes the config editor
func (cem *ConfigEditorModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the config editor
func (cem *ConfigEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If editing
		if cem.editing {
			switch msg.String() {
			case "enter":
				// Save edited value
				newValue := cem.textInput.Value()
				stylesChanged := cem.applyConfigChange(cem.cursor, newValue)
				if stylesChanged {
					cem.statusMsg = "Saved!"
					cem.editing = false
					cem.textInput.Blur()
					// Send style update message if styles were changed
					if cem.items[cem.cursor].Type == "color" ||
					   cem.items[cem.cursor].Key == "SelectorReverse" ||
					   cem.items[cem.cursor].Key == "SelectorBold" ||
					   cem.items[cem.cursor].Key == "SelectorReverseColor" ||
					   cem.items[cem.cursor].Key == "ContentBold" {
						return cem, func() tea.Msg { return stylesUpdatedMsg{} }
					}
					return cem, nil
				} else {
					cem.statusMsg = "Invalid value"
					cem.editing = false
					cem.textInput.Blur()
					return cem, nil
				}

			case "esc", "ctrl+c":
				// Cancel editing
				cem.editing = false
				cem.textInput.Blur()
				cem.statusMsg = ""
				return cem, nil
			}

			// Update text input
			cem.textInput, cmd = cem.textInput.Update(msg)
			return cem, cmd
		}

		// When viewing config list
		switch msg.String() {
		// Navigation
		case "up", "k":
			if cem.cursor > 0 {
				cem.cursor--
				if cem.cursor < cem.topRow {
					cem.topRow--
				}
			}
			cem.statusMsg = ""

		case "down", "j":
			if cem.cursor < len(cem.items)-1 {
				cem.cursor++
				availableLines := cem.getAvailableLines()
				if cem.cursor >= cem.topRow+availableLines {
					cem.topRow++
				}
			}
			cem.statusMsg = ""

		// Edit selected item or trigger action
		case "enter", "e":
			if len(cem.items) > 0 && cem.cursor < len(cem.items) {
				item := cem.items[cem.cursor]

				// Handle actions
				if item.Type == "action" {
					if item.Key == "ClearSearchCache" {
						if err := cache.ClearAllSearchCaches(); err != nil {
							cem.statusMsg = "Error clearing cache!"
						} else {
							cem.statusMsg = "Search cache cleared!"
						}
					} else if item.Key == "ClearFileListCache" {
						if err := cache.ClearFileListCache(); err != nil {
							cem.statusMsg = "Error clearing file list cache!"
						} else {
							cem.statusMsg = "File list cache cleared! Restart to rescan."
						}
					}
					break
				}

				// Don't allow editing colors (use arrows instead)
				if item.Type == "color" {
					cem.statusMsg = "Use ←→ to change colors"
					break
				}
				cem.textInput.SetValue(item.Value)
				cem.textInput.Focus()
				cem.editing = true
				cem.statusMsg = ""
				return cem, textinput.Blink
			}

		// Toggle boolean values
		case " ":
			if len(cem.items) > 0 && cem.cursor < len(cem.items) {
				item := cem.items[cem.cursor]
				if item.Type == "bool" {
					newValue := "true"
					if item.Value == "true" {
						newValue = "false"
					}
					if cem.applyConfigChange(cem.cursor, newValue) {
						cem.statusMsg = "Toggled!"
						// Send style update message if a style-related bool was toggled
						if item.Key == "SelectorReverse" || item.Key == "SelectorBold" || item.Key == "ContentBold" {
							return cem, func() tea.Msg { return stylesUpdatedMsg{} }
						}
					}
				}
			}

		// Cycle colors with left/right arrows
		case "left", "h":
			if len(cem.items) > 0 && cem.cursor < len(cem.items) {
				item := cem.items[cem.cursor]
				if item.Type == "color" {
					cem.cycleColor(cem.cursor, -1)
					cem.statusMsg = "Changed!"
					return cem, func() tea.Msg { return stylesUpdatedMsg{} }
				}
			}

		case "right", "l":
			if len(cem.items) > 0 && cem.cursor < len(cem.items) {
				item := cem.items[cem.cursor]
				if item.Type == "color" {
					cem.cycleColor(cem.cursor, 1)
					cem.statusMsg = "Changed!"
					return cem, func() tea.Msg { return stylesUpdatedMsg{} }
				}
			}

		// Return to dashboard (handled by app.go navigation)
		case "q":
			// Don't handle here - let app.go handle navigation
			break
		}
	}

	return cem, nil
}

// View renders the config editor
func (cem *ConfigEditorModel) View() string {
	if cem.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := "Configuration Editor"
	b.WriteString(cem.styles.RenderTitle(title, cem.width))
	b.WriteString("\n\n")

	if cem.editing {
		// Show edit input
		item := cem.items[cem.cursor]
		b.WriteString(fmt.Sprintf("  Editing: %s\n", item.DisplayName))
		b.WriteString("  Value: ")
		b.WriteString(cem.textInput.View())
		b.WriteString("\n")
		if item.Type == "bool" {
			b.WriteString("  (Enter 'true' or 'false')\n")
		}
		b.WriteString("  Press Enter to save, Esc to cancel\n\n")
	} else {
		// Show config list
		availableLines := cem.getAvailableLines()
		displayCount := min(len(cem.items), availableLines)

		for idx := 0; idx < displayCount; idx++ {
			itemIdx := cem.topRow + idx
			if itemIdx >= len(cem.items) {
				break
			}

			item := cem.items[itemIdx]
			isSelected := itemIdx == cem.cursor

			// Format: Setting Name: value (with special formatting for color/bool/action types)
			valueDisplay := item.Value
			if item.Type == "color" {
				valueDisplay = fmt.Sprintf("[◄ %s ►]", item.Value)
			} else if item.Type == "bool" {
				valueDisplay = fmt.Sprintf("[◄ %s ►]", item.Value)
			} else if item.Type == "action" {
				valueDisplay = fmt.Sprintf("%s  (↵ to execute)", item.Value)
			}
			line := fmt.Sprintf("  %-25s: %s", item.DisplayName, valueDisplay)

			// Truncate if too long
			if len(line) > cem.width-1 {
				line = line[:cem.width-4] + "..."
			}

			if isSelected {
				b.WriteString(cem.styles.ListCursor.Render(line))
			} else {
				b.WriteString(cem.styles.ListItem.Render(line))
			}
			b.WriteString("\n")
		}

		// Status message
		if cem.statusMsg != "" {
			b.WriteString("\n")
			b.WriteString(cem.styles.StatusBar.Render(fmt.Sprintf("  %s", cem.statusMsg)))
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	currentLines := strings.Count(b.String(), "\n")
	for currentLines < cem.height-2 {
		b.WriteString("\n")
		currentLines++
	}

	// Help bar
	helpText := "↑↓:Scroll ↵:Edit ←→:Color Space:Toggle q:Back"
	b.WriteString(cem.styles.RenderHelpBar(helpText, cem.width))

	return b.String()
}

// Helper methods

func (cem *ConfigEditorModel) getAvailableLines() int {
	// Height minus title(2), status area(2), help(1)
	return cem.height - 5
}

// applyConfigChange applies a configuration change
func (cem *ConfigEditorModel) applyConfigChange(itemIdx int, newValue string) bool {
	if itemIdx >= len(cem.items) {
		return false
	}

	item := &cem.items[itemIdx]

	// Validate and convert based on type
	switch item.Type {
	case "string":
		// String values are always valid
		item.Value = newValue

	case "bool":
		if newValue != "true" && newValue != "false" {
			return false
		}
		item.Value = newValue

	case "int":
		// Parse and validate integer
		val, err := strconv.Atoi(newValue)
		if err != nil || val < 0 {
			return false
		}
		item.Value = newValue

	case "color":
		// Validate that it's a valid color
		validColor := false
		for _, c := range availableColors {
			if c == newValue {
				validColor = true
				break
			}
		}
		if !validColor {
			return false
		}
		item.Value = newValue

	default:
		return false
	}

	// Apply to config
	c := cem.state.Config
	switch item.Key {
	case "SearchDir":
		c.SearchDir = item.Value
	case "ExportDir":
		c.ExportDir = item.Value
	case "Editor":
		c.Editor = item.Value
	case "UIColor":
		c.UIColor = item.Value
	case "SelectorColor":
		c.SelectorColor = item.Value
	case "SelectorReverse":
		c.SelectorReverse = item.Value == "true"
	case "SelectorBold":
		c.SelectorBold = item.Value == "true"
	case "SelectorReverseColor":
		c.SelectorReverseColor = item.Value
	case "ContentColor":
		c.ContentColor = item.Value
	case "ContentBold":
		c.ContentBold = item.Value == "true"
	case "CacheExpiryDays":
		val, _ := strconv.Atoi(item.Value)
		c.CacheExpiryDays = val
	}

	// Save config
	cem.state.Config.Save()

	// Rebuild styles if a color was changed (for live preview)
	if item.Type == "color" || item.Key == "SelectorReverse" || item.Key == "SelectorBold" ||
	   item.Key == "SelectorReverseColor" || item.Key == "ContentBold" {
		cem.styles = NewStyles(cem.state.Config)
		// Also rebuild the config items to reflect new values
		cem.buildConfigItems()
	}

	return true
}

// cycleColor cycles through available colors
func (cem *ConfigEditorModel) cycleColor(itemIdx int, direction int) {
	if itemIdx >= len(cem.items) {
		return
	}

	item := &cem.items[itemIdx]
	if item.Type != "color" {
		return
	}

	// Find current color index
	currentIdx := -1
	for i, color := range availableColors {
		if color == item.Value {
			currentIdx = i
			break
		}
	}

	// If current color not found, start at 0
	if currentIdx == -1 {
		currentIdx = 0
	}

	// Cycle to next/previous color
	newIdx := currentIdx + direction
	if newIdx < 0 {
		newIdx = len(availableColors) - 1
	} else if newIdx >= len(availableColors) {
		newIdx = 0
	}

	// Apply the change
	cem.applyConfigChange(itemIdx, availableColors[newIdx])
}

// Ensure ConfigEditorModel implements tea.Model
var _ tea.Model = (*ConfigEditorModel)(nil)
