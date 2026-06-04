package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"argc.dev/chiquito/internal/syntax"
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

	b.WriteString(m.statusBar())
	return b.String()
}

// ensureSyntax recomputes the per-line entering lexical states when the document
// has changed. This is O(lines) but happens at most once per edit, not per
// frame; tokens for the visible rows are derived from these cached states.
func (m *Model) ensureSyntax() {
	if !m.cfg.Features.SyntaxHighlighting || m.lang == nil {
		return
	}
	if !m.synStale && len(m.enterStates) == m.ed.LineCount() {
		return
	}
	n := m.ed.LineCount()
	states := make([]syntax.State, n)
	st := syntax.StateDefault
	for i := 0; i < n; i++ {
		states[i] = st
		_, st = m.lang.TokenizeLine(m.ed.Line(i), st)
	}
	m.enterStates = states
	m.synStale = false
}

// lineStyles builds a per-rune style slice for a line: syntax token colors,
// overlaid with search-match highlights.
func (m *Model) lineStyles(lineIdx int, runes []rune) []lipgloss.Style {
	styles := make([]lipgloss.Style, len(runes))

	if m.cfg.Features.SyntaxHighlighting && m.lang != nil && lineIdx < len(m.enterStates) {
		tokens, _ := m.lang.TokenizeLine(string(runes), m.enterStates[lineIdx])
		for _, tk := range tokens {
			st := m.theme.style(tk.Type)
			for i := tk.Start; i < tk.End && i < len(styles); i++ {
				styles[i] = st
			}
		}
	}

	// Overlay search matches (only while a search prompt is active).
	if m.mode == modeSearch && len(m.matches) > 0 {
		lineStart := m.ed.LineStartPos(lineIdx)
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
	right := fmt.Sprintf(" %s  %d:%d ", m.lang.Name(), line+1, col+1)

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
		} else if m.query != "" {
			count = " (no matches)"
		}
		return fmt.Sprintf(" I-search[%s]: %s%s", cs, m.query, count)
	case modeReplaceFrom:
		return fmt.Sprintf(" Replace[%s]: %s", cs, m.query)
	case modeReplaceTo:
		return fmt.Sprintf(" Replace %q with: %s", m.query, m.replaceWith)
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
	h := m.height - 1
	if h < 1 {
		return 1
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
