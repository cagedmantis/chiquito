package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	gutterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	tildeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	cursorStyle = lipgloss.NewStyle().Reverse(true)
	statusStyle = lipgloss.NewStyle().Reverse(true)
)

// View implements tea.Model.
func (m *Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	top := m.ed.Top()
	height := m.textHeight()
	gutter := m.gutterWidth()
	lineCount := m.ed.LineCount()
	curLine, curCol := m.ed.CursorLineCol()

	for row := 0; row < height; row++ {
		ln := top + row
		if ln >= lineCount {
			b.WriteString(tildeStyle.Render("~"))
			b.WriteByte('\n')
			continue
		}
		if m.lineNumbers {
			b.WriteString(gutterStyle.Render(fmt.Sprintf("%*d ", gutter-1, ln+1)))
		}
		isCursor := ln == curLine
		col := -1
		if isCursor {
			col = curCol
		}
		b.WriteString(m.renderRow(m.ed.Line(ln), col))
		b.WriteByte('\n')
	}

	b.WriteString(m.statusBar())
	return b.String()
}

// renderRow renders one text line clipped to the horizontal window
// [hscroll, hscroll+textWidth) in display columns. If cursorCol >= 0 the rune at
// that column (or a trailing space) is drawn with the cursor style. Tabs are
// expanded to the next tab stop; wide runes (CJK/emoji) advance by their display
// width so the clip and cursor stay aligned.
func (m *Model) renderRow(content string, cursorCol int) string {
	var sb strings.Builder
	width := m.textWidth()
	runes := []rune(content)
	dispCol := 0 // display column of the next rune, from the start of the line
	emitted := 0 // display columns already written within the window

	for i, r := range runes {
		w := runewidth.RuneWidth(r)
		if r == '\t' {
			w = m.tabWidth - (dispCol % m.tabWidth)
		}
		// Skip runes scrolled off to the left.
		if dispCol+w <= m.hscroll {
			dispCol += w
			continue
		}
		if emitted+w > width {
			break
		}
		cell := string(r)
		if r == '\t' {
			cell = strings.Repeat(" ", w)
		}
		if i == cursorCol {
			sb.WriteString(cursorStyle.Render(cell))
		} else {
			sb.WriteString(cell)
		}
		dispCol += w
		emitted += w
	}

	// Cursor sitting at end-of-line: draw it as a reversed space.
	if cursorCol >= len(runes) && cursorCol >= 0 && emitted < width {
		sb.WriteString(cursorStyle.Render(" "))
	}
	return sb.String()
}

func (m *Model) statusBar() string {
	name := m.ed.Name()
	if name == "" {
		name = "[scratch]"
	}
	flag := ""
	if m.ed.Dirty() {
		flag = " *"
	}
	line, col := m.ed.CursorLineCol()

	left := fmt.Sprintf(" %s%s ", name, flag)
	if m.status != "" {
		left += "| " + m.status + " "
	}
	right := fmt.Sprintf(" %d:%d ", line+1, col+1)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		// Truncate the left side so the position indicator always fits.
		max := m.width - lipgloss.Width(right)
		if max < 0 {
			max = 0
		}
		left = runewidth.Truncate(left, max, "…")
		gap = m.width - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 0 {
			gap = 0
		}
	}
	return statusStyle.Render(left + strings.Repeat(" ", gap) + right)
}

// --- layout helpers --------------------------------------------------------

// gutterWidth returns the width of the line-number column (including its
// trailing space), or 0 when line numbers are disabled.
func (m *Model) gutterWidth() int {
	if !m.lineNumbers {
		return 0
	}
	return len(strconv.Itoa(m.ed.LineCount())) + 1
}

// textWidth is the number of columns available for text, after the gutter.
func (m *Model) textWidth() int {
	w := m.width - m.gutterWidth()
	if w < 1 {
		return 1
	}
	return w
}

// textHeight is the number of rows available for text, reserving one row for
// the status bar.
func (m *Model) textHeight() int {
	h := m.height - 1
	if h < 1 {
		return 1
	}
	return h
}

// syncHScroll adjusts the horizontal scroll so the cursor's display column stays
// within the visible window.
func (m *Model) syncHScroll() {
	line, col := m.ed.CursorLineCol()
	target := displayCol(m.ed.Line(line), col, m.tabWidth)
	w := m.textWidth()
	if target < m.hscroll {
		m.hscroll = target
	}
	if target >= m.hscroll+w {
		m.hscroll = target - w + 1
	}
	if m.hscroll < 0 {
		m.hscroll = 0
	}
}

// displayCol returns the display column at rune offset col within line, honoring
// tab stops and wide runes.
func displayCol(line string, col, tabWidth int) int {
	dc := 0
	for i, r := range []rune(line) {
		if i >= col {
			break
		}
		if r == '\t' {
			dc += tabWidth - (dc % tabWidth)
		} else {
			dc += runewidth.RuneWidth(r)
		}
	}
	return dc
}
