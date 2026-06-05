package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"argc.dev/chiquito/internal/fuzzy"
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
// for selection, with a type-to-filter fuzzy search. Choosing a directory
// navigates into it; choosing a file emits an openFileMsg.
type filePane struct {
	dir      string
	entries  []fileEntry // all entries in dir (parent, dirs, files)
	filter   string      // fuzzy filter text typed by the user
	matches  []int       // indices into entries that match filter, ranked
	selected int         // index into matches
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

	p.filter = ""
	p.applyFilter()
}

// applyFilter recomputes the ranked list of matching entries for the current
// filter. With no filter, entries keep their natural order (parent, dirs,
// files); otherwise they are ranked by fuzzy score.
func (p *filePane) applyFilter() {
	p.matches = p.matches[:0]
	if p.filter == "" {
		for i := range p.entries {
			p.matches = append(p.matches, i)
		}
	} else {
		labels := make([]string, len(p.entries))
		for i, e := range p.entries {
			labels[i] = strings.TrimSuffix(e.label, "/")
		}
		for _, r := range fuzzy.Rank(p.filter, labels) {
			p.matches = append(p.matches, r.Index)
		}
	}
	if p.selected >= len(p.matches) {
		p.selected = len(p.matches) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
	p.top = 0
}

func (p *filePane) setFilter(s string) {
	p.filter = s
	p.applyFilter()
}

func sortEntries(es []fileEntry) {
	sort.Slice(es, func(i, j int) bool {
		return strings.ToLower(es[i].label) < strings.ToLower(es[j].label)
	})
}

// current returns the highlighted entry within the filtered list, or nil.
func (p *filePane) current() *fileEntry {
	if p.selected < 0 || p.selected >= len(p.matches) {
		return nil
	}
	return &p.entries[p.matches[p.selected]]
}

func (p *filePane) update(msg tea.KeyMsg) (paneOutcome, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlG:
		return paneClose, nil
	case tea.KeyEnter:
		return p.choose()
	case tea.KeyUp, tea.KeyCtrlP:
		p.move(-1)
	case tea.KeyDown, tea.KeyCtrlN:
		p.move(1)
	case tea.KeyPgUp:
		p.move(-(filePaneHeight - 1))
	case tea.KeyPgDown:
		p.move(filePaneHeight - 1)
	case tea.KeyHome:
		p.selected = 0
	case tea.KeyEnd:
		p.move(len(p.matches))
	case tea.KeyLeft:
		p.load(filepath.Dir(p.dir)) // go to parent
	case tea.KeyRight:
		if e := p.current(); e != nil && e.isDir {
			p.load(e.path)
		}
	case tea.KeyBackspace:
		// Backspace edits the filter; when it is empty, it goes to the parent.
		if p.filter != "" {
			p.setFilter(trimLastRune(p.filter))
		} else {
			p.load(filepath.Dir(p.dir))
		}
	case tea.KeySpace:
		p.setFilter(p.filter + " ")
	case tea.KeyRunes:
		p.setFilter(p.filter + string(msg.Runes))
	}
	return paneStay, nil
}

func (p *filePane) move(delta int) {
	p.selected += delta
	if p.selected < 0 {
		p.selected = 0
	}
	if p.selected >= len(p.matches) {
		p.selected = len(p.matches) - 1
	}
}

func trimLastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
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
	header := " Open file — " + p.dir
	if p.filter != "" {
		header += "  filter: " + p.filter
	}
	if len(p.matches) == 0 {
		header += "  (no matches)"
	}
	if p.note != "" {
		header = " " + p.note
	}
	b.WriteString(paneHeaderStyle.Render(fitWidth(header, width)))
	b.WriteByte('\n')

	for i := 0; i < rows; i++ {
		mi := p.top + i
		switch {
		case mi >= len(p.matches):
			b.WriteString(fitWidth("", width))
		default:
			e := p.entries[p.matches[mi]]
			switch {
			case mi == p.selected:
				b.WriteString(paneSelStyle.Render(fitWidth("> "+e.label, width)))
			case e.isDir:
				b.WriteString(paneDirStyle.Render(fitWidth("  "+e.label, width)))
			default:
				b.WriteString(fitWidth("  "+e.label, width))
			}
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
