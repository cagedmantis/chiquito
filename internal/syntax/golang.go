package syntax

// Go is a lightweight tokenizer for Go source. It is not a full lexer — it is
// good enough for coloring: keywords, identifiers, numbers, strings (including
// multi-line raw strings), runes, operators, and line/block comments.
type Go struct{}

func (Go) Name() string { return "go" }

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
	// predeclared names worth highlighting as keyword-ish
	"nil": true, "true": true, "false": true, "iota": true,
}

func (g Go) TokenizeLine(line string, in State) ([]Token, State) {
	r := []rune(line)
	n := len(r)
	var toks []Token

	// Resume a multi-line construct entered on a previous line.
	switch in {
	case StateBlockComment:
		if end, ok := indexRunes(r, 0, []rune("*/")); ok {
			toks = append(toks, Token{0, end + 2, Comment})
			return g.scan(r, end+2, toks)
		}
		return []Token{{0, n, Comment}}, StateBlockComment
	case StateRawString:
		if end, ok := indexRune(r, 0, '`'); ok {
			toks = append(toks, Token{0, end + 1, String})
			return g.scan(r, end+1, toks)
		}
		return []Token{{0, n, String}}, StateRawString
	}
	return g.scan(r, 0, toks)
}

// scan tokenizes r starting at offset i in the default state.
func (g Go) scan(r []rune, i int, toks []Token) ([]Token, State) {
	n := len(r)
	for i < n {
		c := r[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '/' && i+1 < n && r[i+1] == '/':
			toks = append(toks, Token{i, n, Comment})
			return toks, StateDefault
		case c == '/' && i+1 < n && r[i+1] == '*':
			if end, ok := indexRunes(r, i+2, []rune("*/")); ok {
				toks = append(toks, Token{i, end + 2, Comment})
				i = end + 2
			} else {
				toks = append(toks, Token{i, n, Comment})
				return toks, StateBlockComment
			}
		case c == '"':
			end := scanQuoted(r, i, '"')
			toks = append(toks, Token{i, end, String})
			i = end
		case c == '\'':
			end := scanQuoted(r, i, '\'')
			toks = append(toks, Token{i, end, String})
			i = end
		case c == '`':
			if end, ok := indexRune(r, i+1, '`'); ok {
				toks = append(toks, Token{i, end + 1, String})
				i = end + 1
			} else {
				toks = append(toks, Token{i, n, String})
				return toks, StateRawString
			}
		case isDigit(c):
			j := i + 1
			for j < n && (isHexDigit(r[j]) || r[j] == 'x' || r[j] == 'X' || r[j] == '.' || r[j] == '_') {
				j++
			}
			toks = append(toks, Token{i, j, Number})
			i = j
		case isIdentStart(c):
			j := i + 1
			for j < n && isIdentPart(r[j]) {
				j++
			}
			if goKeywords[string(r[i:j])] {
				toks = append(toks, Token{i, j, Keyword})
			} else {
				toks = append(toks, Token{i, j, Ident})
			}
			i = j
		case isOperator(c):
			toks = append(toks, Token{i, i + 1, Operator})
			i++
		default:
			i++
		}
	}
	return toks, StateDefault
}

// scanQuoted returns the index just past a closing quote q starting at the
// opening quote at i, honoring backslash escapes; if unterminated it returns n.
func scanQuoted(r []rune, i int, q rune) int {
	n := len(r)
	j := i + 1
	for j < n {
		if r[j] == '\\' {
			j += 2
			continue
		}
		if r[j] == q {
			return j + 1
		}
		j++
	}
	return n
}

func indexRune(r []rune, from int, target rune) (int, bool) {
	for i := from; i < len(r); i++ {
		if r[i] == target {
			return i, true
		}
	}
	return 0, false
}

func indexRunes(r []rune, from int, target []rune) (int, bool) {
	for i := from; i+len(target) <= len(r); i++ {
		match := true
		for k, t := range target {
			if r[i+k] != t {
				match = false
				break
			}
		}
		if match {
			return i, true
		}
	}
	return 0, false
}

func isDigit(c rune) bool    { return c >= '0' && c <= '9' }
func isHexDigit(c rune) bool { return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') }

func isIdentStart(c rune) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}
func isIdentPart(c rune) bool { return isIdentStart(c) || isDigit(c) }

func isOperator(c rune) bool {
	switch c {
	case '+', '-', '*', '/', '%', '=', '<', '>', '!', '&', '|', '^', ':', '.', ',', ';', '(', ')', '{', '}', '[', ']':
		return true
	}
	return false
}
