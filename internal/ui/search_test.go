package ui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/syntax"
)

func altRune(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Alt: true}
}

func TestIncrementalSearch(t *testing.T) {
	m := newModel(t, "foo bar foo baz", "")
	m, _ = send(m, ctrl(tea.KeyCtrlS))
	if m.mode != modeSearch {
		t.Fatalf("mode = %v, want search", m.mode)
	}
	for _, r := range "foo" {
		m, _ = send(m, runes(string(r)))
	}
	if m.ed.CursorPos() != 0 {
		t.Fatalf("cursor = %d, want 0 (first match)", m.ed.CursorPos())
	}
	if len(m.matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(m.matches))
	}
	// C-s jumps to the next match (second "foo" at rune 8).
	m, _ = send(m, ctrl(tea.KeyCtrlS))
	if m.ed.CursorPos() != 8 {
		t.Fatalf("cursor = %d, want 8 (second match)", m.ed.CursorPos())
	}
	// C-s wraps back to the first.
	m, _ = send(m, ctrl(tea.KeyCtrlS))
	if m.ed.CursorPos() != 0 {
		t.Fatalf("cursor = %d, want 0 (wrapped)", m.ed.CursorPos())
	}
	// Enter accepts and returns to normal mode, leaving the cursor put.
	m, _ = send(m, key(tea.KeyEnter))
	if m.mode != modeNormal {
		t.Fatalf("mode = %v, want normal after Enter", m.mode)
	}
}

func TestSearchCancelRestoresCursor(t *testing.T) {
	m := newModel(t, "abc foo", "")
	for i := 0; i < 7; i++ {
		m, _ = send(m, ctrl(tea.KeyCtrlF))
	}
	origin := m.ed.CursorPos()
	m, _ = send(m, ctrl(tea.KeyCtrlS))
	for _, r := range "abc" {
		m, _ = send(m, runes(string(r)))
	}
	if m.ed.CursorPos() != 0 {
		t.Fatalf("cursor during search = %d, want 0", m.ed.CursorPos())
	}
	// Esc cancels and restores the original cursor.
	m, _ = send(m, key(tea.KeyEsc))
	if m.mode != modeNormal {
		t.Errorf("mode = %v, want normal", m.mode)
	}
	if m.ed.CursorPos() != origin {
		t.Errorf("cursor = %d, want restored %d", m.ed.CursorPos(), origin)
	}
}

func TestSearchCaseToggle(t *testing.T) {
	m := newModel(t, "FOO foo Foo", "")
	m, _ = send(m, ctrl(tea.KeyCtrlS))
	for _, r := range "foo" {
		m, _ = send(m, runes(string(r)))
	}
	// Case-insensitive by default: all three.
	if len(m.matches) != 3 {
		t.Fatalf("insensitive matches = %d, want 3", len(m.matches))
	}
	// C-t toggles to case-sensitive: only the lowercase "foo".
	m, _ = send(m, ctrl(tea.KeyCtrlT))
	if len(m.matches) != 1 {
		t.Fatalf("sensitive matches = %d, want 1", len(m.matches))
	}
}

func TestReplaceAllFlow(t *testing.T) {
	m := newModel(t, "fox fox fox", "")
	m, _ = send(m, altRune("%")) // replace binding is alt+%
	if m.mode != modeReplaceFrom {
		t.Fatalf("mode = %v, want replaceFrom", m.mode)
	}
	for _, r := range "fox" {
		m, _ = send(m, runes(string(r)))
	}
	m, _ = send(m, key(tea.KeyEnter)) // move to replacement entry
	if m.mode != modeReplaceTo {
		t.Fatalf("mode = %v, want replaceTo", m.mode)
	}
	for _, r := range "cat" {
		m, _ = send(m, runes(string(r)))
	}
	m, _ = send(m, key(tea.KeyEnter)) // perform replacement
	if got := string(m.ed.Bytes()); got != "cat cat cat" {
		t.Fatalf("buffer = %q, want %q", got, "cat cat cat")
	}
	if m.mode != modeNormal {
		t.Errorf("mode = %v, want normal", m.mode)
	}
}

func TestReplaceCancel(t *testing.T) {
	m := newModel(t, "keep me", "")
	m, _ = send(m, altRune("%"))
	for _, r := range "keep" {
		m, _ = send(m, runes(string(r)))
	}
	m, _ = send(m, key(tea.KeyEsc))
	if m.mode != modeNormal {
		t.Errorf("mode = %v, want normal", m.mode)
	}
	if got := string(m.ed.Bytes()); got != "keep me" {
		t.Errorf("buffer changed on cancel: %q", got)
	}
}

func TestSyntaxEnterStatesThreadAcrossLines(t *testing.T) {
	// A Go block comment spans lines 1–2; the cache must carry the state.
	src := "package main\n/* block\ncomment */\nvar x = 1"
	m := New(editor.New([]byte(src), "main.go"), config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = updated.(*Model)
	_ = m.View() // triggers ensureSyntax

	want := []syntax.State{
		syntax.StateDefault,      // line 0: package main
		syntax.StateDefault,      // line 1: /* block   (enters comment at end)
		syntax.StateBlockComment, // line 2: comment */ (was inside comment)
		syntax.StateDefault,      // line 3: var x = 1
	}
	if !reflect.DeepEqual(m.enterStates, want) {
		t.Errorf("enterStates = %v, want %v", m.enterStates, want)
	}
}

func TestSyntaxDisabledNoStates(t *testing.T) {
	cfg := config.Default()
	cfg.Features.SyntaxHighlighting = false
	m := New(editor.New([]byte("package main"), "main.go"), cfg)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = updated.(*Model)
	_ = m.View()
	if len(m.enterStates) != 0 {
		t.Errorf("enterStates should be empty when highlighting disabled, got %d", len(m.enterStates))
	}
}

func TestViewWithGoFileDoesNotPanic(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tx := `raw\n\tstring`\n\t_ = x // done\n}\n"
	m := New(editor.New([]byte(src), "main.go"), config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 6})
	m = updated.(*Model)
	if out := m.View(); out == "" {
		t.Error("expected non-empty view")
	}
}
