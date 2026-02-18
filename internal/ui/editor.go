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

// editorSegment holds a wrapped visual segment of a display line.
type editorSegment struct {
	text  string
	start int // byte offset in the tab-expanded display string where this segment starts
}

// editorWrapSegments splits a tab-expanded line into wrapped segments that fit within width bytes.
// It breaks at word boundaries (spaces) where possible, otherwise hard-breaks at width.
func editorWrapSegments(line string, width int) []editorSegment {
	if width <= 0 || len(line) <= width {
		return []editorSegment{{text: line, start: 0}}
	}

	var segs []editorSegment
	pos := 0
	n := len(line)

	for pos < n {
		if n-pos <= width {
			segs = append(segs, editorSegment{text: line[pos:], start: pos})
			break
		}

		// Find last ASCII space in [pos, pos+width) to break at a word boundary.
		end := pos + width
		breakAt := end
		for i := end - 1; i > pos; i-- {
			if line[i] == ' ' { // safe: ' ' is single-byte ASCII
				breakAt = i + 1 // include the space in this segment
				break
			}
		}

		segs = append(segs, editorSegment{text: line[pos:breakAt], start: pos})
		pos = breakAt
	}

	if len(segs) == 0 {
		return []editorSegment{{text: line, start: 0}}
	}
	return segs
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

	// Word wrap toggle
	wordWrap bool

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

		case tea.KeyCtrlW:
			em.wordWrap = !em.wordWrap

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

		case tea.KeySpace:
			em.insertChar(' ')

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
	if !em.wordWrap {
		if em.cursorRow > 0 {
			em.cursorRow--
			em.clampCursorCol()
			em.ensureCursorVisible()
		}
		return
	}

	// Word wrap: move by one visual row.
	line := em.currentLine()
	displayLine := expandTabs(line, editorTabWidth)
	displayCursorCol := tabExpandedCol(line, em.cursorCol, editorTabWidth)
	segs := editorWrapSegments(displayLine, em.availWidth())
	curSegIdx, cursorPosInSeg := em.findSegment(segs, displayCursorCol)

	if curSegIdx > 0 {
		// Previous visual row is the previous segment of the same logical line.
		prev := segs[curSegIdx-1]
		target := prev.start + min(cursorPosInSeg, len(prev.text))
		em.cursorCol = tabExpandedColToByteOffset(line, target, editorTabWidth)
	} else if em.cursorRow > 0 {
		// Move to the last segment of the previous logical line.
		em.cursorRow--
		prevLine := em.currentLine()
		prevDisplay := expandTabs(prevLine, editorTabWidth)
		prevSegs := editorWrapSegments(prevDisplay, em.availWidth())
		last := prevSegs[len(prevSegs)-1]
		target := last.start + min(cursorPosInSeg, len(last.text))
		em.cursorCol = tabExpandedColToByteOffset(prevLine, target, editorTabWidth)
	}
	em.ensureCursorVisible()
}

func (em *EditorModel) moveCursorDown() {
	if !em.wordWrap {
		if em.cursorRow < len(em.lines)-1 {
			em.cursorRow++
			em.clampCursorCol()
			em.ensureCursorVisible()
		}
		return
	}

	// Word wrap: move by one visual row.
	line := em.currentLine()
	displayLine := expandTabs(line, editorTabWidth)
	displayCursorCol := tabExpandedCol(line, em.cursorCol, editorTabWidth)
	segs := editorWrapSegments(displayLine, em.availWidth())
	curSegIdx, cursorPosInSeg := em.findSegment(segs, displayCursorCol)

	if curSegIdx < len(segs)-1 {
		// Next visual row is the next segment of the same logical line.
		next := segs[curSegIdx+1]
		target := next.start + min(cursorPosInSeg, len(next.text))
		em.cursorCol = tabExpandedColToByteOffset(line, target, editorTabWidth)
	} else if em.cursorRow < len(em.lines)-1 {
		// Move to the first segment of the next logical line.
		em.cursorRow++
		nextLine := em.currentLine()
		nextDisplay := expandTabs(nextLine, editorTabWidth)
		nextSegs := editorWrapSegments(nextDisplay, em.availWidth())
		first := nextSegs[0]
		target := first.start + min(cursorPosInSeg, len(first.text))
		em.cursorCol = tabExpandedColToByteOffset(nextLine, target, editorTabWidth)
	}
	em.ensureCursorVisible()
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
	if !em.wordWrap {
		em.cursorRow -= visible
		if em.cursorRow < 0 {
			em.cursorRow = 0
		}
		em.clampCursorCol()
		em.ensureCursorVisible()
		return
	}
	// Word wrap: scroll back by approximately one screenful of visual rows.
	rowsToSkip := visible
	for rowsToSkip > 0 {
		if em.cursorRow == 0 {
			break
		}
		em.cursorRow--
		rowsToSkip -= em.lineVisualRows(em.cursorRow)
	}
	em.clampCursorCol()
	em.ensureCursorVisible()
}

