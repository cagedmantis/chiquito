package ui

import (
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/lipgloss"
)

func TestHighlighterChromaLanguages(t *testing.T) {
	th := themeByName("default")
	cases := []struct {
		file, lang, src string
	}{
		{"main.go", "go", "package main\nfunc main() {}\n"},
		{"app.py", "python", "def f(x):\n    return x + 1\n"},
		{"data.json", "json", "{\n  \"a\": 1\n}\n"},
		{"readme.md", "markdown", "# Title\n\nsome *text*\n"},
	}
	for _, c := range cases {
		h := newHighlighter(c.file, "default", th)
		if h.name != c.lang {
			t.Errorf("%s: name = %q, want %q", c.file, h.name, c.lang)
		}
		h.highlight(c.src)
		if len(h.lineSpans) == 0 {
			t.Errorf("%s: produced no line spans", c.file)
		}
		// Spans must stay within each line and be ordered.
		for li, spans := range h.lineSpans {
			prev := 0
			for _, sp := range spans {
				if sp.start < prev || sp.end < sp.start {
					t.Errorf("%s line %d: bad span %+v (prev end %d)", c.file, li, sp, prev)
				}
				prev = sp.end
			}
		}
	}
}

func TestHighlighterUnknownFallsBackToBuiltin(t *testing.T) {
	th := themeByName("default")
	h := newHighlighter("notes.unknownext", "default", th)
	// No Chroma lexer matches, so the builtin (plain) tokenizer is used.
	if h.lexer != nil {
		t.Error("expected no Chroma lexer for unknown extension")
	}
	if h.name != "plain" {
		t.Errorf("name = %q, want plain", h.name)
	}
	h.highlight("just some text\nwith two lines\n")
	if len(h.lineSpans) == 0 {
		t.Error("builtin highlighter should still produce per-line entries")
	}
}

func TestHighlighterChromaStyleHasColor(t *testing.T) {
	h := newHighlighter("main.go", "monokai", themeByName("default"))
	// A keyword in the Monokai style carries a foreground color.
	st := h.chromaStyleFor(chroma.KeywordDeclaration)
	if c, ok := st.GetForeground().(lipgloss.Color); !ok || string(c) == "" {
		t.Errorf("keyword style foreground = %#v, want a non-empty color", st.GetForeground())
	}
}
