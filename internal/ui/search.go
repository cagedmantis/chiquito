package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/search"
)

// handleMinibuffer routes keys while a search, replace, or file prompt is active.
func (m *Model) handleMinibuffer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	chord := msg.String()

	if chord == "esc" || chord == "ctrl+g" {
		m.cancelPrompt()
		return m, nil
	}

	switch m.mode {
	case modeSearch:
		return m.searchKey(msg, chord)
	case modeReplaceFrom:
		return m.replaceFromKey(msg)
	case modeReplaceTo:
		return m.replaceToKey(msg)
	case modeOpen, modeSaveAs:
		return m.promptKey(msg)
	}
	return m, nil
}

func (m *Model) searchKey(msg tea.KeyMsg, chord string) (tea.Model, tea.Cmd) {
	switch {
	case chord == "enter":
		m.mode = modeNormal // accept: leave cursor on the current match
		m.status = ""
	case chord == "ctrl+s":
		m.nextMatch()
	case chord == "ctrl+r":
		m.prevMatch()
	case chord == "ctrl+t":
		m.caseSensitive = !m.caseSensitive
		m.recomputeSearch()
	case msg.Type == tea.KeyBackspace:
		m.query.backspace()
		m.recomputeSearch()
	case msg.Type == tea.KeyRunes:
		m.query.insert(string(msg.Runes))
		m.recomputeSearch()
	case msg.Type == tea.KeySpace:
		m.query.insert(" ")
		m.recomputeSearch()
	}
	return m, nil
}

func (m *Model) replaceFromKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.query.String() == "" {
			m.cancelPrompt()
			return m, nil
		}
		m.mode = modeReplaceTo
		m.replaceWith.clear()
	case tea.KeyBackspace:
		m.query.backspace()
	case tea.KeyRunes:
		m.query.insert(string(msg.Runes))
	case tea.KeySpace:
		m.query.insert(" ")
	}
	return m, nil
}

func (m *Model) replaceToKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		newText, count := search.ReplaceAll(m.ed.Text(), m.query.String(), m.replaceWith.String(), m.opts())
		var cmd tea.Cmd
		if count > 0 {
			m.ed.SetText(newText)
			cmd = m.onEdit()
		}
		m.status = fmt.Sprintf("Replaced %d occurrence(s)", count)
		m.mode = modeNormal
		m.matches = nil
		m.applyLayout()
		return m, cmd
	case tea.KeyBackspace:
		m.replaceWith.backspace()
	case tea.KeyRunes:
		m.replaceWith.insert(string(msg.Runes))
	case tea.KeySpace:
		m.replaceWith.insert(" ")
	}
	return m, nil
}

// --- search lifecycle ------------------------------------------------------

func (m *Model) startSearch() {
	m.mode = modeSearch
	m.query.clear()
	m.matches = nil
	m.matchIdx = 0
	m.searchOrigin = m.ed.CursorPos()
	m.status = ""
}

func (m *Model) startReplace() {
	m.mode = modeReplaceFrom
	m.query.clear()
	m.replaceWith.clear()
	m.searchOrigin = m.ed.CursorPos()
	m.status = ""
}

// recomputeSearch refreshes matches for the current query and jumps the cursor
// to the first match at or after the search origin.
func (m *Model) recomputeSearch() {
	m.matches = search.FindAll(m.ed.Text(), m.query.String(), m.opts())
	if len(m.matches) == 0 {
		m.ed.SetCursor(m.searchOrigin)
		m.applyLayout()
		return
	}
	m.matchIdx = 0
	for i, mt := range m.matches {
		if mt.Start >= m.searchOrigin {
			m.matchIdx = i
			break
		}
	}
	m.gotoCurrentMatch()
}

func (m *Model) nextMatch() {
	if len(m.matches) == 0 {
		return
	}
	m.matchIdx = (m.matchIdx + 1) % len(m.matches)
	m.gotoCurrentMatch()
}

func (m *Model) prevMatch() {
	if len(m.matches) == 0 {
		return
	}
	m.matchIdx = (m.matchIdx - 1 + len(m.matches)) % len(m.matches)
	m.gotoCurrentMatch()
}

func (m *Model) gotoCurrentMatch() {
	mt := m.matches[m.matchIdx]
	m.ed.SetCursor(mt.Start)
	m.applyLayout()
}

// cancelPrompt closes any active prompt; for search it restores the cursor to
// where the search began.
func (m *Model) cancelPrompt() {
	if m.mode == modeSearch {
		m.ed.SetCursor(m.searchOrigin)
		m.applyLayout()
	}
	m.mode = modeNormal
	m.matches = nil
	m.input.clear()
	m.status = ""
}
