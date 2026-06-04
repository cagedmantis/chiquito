package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
)

func newModel(t *testing.T, content, name string) *Model {
	t.Helper()
	m := New(editor.New([]byte(content), name), config.Default())
	// Give it a concrete size, as Bubble Tea would via WindowSizeMsg.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	return updated.(*Model)
}

// send feeds a key message and returns the (possibly updated) model plus cmd.
func send(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(*Model), cmd
}

func runes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func key(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func ctrl(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func TestTypingInserts(t *testing.T) {
	m := newModel(t, "", "")
	for _, r := range "hello" {
		m, _ = send(m, runes(string(r)))
	}
	if got := string(m.ed.Bytes()); got != "hello" {
		t.Fatalf("buffer = %q, want hello", got)
	}
	if !m.ed.Dirty() {
		t.Error("expected dirty after typing")
	}
}

func TestEnterAndBackspace(t *testing.T) {
	m := newModel(t, "", "")
	for _, r := range "ab" {
		m, _ = send(m, runes(string(r)))
	}
	m, _ = send(m, key(tea.KeyEnter))
	m, _ = send(m, runes("c"))
	if got := string(m.ed.Bytes()); got != "ab\nc" {
		t.Fatalf("buffer = %q, want %q", got, "ab\nc")
	}
	m, _ = send(m, key(tea.KeyBackspace))
	m, _ = send(m, key(tea.KeyBackspace)) // removes the newline
	if got := string(m.ed.Bytes()); got != "ab" {
		t.Fatalf("buffer = %q, want ab", got)
	}
}

func TestEmacsNavigation(t *testing.T) {
	m := newModel(t, "hello world", "")
	// C-e to end of line.
	m, _ = send(m, ctrl(tea.KeyCtrlE))
	if _, c := m.ed.CursorLineCol(); c != 11 {
		t.Fatalf("after C-e col = %d, want 11", c)
	}
	// C-a back to start.
	m, _ = send(m, ctrl(tea.KeyCtrlA))
	if _, c := m.ed.CursorLineCol(); c != 0 {
		t.Fatalf("after C-a col = %d, want 0", c)
	}
	// C-f, C-f forward two.
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	if _, c := m.ed.CursorLineCol(); c != 2 {
		t.Fatalf("after 2x C-f col = %d, want 2", c)
	}
	// C-b back one.
	m, _ = send(m, ctrl(tea.KeyCtrlB))
	if _, c := m.ed.CursorLineCol(); c != 1 {
		t.Fatalf("after C-b col = %d, want 1", c)
	}
}

func TestEmacsVerticalNavigation(t *testing.T) {
	m := newModel(t, "one\ntwo\nthree", "")
	m, _ = send(m, ctrl(tea.KeyCtrlN)) // down
	if l, _ := m.ed.CursorLineCol(); l != 1 {
		t.Fatalf("after C-n line = %d, want 1", l)
	}
	m, _ = send(m, ctrl(tea.KeyCtrlP)) // up
	if l, _ := m.ed.CursorLineCol(); l != 0 {
		t.Fatalf("after C-p line = %d, want 0", l)
	}
}

func TestKillLine(t *testing.T) {
	m := newModel(t, "hello world", "")
	for i := 0; i < 5; i++ {
		m, _ = send(m, ctrl(tea.KeyCtrlF))
	}
	m, _ = send(m, ctrl(tea.KeyCtrlK))
	if got := string(m.ed.Bytes()); got != "hello" {
		t.Fatalf("after C-k buffer = %q, want hello", got)
	}
}

func TestSaveSequence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	m := newModel(t, "", path)
	for _, r := range "saved!" {
		m, _ = send(m, runes(string(r)))
	}
	// C-x C-s
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	if m.pending != "ctrl+x" {
		t.Fatalf("pending = %q, want ctrl+x", m.pending)
	}
	m, _ = send(m, ctrl(tea.KeyCtrlS))
	if m.pending != "" {
		t.Errorf("pending not cleared: %q", m.pending)
	}
	if m.ed.Dirty() {
		t.Error("editor should be clean after save")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "saved!" {
		t.Fatalf("file contents = %q, err = %v", got, err)
	}
}

func TestQuitConfirmsWhenDirty(t *testing.T) {
	m := newModel(t, "", "x.txt")
	m, _ = send(m, runes("z")) // make it dirty
	// First C-x C-c: should arm confirmation, not quit.
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, cmd := send(m, ctrl(tea.KeyCtrlC))
	if cmd != nil {
		t.Fatal("first quit on dirty buffer should not return a command")
	}
	if !m.confirmQuit {
		t.Error("expected confirmQuit armed")
	}
	// Second C-x C-c: should quit.
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, cmd = send(m, ctrl(tea.KeyCtrlC))
	if cmd == nil {
		t.Fatal("second quit should return tea.Quit command")
	}
	if !m.quitting {
		t.Error("expected quitting set")
	}
}

func TestQuitImmediateWhenClean(t *testing.T) {
	m := newModel(t, "content", "x.txt")
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	_, cmd := send(m, ctrl(tea.KeyCtrlC))
	if cmd == nil {
		t.Fatal("clean buffer should quit immediately")
	}
}

func TestUnknownSequenceReported(t *testing.T) {
	m := newModel(t, "", "")
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, runes("z")) // C-x z is not bound
	if !strings.Contains(m.status, "Unknown sequence") {
		t.Errorf("status = %q, want unknown-sequence message", m.status)
	}
	// The stray keys must not have modified the buffer.
	if got := string(m.ed.Bytes()); got != "" {
		t.Errorf("buffer = %q, want empty", got)
	}
}

func TestViewRendersContentAndCursor(t *testing.T) {
	m := newModel(t, "alpha\nbeta", "f.txt")
	view := m.View()
	if !strings.Contains(view, "alpha") || !strings.Contains(view, "beta") {
		t.Errorf("view missing content:\n%s", view)
	}
	// Status bar shows the filename and a 1-based position.
	if !strings.Contains(view, "f.txt") || !strings.Contains(view, "1:1") {
		t.Errorf("view missing status info:\n%s", view)
	}
	// Line numbers are on by default.
	if !strings.Contains(view, "1") || !strings.Contains(view, "2") {
		t.Errorf("view missing line numbers:\n%s", view)
	}
}

func TestViewHandlesUnicodeWidth(t *testing.T) {
	// Wide runes must not panic the renderer or the clip logic.
	m := newModel(t, "世界🌍 test", "u.txt")
	m, _ = send(m, ctrl(tea.KeyCtrlE)) // cursor to end
	_ = m.View()                       // should not panic
}
