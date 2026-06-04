package syntax

import (
	"strings"
	"testing"
)

const goLine = `func process(items []string) (int, error) { return len(items), nil } // count`

// BenchmarkTokenizeGo measures tokenizing a representative line of Go.
func BenchmarkTokenizeGo(b *testing.B) {
	lang := Go{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lang.TokenizeLine(goLine, StateDefault)
	}
}

// BenchmarkTokenizeGoFile measures tokenizing a whole file's worth of lines with
// carried state, the realistic per-edit cost.
func BenchmarkTokenizeGoFile(b *testing.B) {
	lines := make([]string, 2000)
	for i := range lines {
		lines[i] = goLine
	}
	lang := Go{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st := StateDefault
		for _, ln := range lines {
			_, st = lang.TokenizeLine(ln, st)
		}
	}
}

func BenchmarkTokenizeMarkdown(b *testing.B) {
	line := strings.Repeat("a ", 10) + "`code` and *emph* and [link](http://x)"
	lang := Markdown{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lang.TokenizeLine(line, StateDefault)
	}
}
