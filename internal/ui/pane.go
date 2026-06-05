package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// A pane is a bottom-anchored, selectable mini-window. A command (the first
// being "open file") may present its options here instead of as a plain text
// prompt; future commands implement this same interface to get a list pane.
type pane interface {
	// update handles a key, returning whether the pane should stay open and an
	// optional command (e.g. to act on a selection).
	update(msg tea.KeyMsg) (paneOutcome, tea.Cmd)
	// view renders exactly height lines, each width columns wide.
	view(width, height int) string
	// preferredHeight is the number of rows the pane would like to occupy.
	preferredHeight() int
}

type paneOutcome int

const (
	paneStay  paneOutcome = iota // keep the pane open
	paneClose                    // remove the pane
)

const filePaneHeight = 10

var (
	paneHeaderStyle = lipgloss.NewStyle().Reverse(true).Bold(true)
	paneSelStyle    = lipgloss.NewStyle().Reverse(true)
	paneDirStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
)

// openFileMsg is emitted by the file pane when the user selects a file to open.
type openFileMsg struct{ path string }

type fileEntry struct {
	label string // display text (directories carry a trailing "/")
	path  string // absolute path the entry refers to
	isDir bool
}

// filePane browses a directory: it lists the parent, sub-directories, and files
// for selection. Choosing a directory navigates into it; choosing a file emits
// an openFileMsg.
type filePane struct {
	dir      string
	entries  []fileEntry
	selected int
	top      int
	note     string // error/status shown in the header when set
}

func newFilePane(dir string) *filePane {
	p := &filePane{}
	p.load(dir)
	return p
}

func (p *filePane) preferredHeight() int { return filePaneHeight }

// load reads dir and rebuilds the entry list: ".." first, then directories,
// then files, each group sorted case-insensitively.
func (p *filePane) load(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	p.dir = filepath.Clean(abs)
	p.selected = 0
	p.top = 0
	p.note = ""

	entries := []fileEntry{{label: "../", path: filepath.Dir(p.dir), isDir: true}}
	des, err := os.ReadDir(p.dir)
	if err != nil {
		p.note = "cannot read: " + err.Error()
		p.entries = entries
		return
	}

	var dirs, files []fileEntry
	for _, de := range des {
		name := de.Name()
		full := filepath.Join(p.dir, name)
		// Resolve symlinked directories so they are browsable too.
		isDir := de.IsDir()
		if de.Type()&os.ModeSymlink != 0 {
			if info, serr := os.Stat(full); serr == nil && info.IsDir() {
				isDir = true
			}
		}
		if isDir {
			dirs = append(dirs, fileEntry{label: name + "/", path: full, isDir: true})
		} else {
			files = append(files, fileEntry{label: name, path: full, isDir: false})
		}
	}
	sortEntries(dirs)
	sortEntries(files)
	p.entries = append(append(entries, dirs...), files...)
}

func sortEntries(es []fileEntry) {
	sort.Slice(es, func(i, j int) bool {
		return strings.ToLower(es[i].label) < strings.ToLower(es[j].label)
	})
}

func (p *filePane) current() *fileEntry {
	if p.selected < 0 || p.selected >= len(p.entries) {
		return nil
	}
	return &p.entries[p.selected]
}

func (p *filePane) update(msg tea.KeyMsg) (paneOutcome, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+g":
		return paneClose, nil
	case "up", "ctrl+p":
		p.move(-1)
	case "down", "ctrl+n":
		p.move(1)
	case "pgup":
		p.move(-(filePaneHeight - 1))
	case "pgdown":
		p.move(filePaneHeight - 1)
	case "home":
		p.selected = 0
	case "end":
		p.selected = len(p.entries) - 1
	case "backspace", "left":
		p.load(filepath.Dir(p.dir)) // go to parent
	case "right":
		if e := p.current(); e != nil && e.isDir {
			p.load(e.path)
		}
	case "enter":
		return p.choose()
	}
	return paneStay, nil
}

func (p *filePane) move(delta int) {
	p.selected += delta
	if p.selected < 0 {
		p.selected = 0
	}
	if p.selected >= len(p.entries) {
		p.selected = len(p.entries) - 1
	}
}

// choose acts on the highlighted entry: directories navigate, files are opened.
func (p *filePane) choose() (paneOutcome, tea.Cmd) {
	e := p.current()
	if e == nil {
		return paneClose, nil
	}
	if e.isDir {
		p.load(e.path)
		return paneStay, nil
	}
	path := e.path
	return paneClose, func() tea.Msg { return openFileMsg{path: path} }
}

func (p *filePane) view(width, height int) string {
	if height < 2 {
		height = 2
	}
	rows := height - 1 // first line is the header

	// Keep the selection within the visible window.
	if p.selected < p.top {
		p.top = p.selected
	}
	if p.selected >= p.top+rows {
		p.top = p.selected - rows + 1
	}
	if p.top < 0 {
		p.top = 0
	}

	var b strings.Builder
	header := " Open file — " + p.dir + "  (↑↓ select · ↵ open/enter dir · ⌫ up · Esc cancel)"
	if p.note != "" {
		header = " " + p.note
	}
	b.WriteString(paneHeaderStyle.Render(fitWidth(header, width)))
	b.WriteByte('\n')

	for i := 0; i < rows; i++ {
		idx := p.top + i
		switch {
		case idx >= len(p.entries):
			b.WriteString(fitWidth("", width))
		case idx == p.selected:
			b.WriteString(paneSelStyle.Render(fitWidth("> "+p.entries[idx].label, width)))
		case p.entries[idx].isDir:
			b.WriteString(paneDirStyle.Render(fitWidth("  "+p.entries[idx].label, width)))
		default:
			b.WriteString(fitWidth("  "+p.entries[idx].label, width))
		}
		if i < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// fitWidth pads s with spaces or truncates it to exactly width display columns.
func fitWidth(s string, width int) string {
	if width < 0 {
		width = 0
	}
	w := runewidth.StringWidth(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return runewidth.Truncate(s, width, "…")
}
