// Package ui is the Bubble Tea adapter over the framework-agnostic editor core.
// All charmbracelet imports are confined to this package; internal/editor knows
// nothing about the terminal. The Model translates key messages into editor
// commands and renders the editor state to a string.
package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/fileio"
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
}

// New constructs a Model for the given editor and configuration.
func New(ed *editor.Editor, cfg config.Config) *Model {
	ed.SetTabStops(cfg.Editor.TabWidth, cfg.Editor.ExpandTabs)
	return &Model{
		ed:          ed,
		km:          newKeymap(cfg),
		cfg:         cfg,
		tabWidth:    cfg.Editor.TabWidth,
		lineNumbers: cfg.Editor.LineNumbers,
		width:       80,
		height:      24,
	}
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
		return m.handleKey(msg)
	}
	return m, nil
}

// applyLayout recomputes the editor's text viewport from the terminal size and
// the current gutter width (which depends on the line count), then keeps the
// cursor visible both vertically and horizontally.
func (m *Model) applyLayout() {
	m.ed.SetSize(m.textWidth(), m.textHeight())
	m.syncHScroll()
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	chord := msg.String()

	// Complete an in-progress multi-key sequence (e.g. C-x C-s).
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

	// Begin a multi-key sequence.
	if m.km.isPrefix(chord) {
		m.pending = chord
		m.status = chord + "-"
		return m, nil
	}

	// A configured single-chord action.
	if action, ok := m.km.lookup(chord); ok {
		return m.dispatch(action)
	}

	// Otherwise: built-in navigation/editing keys and text input.
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
	case "kill-line":
		m.ed.KillLine()
	case "save":
		m.save()
	case "open":
		m.status = "open: interactive prompt arrives in a later phase"
	case "search", "replace":
		m.status = action + ": arrives in phase 3"
	case "quit":
		return m.quit()
	default:
		m.status = "unbound action: " + action
	}
	m.applyLayout()
	return m, nil
}

// handleInput processes built-in keys: arrows/home/end/page navigation and text
// entry. Configured Emacs chords are handled in dispatch; this covers keys a
// user expects regardless of the keymap.
func (m *Model) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.confirmQuit = false
	m.status = ""

	switch msg.Type {
	case tea.KeyRunes:
		m.ed.Insert(string(msg.Runes))
	case tea.KeySpace:
		m.ed.InsertRune(' ')
	case tea.KeyEnter:
		m.ed.InsertNewline()
	case tea.KeyTab:
		m.ed.InsertRune('\t')
	case tea.KeyBackspace:
		m.ed.DeleteBackward()
	case tea.KeyDelete:
		m.ed.DeleteForward()
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
