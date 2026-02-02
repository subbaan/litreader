package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/subbass/litreader/internal/state"
)

// editorDoneMsg is sent when the editor is closed
type editorDoneMsg struct {
	saved bool
}

// EditorModel represents a simple text editor
type EditorModel struct {
	state  *state.State
	styles *Styles

	width  int
	height int

	// File
	filename string
	lines    []string
	modified bool

	// Cursor position
	cursorRow int
	cursorCol int

	// Viewport (top visible line)
	topRow int

	// Status message
	statusMsg string
}

// NewEditorModel creates a new editor for the given file
func NewEditorModel(s *state.State, styles *Styles, filename string, startLine int) *EditorModel {
	em := &EditorModel{
		state:    s,
		styles:   styles,
		filename: filename,
	}

	// Load file
	em.loadFile()

	// Position cursor at requested line
	if startLine > 0 && startLine <= len(em.lines) {
		em.cursorRow = startLine - 1
		em.topRow = max(0, em.cursorRow-5)
	}

	return em
}

func (em *EditorModel) loadFile() {
	data, err := os.ReadFile(em.filename)
	if err != nil {
		em.lines = []string{""}
		em.statusMsg = fmt.Sprintf("Error loading file: %v", err)
		return
	}

	content := string(data)
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	em.lines = strings.Split(content, "\n")
	if len(em.lines) == 0 {
		em.lines = []string{""}
	}
	em.modified = false
}

func (em *EditorModel) saveFile() error {
	content := strings.Join(em.lines, "\n")
	err := os.WriteFile(em.filename, []byte(content), 0644)
	if err != nil {
		em.statusMsg = fmt.Sprintf("Error saving: %v", err)
		return err
	}
	em.modified = false
	em.statusMsg = "Saved"
	return nil
}

// Init initializes the editor
func (em *EditorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the editor
func (em *EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		em.statusMsg = "" // Clear status on any key

		switch msg.Type {
		case tea.KeyEsc:
			// Exit editor
			return em, func() tea.Msg { return editorDoneMsg{saved: false} }

		case tea.KeyCtrlS:
			// Save
			em.saveFile()
			return em, nil

		case tea.KeyUp:
			em.moveCursorUp()
		case tea.KeyDown:
			em.moveCursorDown()
		case tea.KeyLeft:
			em.moveCursorLeft()
		case tea.KeyRight:
			em.moveCursorRight()

		case tea.KeyHome:
			em.cursorCol = 0
		case tea.KeyEnd:
			em.cursorCol = len(em.currentLine())

		case tea.KeyPgUp:
			em.pageUp()
		case tea.KeyPgDown:
			em.pageDown()

		case tea.KeyBackspace:
			em.backspace()
		case tea.KeyDelete:
			em.delete()

		case tea.KeyEnter:
			em.insertNewline()

		case tea.KeyTab:
			em.insertChar('\t')

		case tea.KeyRunes:
			for _, r := range msg.Runes {
				em.insertChar(r)
			}
		}
	}

	return em, nil
}

func (em *EditorModel) currentLine() string {
	if em.cursorRow >= 0 && em.cursorRow < len(em.lines) {
		return em.lines[em.cursorRow]
	}
	return ""
}

func (em *EditorModel) moveCursorUp() {
	if em.cursorRow > 0 {
		em.cursorRow--
		em.clampCursorCol()
		em.ensureCursorVisible()
	}
}

func (em *EditorModel) moveCursorDown() {
	if em.cursorRow < len(em.lines)-1 {
		em.cursorRow++
		em.clampCursorCol()
		em.ensureCursorVisible()
	}
}

func (em *EditorModel) moveCursorLeft() {
	if em.cursorCol > 0 {
		em.cursorCol--
	} else if em.cursorRow > 0 {
		// Move to end of previous line
		em.cursorRow--
		em.cursorCol = len(em.currentLine())
		em.ensureCursorVisible()
	}
}

func (em *EditorModel) moveCursorRight() {
	lineLen := len(em.currentLine())
	if em.cursorCol < lineLen {
		em.cursorCol++
	} else if em.cursorRow < len(em.lines)-1 {
		// Move to start of next line
		em.cursorRow++
		em.cursorCol = 0
		em.ensureCursorVisible()
	}
}

func (em *EditorModel) pageUp() {
	visible := em.getVisibleLines()
	em.cursorRow -= visible
	if em.cursorRow < 0 {
		em.cursorRow = 0
	}
	em.clampCursorCol()
	em.ensureCursorVisible()
}

func (em *EditorModel) pageDown() {
	visible := em.getVisibleLines()
	em.cursorRow += visible
	if em.cursorRow >= len(em.lines) {
		em.cursorRow = len(em.lines) - 1
	}
	em.clampCursorCol()
	em.ensureCursorVisible()
}

func (em *EditorModel) clampCursorCol() {
	lineLen := len(em.currentLine())
	if em.cursorCol > lineLen {
		em.cursorCol = lineLen
	}
}

func (em *EditorModel) ensureCursorVisible() {
	visible := em.getVisibleLines()

	// Scroll up if cursor is above viewport
	if em.cursorRow < em.topRow {
		em.topRow = em.cursorRow
	}

	// Scroll down if cursor is below viewport
	if em.cursorRow >= em.topRow+visible {
		em.topRow = em.cursorRow - visible + 1
	}
}