func (em *EditorModel) pageDown() {
	visible := em.getVisibleLines()
	if !em.wordWrap {
		em.cursorRow += visible
		if em.cursorRow >= len(em.lines) {
			em.cursorRow = len(em.lines) - 1
		}
		em.clampCursorCol()
		em.ensureCursorVisible()
		return
	}
	// Word wrap: advance by approximately one screenful of visual rows.
	rowsToSkip := visible
	for rowsToSkip > 0 && em.cursorRow < len(em.lines)-1 {
		rowsToSkip -= em.lineVisualRows(em.cursorRow)
		em.cursorRow++
	}
	if em.cursorRow >= len(em.lines) {
		em.cursorRow = len(em.lines) - 1
	}
	em.clampCursorCol()
	em.ensureCursorVisible()
}

// availWidth returns the usable content width after subtracting the line-number gutter.
func (em *EditorModel) availWidth() int {
	lineNumWidth := len(fmt.Sprintf("%d", len(em.lines))) + 1
	w := em.width - lineNumWidth - 1
	if w < 1 {
		return 1
	}
	return w
}

// findSegment returns the index of the segment containing displayCursorCol and
// the cursor's column offset within that segment.
func (em *EditorModel) findSegment(segs []editorSegment, displayCursorCol int) (segIdx int, posInSeg int) {
	segIdx = len(segs) - 1
	for i, seg := range segs {
		segEnd := seg.start + len(seg.text)
		if displayCursorCol < segEnd {
			segIdx = i
			break
		}
	}
	posInSeg = displayCursorCol - segs[segIdx].start
	return
}

// tabExpandedColToByteOffset is the inverse of tabExpandedCol: given a display
// column in the tab-expanded string, it returns the byte offset in the original s.
func tabExpandedColToByteOffset(s string, expandedCol int, tw int) int {
	col := 0
	for i := 0; i < len(s); i++ {
		if col >= expandedCol {
			return i
		}
		if s[i] == '\t' {
			col += tw - (col % tw)
		} else {
			col++
		}
	}
	return len(s)
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
		return
	}

	if !em.wordWrap {
		// Scroll down if cursor is below viewport
		if em.cursorRow >= em.topRow+visible {
			em.topRow = em.cursorRow - visible + 1
		}
		return
	}

	// Word wrap: advance topRow until cursor's visual rows fit in viewport.
	for {
		visualRows := 0
		for i := em.topRow; i <= em.cursorRow; i++ {
			visualRows += em.lineVisualRows(i)
		}
		if visualRows <= visible {
			break
		}
		em.topRow++
		if em.topRow > em.cursorRow {
			em.topRow = em.cursorRow
			break
		}
	}
}

