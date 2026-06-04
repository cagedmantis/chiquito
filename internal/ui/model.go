// Package ui is the Bubble Tea adapter over the framework-agnostic editor core.
// All charmbracelet imports are confined to this package; internal/editor knows
// nothing about the terminal. The Model translates key messages into editor
// commands and renders the editor state, with syntax highlighting and search,
// to a string.
package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/fileio"
	"argc.dev/chiquito/internal/search"
	"argc.dev/chiquito/internal/syntax"
)

// mode is the model's input mode: normal editing or an active minibuffer prompt.
type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeReplaceFrom // entering the search term for a replace
	modeReplaceTo   // entering the replacement text
)

// Model is the Bubble Tea model. It is used via a pointer so command handlers
// can mutate it in place; the embedded *editor.Editor carries the document.
type Model struct {
	ed  *editor.Editor
	km  keymap
	cfg config.Config

	width, height int // terminal size in cells
	hscroll       int // horizontal scroll offset in display columns
	tabWidth      int
	lineNumbers   bool

	pending     string // first key of an in-progress multi-key sequence
	status      string // transient message shown in the status bar
	confirmQuit bool   // armed when quit was requested with unsaved changes
	quitting    bool

	// search / replace state
	mode          mode
	query         string         // current search term
	replaceWith   string         // replacement text (replace mode)
	caseSensitive bool           // search case sensitivity
	matches       []search.Match // matches for the current query
	matchIdx      int            // index of the current match
	searchOrigin  int            // cursor position when search began (for cancel)

	// syntax highlighting state
	lang        syntax.Language
	theme       theme
	enterStates []syntax.State // entering lexical state per line (cached)
	synStale    bool           // enterStates needs recomputation
}

// New constructs a Model for the given editor and configuration.
func New(ed *editor.Editor, cfg config.Config) *Model {
	ed.SetTabStops(cfg.Editor.TabWidth, cfg.Editor.ExpandTabs)
	m := &Model{
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
	}
	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

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
	}
	return m, nil
}

func (m *Model) applyLayout() {
	m.ed.SetSize(m.textWidth(), m.textHeight())
	m.syncHScroll()
}

// markEdited records that the document changed so syntax state is recomputed.
func (m *Model) markEdited() { m.synStale = true }

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
		m.markEdited()
	case "kill-line":
		m.ed.KillLine()
		m.markEdited()
	case "search":
		m.startSearch()
	case "replace":
		m.startReplace()
	case "save":
		m.save()
	case "open":
		m.status = "open: interactive prompt arrives in a later phase"
	case "quit":
		return m.quit()
	default:
		m.status = "unbound action: " + action
	}
	m.applyLayout()
	return m, nil
}

// handleInput processes built-in keys: navigation and text entry.
func (m *Model) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.confirmQuit = false
	m.status = ""

	switch msg.Type {
	case tea.KeyRunes:
		m.ed.Insert(string(msg.Runes))
		m.markEdited()
	case tea.KeySpace:
		m.ed.InsertRune(' ')
		m.markEdited()
	case tea.KeyEnter:
		m.ed.InsertNewline()
		m.markEdited()
	case tea.KeyTab:
		m.ed.InsertRune('\t')
		m.markEdited()
	case tea.KeyBackspace:
		m.ed.DeleteBackward()
		m.markEdited()
	case tea.KeyDelete:
		m.ed.DeleteForward()
		m.markEdited()
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
		// Unhandled key; ignore.
	}
	m.applyLayout()
	return m, nil
}

func (m *Model) save() {
	if m.ed.Name() == "" {
		m.status = "No file name to save to"
		return
	}
	if err := fileio.WriteAtomic(m.ed.Name(), m.ed.Bytes()); err != nil {
		m.status = "Save failed: " + err.Error()
		return
	}
	m.ed.MarkSaved()
	m.status = fmt.Sprintf("Wrote %s", m.ed.Name())
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
