package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// langName returns the active highlighter's language name (for the status bar).
func (m *Model) langName() string {
	if m.hl == nil {
		return "plain"
	}
	return m.hl.name
}

var (
	gutterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	tildeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	cursorStyle = lipgloss.NewStyle().Reverse(true)
	statusStyle = lipgloss.NewStyle().Reverse(true)

	misspellColor = lipgloss.Color("9")
)

// View implements tea.Model.
func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	m.ensureSyntax()

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
		content := m.ed.Line(ln)
		col := -1
		if ln == curLine {
			col = curCol
		}
		b.WriteString(m.renderRow(ln, content, col))
		b.WriteByte('\n')
	}

	if ph := m.paneHeight(); ph > 0 {
		b.WriteString(m.activePane.view(m.width, ph))
		b.WriteByte('\n')
	}

	b.WriteString(m.statusBar())
	return b.String()
}

// ensureSyntax re-tokenizes the whole document into per-line styled spans when
// it has changed. This is O(document) but happens at most once per edit, not per
// frame; the visible rows just read the cached spans.
func (m *Model) ensureSyntax() {
	if !m.cfg.Features.SyntaxHighlighting || m.hl == nil {
		return
	}
	if !m.synStale {
		return
	}
	m.hl.highlight(m.ed.Text())
	m.synStale = false
}

// lineStyles builds a per-rune style slice for a line: syntax token colors,
// overlaid with spell and search highlights.
func (m *Model) lineStyles(lineIdx int, runes []rune) []lipgloss.Style {
	styles := make([]lipgloss.Style, len(runes))

	if m.cfg.Features.SyntaxHighlighting && m.hl != nil {
		for _, sp := range m.hl.spansFor(lineIdx) {
			for i := sp.start; i < sp.end && i < len(styles); i++ {
				styles[i] = sp.style
			}
		}
	}

	lineStart := m.ed.LineStartPos(lineIdx)

	// Underline misspelled words (normal editing only; search overlay wins).
	if m.mode == modeNormal && len(m.spellSpans) > 0 {
		for _, sp := range m.spellSpans {
			s := sp.Start - lineStart
			e := sp.End - lineStart
			if e <= 0 || s >= len(styles) {
				continue
			}
			for i := maxInt(0, s); i < minInt(len(styles), e); i++ {
				styles[i] = styles[i].Underline(true).Foreground(misspellColor)
			}
		}
	}

	// Overlay search matches (only while a search prompt is active).
	if m.mode == modeSearch && len(m.matches) > 0 {
		for mi, mt := range m.matches {
			s := mt.Start - lineStart
			e := mt.End - lineStart
			if e <= 0 || s >= len(styles) {
				continue
			}
			hl := m.theme.searchMatch
			if mi == m.matchIdx {
				hl = m.theme.currentHit
			}
			for i := maxInt(0, s); i < minInt(len(styles), e); i++ {
				styles[i] = hl
			}
		}
	}
	return styles
}

// renderRow renders one text line clipped to the horizontal window
// [hscroll, hscroll+textWidth) in display columns, applying per-rune styles and
// the cursor. Tabs expand to the next tab stop; wide runes advance by their
// display width so the clip and cursor stay aligned.
func (m *Model) renderRow(lineIdx int, content string, cursorCol int) string {
	var sb strings.Builder
	width := m.textWidth()
	runes := []rune(content)
	styles := m.lineStyles(lineIdx, runes)
	dispCol := 0
	emitted := 0

	for i, r := range runes {
		w := runewidth.RuneWidth(r)
		if r == '\t' {
			w = m.tabWidth - (dispCol % m.tabWidth)
		}
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
		style := styles[i]
		if i == cursorCol {
			style = style.Reverse(true)
		}
		sb.WriteString(style.Render(cell))
		dispCol += w
		emitted += w
	}

	if cursorCol >= len(runes) && cursorCol >= 0 && emitted < width {
		sb.WriteString(cursorStyle.Render(" "))
	}
	return sb.String()
}

func (m *Model) statusBar() string {
	// An active prompt takes over the status line.
	if prompt := m.prompt(); prompt != "" {
		return statusStyle.Render(m.padTo(prompt, m.width))
	}

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
	right := fmt.Sprintf(" %s  %d:%d ", m.langName(), line+1, col+1)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
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

// prompt returns the minibuffer prompt text for the active mode, or "".
func (m *Model) prompt() string {
	cs := "i"
	if m.caseSensitive {
		cs = "C"
	}
	switch m.mode {
	case modeSearch:
		count := ""
		if len(m.matches) > 0 {
			count = fmt.Sprintf(" (%d/%d)", m.matchIdx+1, len(m.matches))
		} else if m.query.String() != "" {
			count = " (no matches)"
		}
		return fmt.Sprintf(" I-search[%s]: %s%s", cs, m.query.String(), count)
	case modeReplaceFrom:
		return fmt.Sprintf(" Replace[%s]: %s", cs, m.query.String())
	case modeReplaceTo:
		return fmt.Sprintf(" Replace %q with: %s", m.query.String(), m.replaceWith.String())
	case modeOpen:
		return fmt.Sprintf(" Open file: %s", m.input.String())
	case modeSaveAs:
		return fmt.Sprintf(" Save as: %s", m.input.String())
	}
	return ""
}

func (m *Model) padTo(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return runewidth.Truncate(s, width, "…")
}

// --- layout helpers --------------------------------------------------------

func (m *Model) gutterWidth() int {
	if !m.lineNumbers {
		return 0
	}
	return len(strconv.Itoa(m.ed.LineCount())) + 1
}

func (m *Model) textWidth() int {
	w := m.width - m.gutterWidth()
	if w < 1 {
		return 1
	}
	return w
}

func (m *Model) textHeight() int {
	h := m.height - 1 - m.paneHeight() // status bar + any open pane
	if h < 1 {
		return 1
	}
	return h
}

// paneHeight is the number of rows reserved for an open pane, clamped so at
// least one editor row and the status bar remain.
func (m *Model) paneHeight() int {
	if m.activePane == nil {
		return 0
	}
	h := m.activePane.preferredHeight()
	if max := m.height - 2; h > max {
		h = max
	}
	if h < 1 {
		h = 1
	}
	return h
}

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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
