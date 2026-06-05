package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
)

// makeTree builds a temp directory with a subdirectory and two files.
func makeTree(t *testing.T) (dir, sub, file string) {
	t.Helper()
	dir = t.TempDir()
	sub = filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	file = filepath.Join(dir, "alpha.txt")
	if err := os.WriteFile(file, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, sub, file
}

func TestFilePaneListsDirectory(t *testing.T) {
	dir, _, _ := makeTree(t)
	p := newFilePane(dir)

	// Parent first, then directories, then files (each sorted).
	want := []string{"../", "sub/", "alpha.txt", "beta.go"}
	if len(p.entries) != len(want) {
		t.Fatalf("entries = %d, want %d (%v)", len(p.entries), len(want), p.entries)
	}
	for i, w := range want {
		if p.entries[i].label != w {
			t.Errorf("entry[%d] = %q, want %q", i, p.entries[i].label, w)
		}
	}
	if !p.entries[1].isDir {
		t.Error("sub/ should be marked as a directory")
	}
}

func TestFilePaneNavigateIntoAndParent(t *testing.T) {
	dir, sub, _ := makeTree(t)
	p := newFilePane(dir)

	// Move to "sub/" (index 1) and enter it.
	p.move(1)
	outcome, cmd := p.update(tea.KeyMsg{Type: tea.KeyEnter})
	if outcome != paneStay || cmd != nil {
		t.Fatalf("entering a dir should keep the pane open, got %v", outcome)
	}
	if p.dir != sub {
		t.Errorf("dir = %q, want %q", p.dir, sub)
	}

	// Backspace goes back to the parent.
	p.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.dir != dir {
		t.Errorf("after backspace dir = %q, want %q", p.dir, dir)
	}
}

func TestFilePaneSelectFileEmitsOpen(t *testing.T) {
	dir, _, file := makeTree(t)
	p := newFilePane(dir)
	// "alpha.txt" is at index 2 (after "../" and "sub/").
	p.selected = 2
	outcome, cmd := p.update(tea.KeyMsg{Type: tea.KeyEnter})
	if outcome != paneClose {
		t.Fatalf("selecting a file should close the pane")
	}
	if cmd == nil {
		t.Fatal("expected an open command")
	}
	msg, ok := cmd().(openFileMsg)
	if !ok {
		t.Fatalf("command produced %T, want openFileMsg", cmd())
	}
	if msg.path != file {
		t.Errorf("openFileMsg path = %q, want %q", msg.path, file)
	}
}

func TestFilePaneViewLineCount(t *testing.T) {
	dir, _, _ := makeTree(t)
	p := newFilePane(dir)
	out := p.view(200, filePaneHeight)
	if got := strings.Count(out, "\n") + 1; got != filePaneHeight {
		t.Errorf("pane rendered %d lines, want %d", got, filePaneHeight)
	}
	// Header should name the directory (wide enough to avoid truncation).
	if !strings.Contains(out, dir) {
		t.Errorf("pane header missing directory name:\n%s", out)
	}
	// Each rendered line is exactly the requested width.
	for _, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w != 200 {
			t.Errorf("pane line width = %d, want 200: %q", w, line)
		}
	}
}

func TestOpenWithPaneEnabled(t *testing.T) {
	dir, _, file := makeTree(t)
	m := New(editor.New([]byte("scratch"), filepath.Join(dir, "ctx.txt")), config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	m = updated.(*Model)

	// C-x C-f opens the pane (FilePane is on by default).
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	if m.activePane == nil {
		t.Fatal("expected a file pane to open")
	}
	// Opening a pane shrinks the editor viewport.
	if m.textHeight() != 24-1-filePaneHeight {
		t.Errorf("textHeight = %d, want %d", m.textHeight(), 24-1-filePaneHeight)
	}

	// Select "alpha.txt" (../, sub/, alpha.txt) and open it.
	fp := m.activePane.(*filePane)
	fp.selected = 2
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	if m.activePane != nil {
		t.Error("pane should close after selecting a file")
	}
	if cmd == nil {
		t.Fatal("expected an open command")
	}
	// Feed the emitted openFileMsg back in, as the runtime would.
	updated, _ = m.Update(cmd())
	m = updated.(*Model)

	if m.ed.Name() != file {
		t.Errorf("opened %q, want %q", m.ed.Name(), file)
	}
	if string(m.ed.Bytes()) != "hello\n" {
		t.Errorf("content = %q", m.ed.Bytes())
	}
}

func TestOpenPaneCancel(t *testing.T) {
	dir, _, _ := makeTree(t)
	m := New(editor.New([]byte("keep"), filepath.Join(dir, "ctx.txt")), config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	m = updated.(*Model)
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	m, _ = send(m, key(tea.KeyEsc))
	if m.activePane != nil {
		t.Error("Esc should close the pane")
	}
	if string(m.ed.Bytes()) != "keep" {
		t.Error("cancel must not change the document")
	}
}

func TestOpenWithPaneDisabledUsesPrompt(t *testing.T) {
	m := newModel(t, "x", "f.txt")
	m.cfg.Features.FilePane = false
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	if m.activePane != nil {
		t.Error("pane should not open when FilePane is disabled")
	}
	if m.mode != modeOpen {
		t.Errorf("mode = %v, want modeOpen (text prompt)", m.mode)
	}
}
