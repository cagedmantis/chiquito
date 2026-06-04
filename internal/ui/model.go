// Package ui is the Bubble Tea adapter over the framework-agnostic editor core.
// All charmbracelet imports are confined to this package; the core packages
// (editor, search, syntax, spell) know nothing about the terminal. The Model
// translates key messages into editor commands and renders the editor state —
// with syntax highlighting, search, and asynchronous spell checking — to a
// string. Slow work (dictionary loading, spell checking, config polling) runs in
// commands off the Update thread and reports back via messages.
package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/fileio"
	"argc.dev/chiquito/internal/search"
	"argc.dev/chiquito/internal/spell"
	"argc.dev/chiquito/internal/syntax"
)

// mode is the model's input mode: normal editing or an active minibuffer prompt.
type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeReplaceFrom // entering the search term for a replace
	modeReplaceTo   // entering the replacement text
	modeOpen        // entering a path to open
	modeSaveAs      // entering a path to save to
)

const (
	spellDebounce      = 250 * time.Millisecond
	configPollInterval = 1500 * time.Millisecond
)

// Model is the Bubble Tea model. It is used via a pointer so command handlers
// can mutate it in place; the embedded *editor.Editor carries the document.
type Model struct {
	ed  *editor.Editor
	km  keymap
	cfg config.Config

	width, height int
	hscroll       int
	tabWidth      int
	lineNumbers   bool

	pending     string
	status      string
	confirmQuit bool
	quitting    bool

	// minibuffer / search / replace state
	mode          mode
	input         lineInput // generic prompt text (open, save-as)
	query         lineInput // current search term
	replaceWith   lineInput // replacement text
	caseSensitive bool
	matches       []search.Match
	matchIdx      int
	searchOrigin  int

	// syntax highlighting state
	lang        syntax.Language
	theme       theme
	enterStates []syntax.State
	synStale    bool

	// spell checking state
	checker    spell.Dictionary
	spellSpans []spell.Misspelling
	docVersion int // bumped on every edit; guards stale async results

	// config hot-reload state
	configMod time.Time
}

// New constructs a Model for the given editor and configuration.
func New(ed *editor.Editor, cfg config.Config) *Model {
	ed.SetTabStops(cfg.Editor.TabWidth, cfg.Editor.ExpandTabs)
	mt, _ := config.ModTime()
	return &Model{
		ed:          ed,
		km:          newKeymap(cfg),
		cfg:         cfg,
		tabWidth:    cfg.Editor.TabWidth,
		lineNumbers: cfg.Editor.LineNumbers,
		width:       80,
		height:      24,
		lang:        syntax.ForFilename(ed.Name()),
		theme:       themeByName(cfg.Theme.Name),
		synStale:    true,
		configMod:   mt,
	}
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applyLayout()
		return m, nil
	case tea.KeyMsg:
		if m.mode != modeNormal {
			return m.handleMinibuffer(msg)
		}
		return m.handleKey(msg)
	case dictLoadedMsg:
		m.checker = msg.dict
		return m, m.runSpellCmd(m.docVersion)
	case spellTickMsg:
		if msg.version == m.docVersion && m.checker != nil {
			return m, m.runSpellCmd(m.docVersion)
		}
		return m, nil
	case spellResultMsg:
		if msg.version == m.docVersion {
			m.spellSpans = msg.spans
		}
		return m, nil
	case configTickMsg:
		return m, m.maybeReloadConfig()
	}
	return m, nil
}

func (m *Model) applyLayout() {
	m.ed.SetSize(m.textWidth(), m.textHeight())
	m.syncHScroll()
}

