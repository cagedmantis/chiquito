package syntax

import "testing"

// tokenTypeAt returns the token type covering rune offset col, or Text if none.
func tokenTypeAt(toks []Token, col int) TokenType {
	for _, t := range toks {
		if col >= t.Start && col < t.End {
			return t.Type
		}
	}
	return Text
}

func TestForFilename(t *testing.T) {
	cases := map[string]string{
		"main.go":    "go",
		"READ.md":    "markdown",
		"a.markdown": "markdown",
		"notes.txt":  "plain",
		"noext":      "plain",
	}
	for name, want := range cases {
		if got := ForFilename(name).Name(); got != want {
			t.Errorf("ForFilename(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestGoKeywordsAndComments(t *testing.T) {
	toks, st := Go{}.TokenizeLine(`func main() { // hi`, StateDefault)
	if st != StateDefault {
		t.Errorf("state = %d, want default", st)
	}
	if got := tokenTypeAt(toks, 0); got != Keyword { // "func"
		t.Errorf("offset 0 type = %d, want Keyword", got)
	}
	if got := tokenTypeAt(toks, 5); got != Ident { // "main"
		t.Errorf("offset 5 type = %d, want Ident", got)
	}
	if got := tokenTypeAt(toks, 15); got != Comment { // inside "// hi"
		t.Errorf("offset 15 type = %d, want Comment", got)
	}
}

func TestGoString(t *testing.T) {
	toks, _ := Go{}.TokenizeLine(`x := "hello"`, StateDefault)
	if got := tokenTypeAt(toks, 6); got != String {
		t.Errorf("string offset type = %d, want String", got)
	}
}

func TestGoBlockCommentSpansLines(t *testing.T) {
	toks, st := Go{}.TokenizeLine(`a := 1 /* start`, StateDefault)
	if st != StateBlockComment {
		t.Fatalf("state = %d, want StateBlockComment", st)
	}
	if got := tokenTypeAt(toks, 8); got != Comment {
		t.Errorf("type at 8 = %d, want Comment", got)
	}
	// Middle line: entirely comment, still in block.
	toks, st = Go{}.TokenizeLine(`still comment`, StateBlockComment)
	if st != StateBlockComment || tokenTypeAt(toks, 0) != Comment {
		t.Errorf("middle line: state=%d type=%d", st, tokenTypeAt(toks, 0))
	}
	// Closing line: comment ends, code resumes.
	toks, st = Go{}.TokenizeLine(`end */ x`, StateBlockComment)
	if st != StateDefault {
		t.Errorf("state after close = %d, want default", st)
	}
	if tokenTypeAt(toks, 0) != Comment {
		t.Error("expected comment at start of closing line")
	}
	if tokenTypeAt(toks, 7) != Ident {
		t.Error("expected ident after comment close")
	}
}

func TestGoRawStringSpansLines(t *testing.T) {
	_, st := Go{}.TokenizeLine("s := `multi", StateDefault)
	if st != StateRawString {
		t.Fatalf("state = %d, want StateRawString", st)
	}
	toks, st := Go{}.TokenizeLine("line`+x", StateRawString)
	if st != StateDefault {
		t.Errorf("state after close = %d, want default", st)
	}
	if tokenTypeAt(toks, 0) != String {
		t.Error("expected string at start of raw-string close line")
	}
}

func TestMarkdownHeadingAndFence(t *testing.T) {
	toks, st := Markdown{}.TokenizeLine("# Title", StateDefault)
	if st != StateDefault || tokenTypeAt(toks, 0) != Heading {
		t.Errorf("heading: state=%d type=%d", st, tokenTypeAt(toks, 0))
	}
	// Fence open -> enter fence state.
	_, st = Markdown{}.TokenizeLine("```go", StateDefault)
	if st != StateFence {
		t.Fatalf("after fence open state = %d, want StateFence", st)
	}
	// Inside fence: code.
	toks, st = Markdown{}.TokenizeLine("func x()", StateFence)
	if st != StateFence || tokenTypeAt(toks, 0) != Code {
		t.Errorf("inside fence: state=%d type=%d", st, tokenTypeAt(toks, 0))
	}
	// Fence close.
	_, st = Markdown{}.TokenizeLine("```", StateFence)
	if st != StateDefault {
		t.Errorf("after fence close state = %d, want default", st)
	}
}

func TestMarkdownInline(t *testing.T) {
	toks := Markdown{}.inline([]rune("a `code` and *em* and [t](u)"))
	if tokenTypeAt(toks, 3) != Code {
		t.Error("expected code span")
	}
	if tokenTypeAt(toks, 13) != Emphasis {
		t.Error("expected emphasis")
	}
	// The link starts at '['.
	idx := indexOf("a `code` and *em* and [t](u)", '[')
	if tokenTypeAt(toks, idx) != Link {
		t.Errorf("expected link at %d", idx)
	}
}

func indexOf(s string, target rune) int {
	for i, r := range []rune(s) {
		if r == target {
			return i
		}
	}
	return -1
}
