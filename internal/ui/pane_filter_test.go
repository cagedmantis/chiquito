package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func typePane(p *filePane, s string) {
	for _, r := range s {
		p.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestFilePaneFilterNarrows(t *testing.T) {
	dir, _, _ := makeTree(t) // sub/, alpha.txt, beta.go
	p := newFilePane(dir)
	if len(p.matches) != 4 {
		t.Fatalf("unfiltered matches = %d, want 4", len(p.matches))
	}

	typePane(p, "be")
	if len(p.matches) != 1 {
		t.Fatalf("filtered matches = %d, want 1 (%v)", len(p.matches), p.matchLabels())
	}
	if e := p.current(); e == nil || e.label != "beta.go" {
		t.Errorf("current = %v, want beta.go", e)
	}

	// Backspacing restores entries.
	p.update(tea.KeyMsg{Type: tea.KeyBackspace})
	p.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.filter != "" || len(p.matches) != 4 {
		t.Errorf("after clearing filter: filter=%q matches=%d, want empty/4", p.filter, len(p.matches))
	}
}

func TestFilePaneFilterRanking(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"domain.go", "main.go", "main_test.go", "readme.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := newFilePane(dir)
	typePane(p, "main")

	labels := p.matchLabels()
	// readme.md excluded; main.go ranks first (prefix, shortest).
	for _, l := range labels {
		if l == "readme.md" {
			t.Error("readme.md should not match 'main'")
		}
	}
	if len(labels) == 0 || labels[0] != "main.go" {
		t.Errorf("ranked labels = %v, want main.go first", labels)
	}
}

func TestFilePaneFilterThenOpen(t *testing.T) {
	dir, _, _ := makeTree(t)
	p := newFilePane(dir)
	typePane(p, "alp") // -> alpha.txt
	outcome, cmd := p.update(tea.KeyMsg{Type: tea.KeyEnter})
	if outcome != paneClose || cmd == nil {
		t.Fatal("selecting a filtered file should close the pane with an open command")
	}
	msg, ok := cmd().(openFileMsg)
	if !ok || filepath.Base(msg.path) != "alpha.txt" {
		t.Errorf("open msg = %v, want alpha.txt", cmd())
	}
}

func TestFilePaneBackspaceEmptyGoesToParent(t *testing.T) {
	dir, sub, _ := makeTree(t)
	p := newFilePane(sub)
	if p.dir != sub {
		t.Fatalf("start dir = %q, want %q", p.dir, sub)
	}
	// Filter empty: backspace navigates to the parent.
	p.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.dir != dir {
		t.Errorf("after backspace dir = %q, want parent %q", p.dir, dir)
	}
}

func TestFilePaneFilterResetsOnNavigate(t *testing.T) {
	dir, _, _ := makeTree(t)
	p := newFilePane(dir)
	typePane(p, "su") // matches sub/
	// Enter the directory; the filter should reset.
	p.update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.filter != "" {
		t.Errorf("filter = %q, want reset after navigating", p.filter)
	}
}

func TestFilePaneFilterShownInView(t *testing.T) {
	dir, _, _ := makeTree(t)
	p := newFilePane(dir)
	typePane(p, "be")
	out := p.view(120, filePaneHeight)
	if !strings.Contains(out, "filter: be") {
		t.Errorf("view should show the active filter:\n%s", out)
	}
}

func TestFilePaneMatchHighlightStyle(t *testing.T) {
	p := &filePane{}
	file := fileEntry{label: "beta.go", isDir: false}

	// A matched character is bold, underlined, and uses the match color.
	m := p.cellStyle(file, false, true)
	if c, ok := m.GetForeground().(lipgloss.Color); !ok || c != paneMatchColor {
		t.Errorf("matched fg = %#v, want %v", m.GetForeground(), paneMatchColor)
	}
	if !m.GetBold() || !m.GetUnderline() {
		t.Error("matched cell should be bold + underlined")
	}

	// A non-matched file character has no special foreground.
	n := p.cellStyle(file, false, false)
	if _, ok := n.GetForeground().(lipgloss.Color); ok {
		t.Error("non-matched file cell should have no foreground color")
	}

	// Directories are colored; selection adds reverse video.
	d := p.cellStyle(fileEntry{label: "sub/", isDir: true}, false, false)
	if _, ok := d.GetForeground().(lipgloss.Color); !ok {
		t.Error("directory cell should be colored")
	}
	if !p.cellStyle(file, true, false).GetReverse() {
		t.Error("selected cell should be reverse-video")
	}
}

func TestFilePaneRenderEntryWidth(t *testing.T) {
	p := &filePane{}
	row := p.renderEntry(fileEntry{label: "beta.go"}, false, []int{0, 1}, 30)
	if w := lipgloss.Width(row); w != 30 {
		t.Errorf("rendered row width = %d, want 30", w)
	}
}

// matchLabels is a test helper returning the labels of currently matched
// entries in ranked order.
func (p *filePane) matchLabels() []string {
	out := make([]string, len(p.matches))
	for i, idx := range p.matches {
		out[i] = p.entries[idx].label
	}
	return out
}