func (em *EditorModel) getVisibleLines() int {
	// Height minus header (1) and footer (2)
	return em.height - 3
}

func (em *EditorModel) backspace() {
	if em.cursorCol > 0 {
		// Delete character before cursor
		line := em.lines[em.cursorRow]
		em.lines[em.cursorRow] = line[:em.cursorCol-1] + line[em.cursorCol:]
		em.cursorCol--
		em.modified = true
	} else if em.cursorRow > 0 {
		// Join with previous line
		prevLine := em.lines[em.cursorRow-1]
		em.cursorCol = len(prevLine)
		em.lines[em.cursorRow-1] = prevLine + em.lines[em.cursorRow]
		em.lines = append(em.lines[:em.cursorRow], em.lines[em.cursorRow+1:]...)
		em.cursorRow--
		em.modified = true
		em.ensureCursorVisible()
	}
}

func (em *EditorModel) delete() {
	line := em.currentLine()
	if em.cursorCol < len(line) {
		// Delete character at cursor
		em.lines[em.cursorRow] = line[:em.cursorCol] + line[em.cursorCol+1:]
		em.modified = true
	} else if em.cursorRow < len(em.lines)-1 {
		// Join with next line
		em.lines[em.cursorRow] = line + em.lines[em.cursorRow+1]
		em.lines = append(em.lines[:em.cursorRow+1], em.lines[em.cursorRow+2:]...)
		em.modified = true
	}
}

func (em *EditorModel) insertNewline() {
	line := em.currentLine()
	// Split line at cursor
	before := line[:em.cursorCol]
	after := line[em.cursorCol:]

	em.lines[em.cursorRow] = before

	// Insert new line
	newLines := make([]string, len(em.lines)+1)
	copy(newLines[:em.cursorRow+1], em.lines[:em.cursorRow+1])
	newLines[em.cursorRow+1] = after
	copy(newLines[em.cursorRow+2:], em.lines[em.cursorRow+1:])
	em.lines = newLines

	em.cursorRow++
	em.cursorCol = 0
	em.modified = true
	em.ensureCursorVisible()
}

func (em *EditorModel) insertChar(ch rune) {
	line := em.currentLine()
	em.lines[em.cursorRow] = line[:em.cursorCol] + string(ch) + line[em.cursorCol:]
	em.cursorCol++
	em.modified = true
}

// View renders the editor
func (em *EditorModel) View() string {
	var b strings.Builder

	// Header
	modifiedMarker := ""
	if em.modified {
		modifiedMarker = " [modified]"
	}
	header := fmt.Sprintf(" EDITOR: %s%s ", em.filename, modifiedMarker)
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Width(em.width)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	// Content area
	visibleLines := em.getVisibleLines()
	lineNumWidth := len(fmt.Sprintf("%d", len(em.lines))) + 1

	for i := 0; i < visibleLines; i++ {
		lineIdx := em.topRow + i
		if lineIdx >= len(em.lines) {
			b.WriteString("\n")
			continue
		}

		// Line number
		lineNum := fmt.Sprintf("%*d ", lineNumWidth, lineIdx+1)
		lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		b.WriteString(lineNumStyle.Render(lineNum))

		// Line content with cursor
		line := em.lines[lineIdx]
		if lineIdx == em.cursorRow {
			// Render line with cursor
			b.WriteString(em.renderLineWithCursor(line, lineNumWidth))
		} else {
			// Truncate if needed
			availWidth := em.width - lineNumWidth - 1
			if len(line) > availWidth {
				line = line[:availWidth]
			}
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Width(em.width)

	status := em.statusMsg
	if status == "" {
		status = fmt.Sprintf("Line %d/%d, Col %d", em.cursorRow+1, len(em.lines), em.cursorCol+1)
	}

	footer1 := " Ctrl+S:Save  Esc:Exit  Arrows:Move  PgUp/PgDn:Scroll "
	footer2 := fmt.Sprintf(" %s ", status)

	b.WriteString(footerStyle.Render(footer1))
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(footer2))

	return b.String()
}

func (em *EditorModel) renderLineWithCursor(line string, lineNumWidth int) string {
	availWidth := em.width - lineNumWidth - 1

	// Handle cursor position
	cursorPos := em.cursorCol

	// If line is longer than available width, we need to scroll horizontally
	startCol := 0
	if cursorPos >= availWidth {
		startCol = cursorPos - availWidth + 1
	}

	// Extract visible portion
	visibleLine := ""
	if startCol < len(line) {
		endCol := startCol + availWidth
		if endCol > len(line) {
			endCol = len(line)
		}
		visibleLine = line[startCol:endCol]
	}

	// Adjust cursor position for display
	displayCursorPos := cursorPos - startCol

	// Build the line with cursor highlight
	var result strings.Builder
	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("230")).
		Foreground(lipgloss.Color("0"))

	for i, ch := range visibleLine {
		if i == displayCursorPos {
			result.WriteString(cursorStyle.Render(string(ch)))
		} else {
			result.WriteRune(ch)
		}
	}

	// If cursor is at end of line (or beyond visible content)
	if displayCursorPos >= len(visibleLine) {
		result.WriteString(cursorStyle.Render(" "))
	}

	return result.String()
}

// Ensure EditorModel implements tea.Model
var _ tea.Model = (*EditorModel)(nil)
