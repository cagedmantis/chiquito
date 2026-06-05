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

func TestOpenPromptPrefillsCurrentFileDir(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "current.go")
	target := filepath.Join(dir, "target.md")
	if err := os.WriteFile(target, []byte("# hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Features.FilePane = false // exercise the text-prompt path
	m := New(editor.New([]byte("x"), current), cfg)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 10})
	m = updated.(*Model)

	// C-x C-f should prefill the prompt with the current file's directory.
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	wantPrefix := dir + string(filepath.Separator)
	if m.input.String() != wantPrefix {
		t.Fatalf("prefill = %q, want %q", m.input.String(), wantPrefix)
	}

	// Typing just the filename appends to the prefilled directory.
	m = typeString(m, "target.md")
	m, _ = send(m, key(tea.KeyEnter))

	if m.ed.Name() != target {
		t.Errorf("opened name = %q, want %q", m.ed.Name(), target)
	}
	if got := string(m.ed.Bytes()); got != "# hi\n" {
		t.Errorf("content = %q", got)
	}
	if m.langName() != "markdown" {
		t.Errorf("language = %q, want markdown", m.langName())
	}
}

func TestOpenPromptPrefillsWorkingDir(t *testing.T) {
	// A scratch buffer (no file name) falls back to the working directory.
	m := newModel(t, "", "")
	m.cfg.Features.FilePane = false // exercise the text-prompt path
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))

	got := m.input.String()
	if !strings.HasSuffix(got, string(filepath.Separator)) {
		t.Errorf("prefill %q should end with a separator", got)
	}
	if wd, err := os.Getwd(); err == nil {
		if got != wd+string(filepath.Separator) {
			t.Errorf("prefill = %q, want working dir %q", got, wd+string(filepath.Separator))
		}
	}
}

func TestResolveOpenPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	sep := string(filepath.Separator)
	cases := []struct {
		in, want string
	}{
		{"/a/b/c.txt", "/a/b/c.txt"},
		{"/cwd/" + sep + "tmp/x.go", "/tmp/x.go"},                // "//" resets to root
		{"/cwd/~" + sep + "notes.txt", home + sep + "notes.txt"}, // "/~" resets to home
		{"~" + sep + "x", home + sep + "x"},                      // leading ~ expands
		{"plain.txt", "plain.txt"},
	}
	for _, c := range cases {
		if got := resolveOpenPath(c.in); got != c.want {
			t.Errorf("resolveOpenPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestOpenBareDirectoryReportsNotAFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Features.FilePane = false // exercise the text-prompt path
	m := New(editor.New([]byte("x"), filepath.Join(dir, "f.txt")), cfg)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 10})
	m = updated.(*Model)
	// Accept the prefilled directory without typing a filename.
	m, _ = send(m, ctrl(tea.KeyCtrlX))
	m, _ = send(m, ctrl(tea.KeyCtrlF))
	m, _ = send(m, key(tea.KeyEnter))
	if m.mode != modeNormal {
		t.Errorf("mode = %v, want normal", m.mode)
	}
	if !strings.Contains(m.status, "Not a file") {
		t.Errorf("status = %q, want a 'Not a file' message", m.status)
	}
	// The original buffer must be untouched.
	if string(m.ed.Bytes()) != "x" {
		t.Errorf("buffer changed: %q", m.ed.Bytes())
	}
}
