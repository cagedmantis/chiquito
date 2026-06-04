package buffer

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzInsert checks that an insert matches a reference string splice and that
// the cached rune and line totals stay consistent for arbitrary Unicode input.
func FuzzInsert(f *testing.F) {
	f.Add("hello world", "XYZ", uint(5))
	f.Add("", "abc", uint(0))
	f.Add("a\nb\nc", "\n\n", uint(3))
	f.Add("héllo 世界", "🌍", uint(2))

	f.Fuzz(func(t *testing.T, orig, ins string, posSeed uint) {
		if !utf8.ValidString(orig) || !utf8.ValidString(ins) {
			t.Skip()
		}
		origRunes := []rune(orig)
		pos := int(posSeed) % (len(origRunes) + 1)

		b := New([]byte(orig))
		b.Insert(pos, ins)

		want := string(origRunes[:pos]) + ins + string(origRunes[pos:])
		if got := b.String(); got != want {
			t.Fatalf("Insert mismatch:\n got %q\nwant %q", got, want)
		}
		if b.Len() != utf8.RuneCountInString(want) {
			t.Errorf("Len = %d, want %d", b.Len(), utf8.RuneCountInString(want))
		}
		if b.LineCount() != strings.Count(want, "\n")+1 {
			t.Errorf("LineCount = %d, want %d", b.LineCount(), strings.Count(want, "\n")+1)
		}
	})
}

// FuzzInsertDelete checks that deleting exactly what was inserted restores the
// original document — a round-trip invariant the piece table must preserve.
func FuzzInsertDelete(f *testing.F) {
	f.Add("hello", "INSERTED", uint(2))
	f.Add("a\nb", "\nx\n", uint(1))

	f.Fuzz(func(t *testing.T, orig, ins string, posSeed uint) {
		if !utf8.ValidString(orig) || !utf8.ValidString(ins) || ins == "" {
			t.Skip()
		}
		pos := int(posSeed) % (utf8.RuneCountInString(orig) + 1)

		b := New([]byte(orig))
		b.Insert(pos, ins)
		b.Delete(pos, utf8.RuneCountInString(ins))

		if got := b.String(); got != orig {
			t.Fatalf("insert+delete did not round-trip:\n got %q\nwant %q", got, orig)
		}
		if b.LineCount() != strings.Count(orig, "\n")+1 {
			t.Errorf("LineCount = %d, want %d", b.LineCount(), strings.Count(orig, "\n")+1)
		}
	})
}

// FuzzLineStartRunes checks the piece-table line scan against a straightforward
// reference computed from the materialized bytes.
func FuzzLineStartRunes(f *testing.F) {
	f.Add("a\nbb\nccc", "x\n", uint(2))
	f.Add("no newlines here", "", uint(0))

	f.Fuzz(func(t *testing.T, orig, ins string, posSeed uint) {
		if !utf8.ValidString(orig) || !utf8.ValidString(ins) {
			t.Skip()
		}
		b := New([]byte(orig))
		if ins != "" {
			b.Insert(int(posSeed)%(b.Len()+1), ins)
		}

		starts, total := b.LineStartRunes(nil)
		if total != b.Len() {
			t.Fatalf("total = %d, want %d", total, b.Len())
		}
		// Reference: line starts are rune 0 plus the rune index after each '\n'.
		want := []int{0}
		r := 0
		for _, c := range b.String() {
			r++
			if c == '\n' {
				want = append(want, r)
			}
		}
		if len(starts) != len(want) {
			t.Fatalf("line count = %d, want %d", len(starts), len(want))
		}
		for i := range want {
			if starts[i] != want[i] {
				t.Fatalf("start[%d] = %d, want %d", i, starts[i], want[i])
			}
		}
	})
}
