package ui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/fileio"
	"argc.dev/chiquito/internal/syntax"
)

// lineInput is a minimal single-line text field used by the minibuffer prompts
// (search term, replacement, open path, save-as path). It edits a rune slice so
// multi-byte characters behave correctly.
type lineInput struct {
	value []rune
}

func (l *lineInput) String() string { return string(l.value) }

func (l *lineInput) insert(s string) { l.value = append(l.value, []rune(s)...) }

func (l *lineInput) backspace() {
	if len(l.value) > 0 {
		l.value = l.value[:len(l.value)-1]
	}
}

func (l *lineInput) clear() { l.value = l.value[:0] }

// promptKey edits the generic prompt input (open / save-as) and completes on
// Enter.
func (m *Model) promptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.promptAccept()
	case tea.KeyBackspace:
		m.input.backspace()
	case tea.KeyRunes:
		m.input.insert(string(msg.Runes))
	case tea.KeySpace:
		m.input.insert(" ")
	}
	return m, nil
}

func (m *Model) promptAccept() (tea.Model, tea.Cmd) {
	value := m.input.String()
	switch m.mode {
	case modeOpen:
		return m.doOpen(value)
	case modeSaveAs:
		return m.doSaveAs(value)
	}
	m.mode = modeNormal
	return m, nil
}

func (m *Model) startOpen() {
	m.mode = modeOpen
	m.input.clear()
	m.status = ""
}

func (m *Model) startSaveAs() {
	m.mode = modeSaveAs
	m.input.clear()
	m.status = "Save as:"
}

// doOpen loads a file into a fresh editor, re-selecting the syntax language. A
// not-yet-existing path opens an empty buffer bound to that name.
func (m *Model) doOpen(path string) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	if path == "" {
		return m, nil
	}
	data, err := fileio.Read(path)
	if err != nil && !os.IsNotExist(err) {
		m.status = "Open failed: " + err.Error()
		return m, nil
	}

	m.ed = editor.New(data, path)
	m.ed.SetTabStops(m.cfg.Editor.TabWidth, m.cfg.Editor.ExpandTabs)
	m.lang = syntax.ForFilename(path)
	m.resetDocState()
	m.status = fmt.Sprintf("Opened %s", path)
	m.applyLayout()
	return m, m.onEdit() // re-tokenize + spell-check the new document
}

// doSaveAs binds the buffer to a new path and writes it.
func (m *Model) doSaveAs(path string) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	if path == "" {
		m.status = "Save cancelled (no name)"
		return m, nil
	}
	m.ed.SetName(path)
	m.lang = syntax.ForFilename(path)
	m.synStale = true
	if err := fileio.WriteAtomic(path, m.ed.Bytes()); err != nil {
		m.status = "Save failed: " + err.Error()
		return m, nil
	}
	m.ed.MarkSaved()
	m.status = fmt.Sprintf("Wrote %s", path)
	return m, nil
}

// resetDocState clears per-document derived state after the buffer is replaced.
func (m *Model) resetDocState() {
	m.synStale = true
	m.spellSpans = nil
	m.matches = nil
	m.hscroll = 0
	m.docVersion++
}
