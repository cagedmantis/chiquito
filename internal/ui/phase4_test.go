package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/spell"
)

func typeString(m *Model, s string) *Model {
	for _, r := range s {
		m, _ = send(m, runes(string(r)))
	}
	return m
}

func TestOpenFileFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(t, "scratch", "")
	m.cfg.Features.FilePane = false // exercise the text-prompt path
	// C-x C-f opens the file-open prompt.
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	if m.mode != modeOpen {
		t.Fatalf("mode = %v, want modeOpen", m.mode)
	}
	m = typeString(m, path)
	m, _ = send(m, key(tea.KeyEnter))

	if m.mode != modeNormal {
		t.Errorf("mode = %v, want normal after open", m.mode)
	}
	if got := string(m.ed.Bytes()); got != "package main\n" {
		t.Errorf("buffer = %q, want package main", got)
	}
	if m.ed.Name() != path {
		t.Errorf("name = %q, want %q", m.ed.Name(), path)
	}
	if m.langName() != "go" {
		t.Errorf("language = %q, want go (re-selected on open)", m.langName())
	}
}

func TestOpenMissingFileOpensEmptyBuffer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")
	m := newModel(t, "", "")
	m.cfg.Features.FilePane = false // exercise the text-prompt path
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	m = typeString(m, path)
	m, _ = send(m, key(tea.KeyEnter))
	if m.mode != modeNormal || m.ed.Name() != path {
		t.Fatalf("expected empty buffer bound to %q, got name %q mode %v", path, m.ed.Name(), m.mode)
	}
	if string(m.ed.Bytes()) != "" {
		t.Errorf("expected empty buffer, got %q", m.ed.Bytes())
	}
}

func TestSaveAsFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	m := newModel(t, "", "") // scratch buffer, no name
	m = typeString(m, "# Title")
	// C-x C-s on a nameless buffer starts save-as.
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlS))
	if m.mode != modeSaveAs {
		t.Fatalf("mode = %v, want modeSaveAs", m.mode)
	}
	m = typeString(m, path)
	m, _ = send(m, key(tea.KeyEnter))

	if m.ed.Name() != path {
		t.Errorf("name = %q, want %q", m.ed.Name(), path)
	}
	if m.langName() != "markdown" {
		t.Errorf("language = %q, want markdown", m.langName())
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "# Title" {
		t.Fatalf("file = %q, err = %v", got, err)
	}
	if m.ed.Dirty() {
		t.Error("buffer should be clean after save-as")
	}
}

func TestPromptCancel(t *testing.T) {
	m := newModel(t, "untouched", "orig.txt")
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	m = typeString(m, "/some/path")
	m, _ = send(m, key(tea.KeyEsc))
	if m.mode != modeNormal {
		t.Errorf("mode = %v, want normal", m.mode)
	}
	if m.ed.Name() != "orig.txt" || string(m.ed.Bytes()) != "untouched" {
		t.Error("cancel should not change the document")
	}
}

func TestSpellAsyncFlow(t *testing.T) {
	m := newModel(t, "helo wrld", "notes.txt")
	// Simulate the dictionary finishing loading off-thread.
	d := spell.NewWordSet("hello", "world")
	upd, cmd := m.Update(dictLoadedMsg{dict: d})
	m = upd.(*Model)
	if cmd == nil {
		t.Fatal("expected a spell-check command after dictionary load")
	}
	// Run the spell command and feed back its result message.
	msg := cmd()
	upd, _ = m.Update(msg)
	m = upd.(*Model)

	if len(m.spellSpans) != 2 {
		t.Fatalf("spellSpans = %d, want 2 (helo, wrld)", len(m.spellSpans))
	}
	// Spans are document rune ranges: "helo" = [0,4), "wrld" = [5,9).
	if m.spellSpans[0] != (spell.Misspelling{Start: 0, End: 4}) {
		t.Errorf("span[0] = %+v, want {0,4}", m.spellSpans[0])
	}
}

func TestSpellResultVersionGuard(t *testing.T) {
	m := newModel(t, "helo", "notes.txt")
	m.checker = spell.NewWordSet("hello")
	// An edit bumps docVersion, making any older result stale.
	m, _ = send(m, runes("x"))
	stale := spellResultMsg{version: 0, spans: []spell.Misspelling{{Start: 0, End: 4}}}
	upd, _ := m.Update(stale)
	m = upd.(*Model)
	if len(m.spellSpans) != 0 {
		t.Errorf("stale spell result should be discarded, got %d spans", len(m.spellSpans))
	}
}

func TestConfigHotReload(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)

	// Write a config that differs from the defaults.
	changed := config.Default()
	changed.Editor.LineNumbers = false
	changed.Editor.TabWidth = 7
	if err := config.Save(changed); err != nil {
		t.Fatal(err)
	}

	m := New(editor.New([]byte("x"), "f.txt"), config.Default())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = updated.(*Model)
	if !m.lineNumbers {
		t.Fatal("precondition: model should start with default line numbers on")
	}

	// Force the watcher to see the file as new and reload.
	m.configMod = time.Time{}
	m.maybeReloadConfig()

	if m.lineNumbers {
		t.Error("hot-reload did not disable line numbers")
	}
	if m.tabWidth != 7 {
		t.Errorf("hot-reload tabWidth = %d, want 7", m.tabWidth)
	}
}

func TestTerminalResize(t *testing.T) {
	m := newModel(t, strings.Repeat("a line of text\n", 50), "f.txt")
	// Shrink, then grow; View must reflect the new height and never panic.
	for _, sz := range []struct{ w, h int }{{10, 4}, {120, 40}, {1, 1}} {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		m = updated.(*Model)
		out := m.View()
		gotLines := strings.Count(out, "\n")
		// textHeight rows + status bar; allow the clamped minimum.
		if gotLines < 1 {
			t.Errorf("size %dx%d produced %d lines", sz.w, sz.h, gotLines)
		}
	}
}

func TestConfigReloadRebindsKeys(t *testing.T) {
	cfg := config.Default()
	// Emacs-notation binding for an action, exercising NormalizeChord in the UI.
	cfg.Keys.Bindings["line-end"] = "C-o"
	m := New(editor.New([]byte("hello world"), "f.txt"), cfg)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = updated.(*Model)

	// C-o (ctrl+o) should now move to end of line.
	m, _ = send(m, ctrl(tea.KeyCtrlO))
	if _, col := m.ed.CursorLineCol(); col != 11 {
		t.Errorf("after C-o col = %d, want 11 (line-end via Emacs binding)", col)
	}
}
