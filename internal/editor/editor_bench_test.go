package editor

import (
	"strings"
	"testing"
)

func bigEditor(lines int) *Editor {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		sb.WriteString("the quick brown fox jumps over the lazy dog\n")
	}
	return New([]byte(sb.String()), "big.txt")
}

// BenchmarkBuildLineIndex measures the per-edit reindex cost (rebuilt on every
// mutation today; Phase 5 will make it incremental).
func BenchmarkBuildLineIndex(b *testing.B) {
	e := bigEditor(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildLineIndex(e.buf)
	}
}

// BenchmarkMoveDown measures cursor movement, which should be O(log n) via the
// cached line index and allocation-free.
func BenchmarkMoveDown(b *testing.B) {
	e := bigEditor(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if e.pos >= e.idx.Total()-50 {
			e.pos = 0
		}
		e.MoveDown()
	}
}

// BenchmarkTypeChar measures inserting a single character (includes reindex).
func BenchmarkTypeChar(b *testing.B) {
	e := bigEditor(1000)
	e.LineEnd()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.InsertRune('x')
	}
}
