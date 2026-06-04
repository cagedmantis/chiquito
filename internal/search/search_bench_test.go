package search

import (
	"strings"
	"testing"
)

func corpus(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("the quick brown fox jumps over the lazy dog\n")
	}
	// One needle near the end so search has to scan.
	sb.WriteString("find the needle here\n")
	return sb.String()
}

func BenchmarkFindAllCaseSensitive(b *testing.B) {
	text := corpus(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FindAll(text, "needle", Options{CaseSensitive: true})
	}
}

func BenchmarkFindAllCaseInsensitive(b *testing.B) {
	text := corpus(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FindAll(text, "needle", Options{CaseSensitive: false})
	}
}

func BenchmarkReplaceAll(b *testing.B) {
	text := corpus(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ReplaceAll(text, "fox", "cat", Options{CaseSensitive: true})
	}
}
