package ui

import (
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"

	"argc.dev/chiquito/internal/syntax"
)

// defaultChromaStyle is used when the configured theme name is "default" or is
// not a known Chroma style.
const defaultChromaStyle = "monokai"

// styledSpan is a rune range [start, end) within a line and the style to render
// it with.
type styledSpan struct {
	start, end int
	style      lipgloss.Style
}

// highlighter produces per-line styled spans for the document. It uses Chroma
// (≈250 languages, themed colors) when a lexer matches the file name, and falls
// back to chiquito's builtin tokenizer otherwise. Either way it exposes the same
// lineSpans the renderer consumes.
type highlighter struct {
	name      string
	lineSpans [][]styledSpan

	// Chroma path.
	lexer chroma.Lexer
	style *chroma.Style
	cache map[chroma.TokenType]lipgloss.Style

	// Builtin fallback path.
	builtin syntax.Language
	theme   theme
}

func newHighlighter(filename, themeName string, th theme) *highlighter {
	h := &highlighter{theme: th, cache: make(map[chroma.TokenType]lipgloss.Style)}
	if lx := lexers.Match(filepath.Base(filename)); lx != nil {
		h.lexer = chroma.Coalesce(lx)
		h.style = chromaStyle(themeName)
		h.name = strings.ToLower(h.lexer.Config().Name)
	} else {
		h.builtin = syntax.ForFilename(filename)
		h.name = h.builtin.Name()
	}
	return h
}

func chromaStyle(name string) *chroma.Style {
	if name == "" || name == "default" {
		name = defaultChromaStyle
	}
	s := styles.Get(name)
	if s == styles.Fallback { // unknown name → use our default instead of swapoff
		s = styles.Get(defaultChromaStyle)
	}
	return s
}

// highlight (re)computes lineSpans for the whole document.
func (h *highlighter) highlight(text string) {
	if h.lexer != nil {
		h.highlightChroma(text)
		return
	}
	h.highlightBuiltin(text)
}

func (h *highlighter) highlightChroma(text string) {
	h.lineSpans = h.lineSpans[:0]
	it, err := h.lexer.Tokenise(nil, text)
	if err != nil {
		return
	}

	var cur []styledSpan
	col := 0
	for _, tok := range it.Tokens() {
		st := h.chromaStyleFor(tok.Type)
		val := tok.Value
		for {
			nl := strings.IndexByte(val, '\n')
			seg := val
			if nl >= 0 {
				seg = val[:nl]
			}
			if len(seg) > 0 {
				rc := utf8.RuneCountInString(seg)
				cur = append(cur, styledSpan{start: col, end: col + rc, style: st})
				col += rc
			}
			if nl < 0 {
				break
			}
			h.lineSpans = append(h.lineSpans, cur)
			cur = nil
			col = 0
			val = val[nl+1:]
		}
	}
	h.lineSpans = append(h.lineSpans, cur)
}

func (h *highlighter) chromaStyleFor(tt chroma.TokenType) lipgloss.Style {
	if s, ok := h.cache[tt]; ok {
		return s
	}
	e := h.style.Get(tt)
	st := lipgloss.NewStyle()
	if e.Colour.IsSet() {
		st = st.Foreground(lipgloss.Color(e.Colour.String()))
	}
	if e.Bold == chroma.Yes {
		st = st.Bold(true)
	}
	if e.Italic == chroma.Yes {
		st = st.Italic(true)
	}
	if e.Underline == chroma.Yes {
		st = st.Underline(true)
	}
	h.cache[tt] = st
	return st
}

func (h *highlighter) highlightBuiltin(text string) {
	h.lineSpans = h.lineSpans[:0]
	st := syntax.StateDefault
	for _, line := range strings.Split(text, "\n") {
		toks, next := h.builtin.TokenizeLine(line, st)
		st = next
		spans := make([]styledSpan, 0, len(toks))
		for _, tk := range toks {
			spans = append(spans, styledSpan{start: tk.Start, end: tk.End, style: h.theme.style(tk.Type)})
		}
		h.lineSpans = append(h.lineSpans, spans)
	}
}

// spansFor returns the styled spans for a line, or nil if out of range.
func (h *highlighter) spansFor(line int) []styledSpan {
	if line < 0 || line >= len(h.lineSpans) {
		return nil
	}
	return h.lineSpans[line]
}
