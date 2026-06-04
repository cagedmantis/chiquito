package spell

import (
	"strings"
	"testing"
)

func BenchmarkCheck(b *testing.B) {
	d := Load()
	var sb strings.Builder
	for i := 0; i < 2000; i++ {
		sb.WriteString("the quick brown fox jumps over the lazy dog ")
	}
	text := sb.String()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Check(text, d)
	}
}

func BenchmarkContains(b *testing.B) {
	d := Load()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Contains("editor")
	}
}
