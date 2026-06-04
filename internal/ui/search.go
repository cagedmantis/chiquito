package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/search"
)

// handleMinibuffer routes keys while a search or replace prompt is active.
func (m *Model) handleMinibuffer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	chord := msg.String()

	// Cancel from any prompt.
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
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
			m.recomputeSearch()
		}
	case msg.Type == tea.KeyRunes:
		m.query += string(msg.Runes)
		m.recomputeSearch()
	case msg.Type == tea.KeySpace:
		m.query += " "
		m.recomputeSearch()
	}
	return m, nil
}

func (m *Model) replaceFromKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.query == "" {
			m.cancelPrompt()
			return m, nil
		}
		m.mode = modeReplaceTo
		m.replaceWith = ""
	case tea.KeyBackspace:
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
	case tea.KeyRunes:
		m.query += string(msg.Runes)
	case tea.KeySpace:
		m.query += " "
	}
	return m, nil
}

func (m *Model) replaceToKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		newText, count := search.ReplaceAll(m.ed.Text(), m.query, m.replaceWith, m.opts())
		if count > 0 {
			m.ed.SetText(newText)
			m.markEdited()
		}
		m.status = fmt.Sprintf("Replaced %d occurrence(s)", count)
		m.mode = modeNormal
		m.matches = nil
		m.applyLayout()
	case tea.KeyBackspace:
		if r := []rune(m.replaceWith); len(r) > 0 {
			m.replaceWith = string(r[:len(r)-1])
		}
	case tea.KeyRunes:
		m.replaceWith += string(msg.Runes)
	case tea.KeySpace:
		m.replaceWith += " "
	}
	return m, nil
}

// --- search lifecycle ------------------------------------------------------

func (m *Model) startSearch() {
	m.mode = modeSearch
	m.query = ""
	m.matches = nil
	m.matchIdx = 0
	m.searchOrigin = m.ed.CursorPos()
	m.status = ""
}

func (m *Model) startReplace() {
	m.mode = modeReplaceFrom
	m.query = ""
	m.replaceWith = ""
	m.searchOrigin = m.ed.CursorPos()
	m.status = ""
}

// recomputeSearch refreshes matches for the current query and jumps the cursor
// to the first match at or after the search origin.
func (m *Model) recomputeSearch() {
	m.matches = search.FindAll(m.ed.Text(), m.query, m.opts())
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

func (m *Model) cancelPrompt() {
	m.ed.SetCursor(m.searchOrigin)
	m.mode = modeNormal
	m.matches = nil
	m.status = ""
	m.applyLayout()
}