// lineVisualRows returns the number of visual rows a logical line occupies when word-wrapped.
func (em *EditorModel) lineVisualRows(lineIdx int) int {
	if !em.wordWrap || em.width <= 0 {
		return 1
	}
	lineNumWidth := len(fmt.Sprintf("%d", len(em.lines))) + 1
	availWidth := em.width - lineNumWidth - 1
	if availWidth <= 0 {
		return 1
	}
	displayLine := expandTabs(em.lines[lineIdx], editorTabWidth)
	segs := editorWrapSegments(displayLine, availWidth)
	if len(segs) == 0 {
		return 1
	}
	return len(segs)
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
	availWidth := em.width - lineNumWidth - 1

	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("230")).
		Foreground(lipgloss.Color("0"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	visualRowsUsed := 0
	lineIdx := em.topRow

	for visualRowsUsed < visibleLines {
		if lineIdx >= len(em.lines) {
			b.WriteString("\n")
			visualRowsUsed++
			continue
		}

		lineNumStr := fmt.Sprintf("%*d ", lineNumWidth, lineIdx+1)
		line := em.lines[lineIdx]

		if !em.wordWrap {
			b.WriteString(lineNumStyle.Render(lineNumStr))
			if lineIdx == em.cursorRow {
				b.WriteString(em.renderLineWithCursor(line, lineNumWidth))
			} else {
				displayLine := expandTabs(line, editorTabWidth)
				if len(displayLine) > availWidth {
					displayLine = displayLine[:availWidth]
				}
				b.WriteString(displayLine)
			}
			b.WriteString("\n")
			visualRowsUsed++
		} else {
			// Word wrap: render each segment as its own visual row.
			displayLine := expandTabs(line, editorTabWidth)
			segs := editorWrapSegments(displayLine, availWidth)
			if len(segs) == 0 {
				segs = []editorSegment{{text: "", start: 0}}
			}

			displayCursorCol := -1
			if lineIdx == em.cursorRow {
				displayCursorCol = tabExpandedCol(line, em.cursorCol, editorTabWidth)
			}

			for segIdx, seg := range segs {
				if visualRowsUsed >= visibleLines {
					break
				}

				// First segment shows the line number; continuation rows indent.
				if segIdx == 0 {
					b.WriteString(lineNumStyle.Render(lineNumStr))
				} else {
					b.WriteString(strings.Repeat(" ", lineNumWidth+1))
				}

				if displayCursorCol >= 0 {
					isLastSeg := segIdx == len(segs)-1
					segLen := len(seg.text)
					inSeg := seg.start <= displayCursorCol &&
						(displayCursorCol < seg.start+segLen || isLastSeg)

					if inSeg {
						cursorPosInSeg := displayCursorCol - seg.start
						for i, ch := range seg.text {
							if i == cursorPosInSeg {
								b.WriteString(cursorStyle.Render(string(ch)))
							} else {
								b.WriteRune(ch)
							}
						}
						if cursorPosInSeg >= segLen {
							b.WriteString(cursorStyle.Render(" "))
						}
					} else {
						b.WriteString(seg.text)
					}
				} else {
					b.WriteString(seg.text)
				}

				b.WriteString("\n")
				visualRowsUsed++
			}
		}

		lineIdx++
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

	wrapLabel := "^W:Wrap"
	if em.wordWrap {
		wrapLabel = "^W:Wrap[ON]"
	}
	footer1 := fmt.Sprintf(" Ctrl+S:Save  Esc:Exit  %s  Arrows:Move  PgUp/PgDn:Scroll ", wrapLabel)
	footer2 := fmt.Sprintf(" %s ", status)

	b.WriteString(footerStyle.Render(footer1))
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(footer2))

	return b.String()
}

func (em *EditorModel) renderLineWithCursor(line string, lineNumWidth int) string {
	availWidth := em.width - lineNumWidth - 1

	// Expand tabs for display and map cursor position
	displayLine := expandTabs(line, editorTabWidth)
	displayCursorCol := tabExpandedCol(line, em.cursorCol, editorTabWidth)

	// If line is longer than available width, we need to scroll horizontally
	startCol := 0
	if displayCursorCol >= availWidth {
		startCol = displayCursorCol - availWidth + 1
	}

	// Extract visible portion
	visibleLine := ""
	if startCol < len(displayLine) {
		endCol := startCol + availWidth
		if endCol > len(displayLine) {
			endCol = len(displayLine)
		}
		visibleLine = displayLine[startCol:endCol]
	}

	// Adjust cursor position for display
	displayCursorPos := displayCursorCol - startCol

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

const editorTabWidth = 4

// expandTabs replaces tab characters with spaces aligned to tab stops.
func expandTabs(s string, tw int) string {
	var b strings.Builder
	col := 0
	for _, ch := range s {
		if ch == '\t' {
			spaces := tw - (col % tw)
			for j := 0; j < spaces; j++ {
				b.WriteByte(' ')
			}
			col += spaces
		} else {
			b.WriteRune(ch)
			col++
		}
	}
	return b.String()
}

// tabExpandedCol maps a byte offset in s to a display column after tab expansion.
func tabExpandedCol(s string, byteCol int, tw int) int {
	displayCol := 0
	for i := 0; i < len(s) && i < byteCol; i++ {
		if s[i] == '\t' {
			displayCol += tw - (displayCol % tw)
		} else {
			displayCol++
		}
	}
	return displayCol
}

// Ensure EditorModel implements tea.Model
var _ tea.Model = (*EditorModel)(nil)
