package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/fileio"
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

func (l *lineInput) set(s string) { l.value = append(l.value[:0], []rune(s)...) }

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
	m.status = ""
	// When enabled, present a selectable file-browser pane instead of a plain
	// text path prompt.
	if m.cfg.Features.FilePane {
		m.activePane = newFilePane(m.startDirBare())
		m.applyLayout()
		return
	}
	m.mode = modeOpen
	m.input.set(m.startDir())
}

func (m *Model) startSaveAs() {
	m.mode = modeSaveAs
	m.input.set(m.startDir())
	m.status = "Save as:"
}

// startDirBare returns the directory a file command should start in, without a
// trailing separator. It prefers the directory of the current file, then the
// working directory (where chiquito was invoked), then the home directory.
func (m *Model) startDirBare() string {
	if name := m.ed.Name(); name != "" {
		if dir := filepath.Dir(name); dir != "" && dir != "." {
			return dir
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

// startDir is startDirBare with a trailing separator, for prefilling a text
// prompt so the user can type just a filename.
func (m *Model) startDir() string { return withSeparator(m.startDirBare()) }

func withSeparator(dir string) string {
	sep := string(filepath.Separator)
	if dir == "" || dir == sep {
		return dir
	}
	if dir[len(dir)-1] != filepath.Separator {
		return dir + sep
	}
	return dir
}

func endsWithSeparator(p string) bool {
	return p != "" && p[len(p)-1] == filepath.Separator
}

// resolveOpenPath cleans a path typed into a prefilled prompt. Following Emacs
// minibuffer convention, an embedded "//" resets to the filesystem root and a
// "/~" resets to the home directory, so a user can type an absolute or home path
// without first erasing the prefilled directory. A leading "~" is expanded.
func resolveOpenPath(s string) string {
	if i := strings.LastIndex(s, "//"); i >= 0 {
		s = s[i+1:] // keep the leading separator
	}
	if i := strings.LastIndex(s, "/~"); i >= 0 {
		s = s[i+1:] // now begins with "~"
	}
	if strings.HasPrefix(s, "~") && (len(s) == 1 || s[1] == filepath.Separator) {
		if home, err := os.UserHomeDir(); err == nil {
			s = home + s[1:]
		}
	}
	return s
}

// doOpen handles the text-prompt open path: it cleans the entered path, then
// loads it.
func (m *Model) doOpen(path string) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	path = resolveOpenPath(path)
	if path == "" {
		return m, nil
	}
	if endsWithSeparator(path) {
		m.status = "Not a file: " + path
		return m, nil
	}
	cmd := m.loadFile(path)
	m.applyLayout()
	return m, cmd
}

// loadFile replaces the editor with the contents of path, re-selecting the
// syntax language. A not-yet-existing path opens an empty buffer bound to that
// name; a hard read error leaves the current buffer untouched. It returns a
// command to re-tokenize and spell-check the new document.
func (m *Model) loadFile(path string) tea.Cmd {
	data, err := fileio.Read(path)
	if err != nil && !os.IsNotExist(err) {
		m.status = "Open failed: " + err.Error()
		return nil
	}
	m.ed = editor.New(data, path)
	m.ed.SetTabStops(m.cfg.Editor.TabWidth, m.cfg.Editor.ExpandTabs)
	m.hl = newHighlighter(path, m.cfg.Theme.Name, m.theme)
	m.resetDocState()
	m.status = fmt.Sprintf("Opened %s", path)
	return m.onEdit()
}

// doSaveAs binds the buffer to a new path and writes it.
func (m *Model) doSaveAs(path string) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	path = resolveOpenPath(path)
	if path == "" || endsWithSeparator(path) {
		m.status = "Save cancelled (no file name)"
		return m, nil
	}
	m.ed.SetName(path)
	m.hl = newHighlighter(path, m.cfg.Theme.Name, m.theme)
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
