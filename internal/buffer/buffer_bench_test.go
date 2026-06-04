package buffer

import (
	"strings"
	"testing"
)

func makeLargeText(lines int) []byte {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		sb.WriteString("the quick brown fox jumps over the lazy dog\n")
	}
	return []byte(sb.String())
}

// BenchmarkInsertAppend measures the steady-state cost of typing at the end of
// the buffer (coalesced appends).
func BenchmarkInsertAppend(b *testing.B) {
	buf := New(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Insert(buf.Len(), "x")
	}
}

// BenchmarkInsertMiddle measures inserts that force a piece split each time.
func BenchmarkInsertMiddle(b *testing.B) {
	buf := New(makeLargeText(1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Insert(buf.Len()/2, "z")
	}
}

func BenchmarkDeleteMiddle(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf := New(makeLargeText(200))
		mid := buf.Len() / 2
		b.StartTimer()
		buf.Delete(mid, 10)
	}
}

// BenchmarkBytes measures materialising a large document.
func BenchmarkBytes(b *testing.B) {
	buf := New(makeLargeText(5000))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Bytes()
	}
}

// BenchmarkLineHead measures reading an early line from a large document.
func BenchmarkLineHead(b *testing.B) {
	buf := New(makeLargeText(10000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Line(5)
	}
}

func BenchmarkRuneToLineCol(b *testing.B) {
	buf := New(makeLargeText(1000))
	idx := buf.Len() / 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buf.RuneToLineCol(idx)
	}
}