// onEdit records a document change: it invalidates syntax state, bumps the doc
// version (so in-flight async results are discarded), and returns a debounced
// spell-check command.
func (m *Model) onEdit() tea.Cmd {
	m.synStale = true
	m.docVersion++
	return m.scheduleSpell()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	chord := msg.String()

	if m.pending != "" {
		combo := m.pending + " " + chord
		m.pending = ""
		m.status = ""
		if action, ok := m.km.lookup(combo); ok {
			return m.dispatch(action)
		}
		m.status = "Unknown sequence: " + combo
		return m, nil
	}

	if m.km.isPrefix(chord) {
		m.pending = chord
		m.status = chord + "-"
		return m, nil
	}

	if action, ok := m.km.lookup(chord); ok {
		return m.dispatch(action)
	}

	return m.handleInput(msg)
}

// dispatch executes a logical editor action.
func (m *Model) dispatch(action string) (tea.Model, tea.Cmd) {
	if action != "quit" {
		m.confirmQuit = false
	}
	m.status = ""
	var cmd tea.Cmd

	switch action {
	case "cursor-forward":
		m.ed.MoveRight()
	case "cursor-backward":
		m.ed.MoveLeft()
	case "cursor-up":
		m.ed.MoveUp()
	case "cursor-down":
		m.ed.MoveDown()
	case "line-start":
		m.ed.LineStart()
	case "line-end":
		m.ed.LineEnd()
	case "delete-forward":
		m.ed.DeleteForward()
		cmd = m.onEdit()
	case "kill-line":
		m.ed.KillLine()
		cmd = m.onEdit()
	case "search":
		m.startSearch()
	case "replace":
		m.startReplace()
	case "open":
		m.startOpen()
	case "save":
		return m.save()
	case "quit":
		return m.quit()
	default:
		m.status = "unbound action: " + action
	}
	m.applyLayout()
	return m, cmd
}

// handleInput processes built-in keys: navigation and text entry.
func (m *Model) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.confirmQuit = false
	m.status = ""
	edited := false

	switch msg.Type {
	case tea.KeyRunes:
		m.ed.Insert(string(msg.Runes))
		edited = true
	case tea.KeySpace:
		m.ed.InsertRune(' ')
		edited = true
	case tea.KeyEnter:
		m.ed.InsertNewline()
		edited = true
	case tea.KeyTab:
		m.ed.InsertRune('\t')
		edited = true
	case tea.KeyBackspace:
		m.ed.DeleteBackward()
		edited = true
	case tea.KeyDelete:
		m.ed.DeleteForward()
		edited = true
	case tea.KeyLeft:
		m.ed.MoveLeft()
	case tea.KeyRight:
		m.ed.MoveRight()
	case tea.KeyUp:
		m.ed.MoveUp()
	case tea.KeyDown:
		m.ed.MoveDown()
	case tea.KeyHome:
		m.ed.LineStart()
	case tea.KeyEnd:
		m.ed.LineEnd()
	case tea.KeyPgUp:
		for i := 0; i < m.textHeight(); i++ {
			m.ed.MoveUp()
		}
	case tea.KeyPgDown:
		for i := 0; i < m.textHeight(); i++ {
			m.ed.MoveDown()
		}
	default:
	}

	var cmd tea.Cmd
	if edited {
		cmd = m.onEdit()
	}
	m.applyLayout()
	return m, cmd
}

// save writes the current file, or starts a save-as prompt for a scratch buffer.
func (m *Model) save() (tea.Model, tea.Cmd) {
	if m.ed.Name() == "" {
		m.startSaveAs()
		return m, nil
	}
	if err := fileio.WriteAtomic(m.ed.Name(), m.ed.Bytes()); err != nil {
		m.status = "Save failed: " + err.Error()
		return m, nil
	}
	m.ed.MarkSaved()
	m.status = fmt.Sprintf("Wrote %s", m.ed.Name())
	return m, nil
}

func (m *Model) quit() (tea.Model, tea.Cmd) {
	if m.ed.Dirty() && !m.confirmQuit {
		m.confirmQuit = true
		m.status = "Unsaved changes — press quit again to discard"
		return m, nil
	}
	m.quitting = true
	return m, tea.Quit
}

func (m *Model) opts() search.Options {
	return search.Options{CaseSensitive: m.caseSensitive}
}
