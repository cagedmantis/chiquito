// Package syntax is chiquito's tokenizer engine. It is pure (standard library
// only) and line-oriented: each language tokenizes one line at a time, given the
// lexical state carried in from the previous line, and returns the state to
// carry to the next. This design supports highlighting only the visible
// viewport (re-deriving the entering state from a cached vector) and incremental
// re-tokenization after edits, without ever holding a parse tree.
//
// Tokens carry rune offsets within their line; the UI layer maps token types to
// colors via a theme.
package syntax

import (
	"path/filepath"
	"strings"
)

// TokenType classifies a span of text for coloring.
type TokenType uint8

const (
	Text TokenType = iota
	Keyword
	Ident
	Number
	String
	Comment
	Operator
	// Markdown-oriented types.
	Heading
	Emphasis
	Code
	Link
)

// Token is a half-open rune range [Start, End) within a single line.
type Token struct {
	Start int
	End   int
	Type  TokenType
}

// State is the lexical context carried between lines (e.g. inside a Go block
// comment or a Markdown fenced code block).
type State uint8

const (
	StateDefault      State = iota
	StateBlockComment       // Go: inside /* ... */
	StateRawString          // Go: inside a `...` raw string
	StateFence              // Markdown: inside a ``` fenced code block
)

// Language tokenizes source text one line at a time.
type Language interface {
	Name() string
	// TokenizeLine returns the tokens for line (which must not contain a
	// newline) given the entering state, plus the state to carry to the next
	// line.
	TokenizeLine(line string, in State) ([]Token, State)
}

// ForFilename selects a language from a file name's extension, falling back to
// Plain (no highlighting) for unknown types.
func ForFilename(name string) Language {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go":
		return Go{}
	case ".md", ".markdown":
		return Markdown{}
	default:
		return Plain{}
	}
}

// Plain performs no tokenization.
type Plain struct{}

func (Plain) Name() string { return "plain" }

func (Plain) TokenizeLine(string, State) ([]Token, State) { return nil, StateDefault }
