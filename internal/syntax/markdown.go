package syntax

import "strings"

// Markdown is a lightweight tokenizer for Markdown: ATX headings, fenced code
// blocks (``` ... ```), inline code spans, emphasis (* or _), and links.
type Markdown struct{}

func (Markdown) Name() string { return "markdown" }

func (m Markdown) TokenizeLine(line string, in State) ([]Token, State) {
	r := []rune(line)
	n := len(r)

	trimmed := strings.TrimSpace(line)

	// Fenced code blocks.
	if in == StateFence {
		if strings.HasPrefix(trimmed, "```") {
			return []Token{{0, n, Code}}, StateDefault // closing fence
		}
		return []Token{{0, n, Code}}, StateFence
	}
	if strings.HasPrefix(trimmed, "```") {
		return []Token{{0, n, Code}}, StateFence // opening fence
	}

	// ATX heading: the whole line.
	if strings.HasPrefix(trimmed, "#") {
		return []Token{{0, n, Heading}}, StateDefault
	}

	return m.inline(r), StateDefault
}

// inline scans a default-state line for code spans, emphasis runs, and links.
func (m Markdown) inline(r []rune) []Token {
	n := len(r)
	var toks []Token
	i := 0
	for i < n {
		switch {
		case r[i] == '`':
			if end, ok := indexRune(r, i+1, '`'); ok {
				toks = append(toks, Token{i, end + 1, Code})
				i = end + 1
				continue
			}
			i++
		case r[i] == '*' || r[i] == '_':
			marker := r[i]
			// Bold uses a doubled marker; treat either as emphasis coloring.
			width := 1
			if i+1 < n && r[i+1] == marker {
				width = 2
			}
			if end, ok := findCloser(r, i+width, marker, width); ok {
				toks = append(toks, Token{i, end + width, Emphasis})
				i = end + width
				continue
			}
			i++
		case r[i] == '[':
			if tok, next, ok := scanLink(r, i); ok {
				toks = append(toks, tok)
				i = next
				continue
			}
			i++
		default:
			i++
		}
	}
	return toks
}

// findCloser finds the start index of a run of `width` copies of marker at or
// after from.
func findCloser(r []rune, from int, marker rune, width int) (int, bool) {
	n := len(r)
	for i := from; i < n; i++ {
		if r[i] != marker {
			continue
		}
		if width == 1 {
			return i, true
		}
		if i+1 < n && r[i+1] == marker {
			return i, true
		}
	}
	return 0, false
}

// scanLink matches a [text](url) link starting at the '[' at i.
func scanLink(r []rune, i int) (Token, int, bool) {
	closeBracket, ok := indexRune(r, i+1, ']')
	if !ok || closeBracket+1 >= len(r) || r[closeBracket+1] != '(' {
		return Token{}, i, false
	}
	closeParen, ok := indexRune(r, closeBracket+2, ')')
	if !ok {
		return Token{}, i, false
	}
	return Token{i, closeParen + 1, Link}, closeParen + 1, true
}
