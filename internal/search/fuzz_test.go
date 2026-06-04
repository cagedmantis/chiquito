package search

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzFindAll checks core invariants: matches are ordered, non-overlapping, in
// range, and the matched text equals the query; and for case-sensitive search
// the count agrees with strings.Count.
func FuzzFindAll(f *testing.F) {
	f.Add("ababab", "ab")
	f.Add("the cat sat", "at")
	f.Add("héllo 世界 héllo", "héllo")
	f.Add("aaaa", "aa")

	f.Fuzz(func(t *testing.T, text, query string) {
		if !utf8.ValidString(text) || !utf8.ValidString(query) || query == "" {
			t.Skip()
		}
		matches := FindAll(text, query, Options{CaseSensitive: true})
		runes := []rune(text)
		qlen := utf8.RuneCountInString(query)

		prevEnd := -1
		for _, mt := range matches {
			if mt.Start < 0 || mt.End > len(runes) || mt.End != mt.Start+qlen {
				t.Fatalf("bad span %+v (len text=%d, qlen=%d)", mt, len(runes), qlen)
			}
			if mt.Start < prevEnd {
				t.Fatalf("overlapping/unordered match %+v after end %d", mt, prevEnd)
			}
			if got := string(runes[mt.Start:mt.End]); got != query {
				t.Fatalf("matched %q, want %q", got, query)
			}
			prevEnd = mt.End
		}
		if got := strings.Count(text, query); got != len(matches) {
			t.Fatalf("FindAll found %d, strings.Count found %d", len(matches), got)
		}
	})
}

// FuzzReplaceAll checks that replacing a query with itself never changes the
// text, and that the reported count matches FindAll.
func FuzzReplaceAll(f *testing.F) {
	f.Add("hello hello", "hello", "world")
	f.Add("a.b.c", ".", "::")

	f.Fuzz(func(t *testing.T, text, query, repl string) {
		if !utf8.ValidString(text) || !utf8.ValidString(query) || !utf8.ValidString(repl) || query == "" {
			t.Skip()
		}
		opts := Options{CaseSensitive: true}

		identity, n := ReplaceAll(text, query, query, opts)
		if identity != text {
			t.Fatalf("replacing query with itself changed text:\n got %q\nwant %q", identity, text)
		}
		if n != len(FindAll(text, query, opts)) {
			t.Fatalf("ReplaceAll count %d != FindAll count %d", n, len(FindAll(text, query, opts)))
		}
		// A real replacement must not panic and must round-trip rune-validly.
		out, _ := ReplaceAll(text, query, repl, opts)
		if !utf8.ValidString(out) {
			t.Fatalf("ReplaceAll produced invalid UTF-8")
		}
	})
}
