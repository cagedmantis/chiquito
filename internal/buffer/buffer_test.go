package buffer

import (
	"strings"
	"testing"
)

func TestNewEmpty(t *testing.T) {
	b := New(nil)
	if b.Len() != 0 {
		t.Errorf("Len() = %d, want 0", b.Len())
	}
	if b.LineCount() != 1 {
		t.Errorf("LineCount() = %d, want 1", b.LineCount())
	}
	if got := b.String(); got != "" {
		t.Errorf("String() = %q, want empty", got)
	}
}

func TestNewFromBytes(t *testing.T) {
	b := New([]byte("hello\nworld"))
	if b.Len() != 11 {
		t.Errorf("Len() = %d, want 11", b.Len())
	}
	if b.LineCount() != 2 {
		t.Errorf("LineCount() = %d, want 2", b.LineCount())
	}
	if got := b.String(); got != "hello\nworld" {
		t.Errorf("String() = %q", got)
	}
}

func TestInsert(t *testing.T) {
	tests := []struct {
		name  string
		start string
		idx   int
		text  string
		want  string
	}{
		{"into empty", "", 0, "abc", "abc"},
		{"at start", "world", 0, "hello ", "hello world"},
		{"at end", "hello", 5, " world", "hello world"},
		{"in middle", "held", 2, "l", "helld"},
		{"clamp high", "ab", 99, "c", "abc"},
		{"clamp low", "bc", -5, "a", "abc"},
		{"empty noop", "abc", 1, "", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New([]byte(tt.start))
			b.Insert(tt.idx, tt.text)
			if got := b.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
			if b.Len() != len([]rune(tt.want)) {
				t.Errorf("Len() = %d, want %d", b.Len(), len([]rune(tt.want)))
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		start      string
		idx, count int
		want       string
	}{
		{"middle", "hello world", 5, 6, "hello"},
		{"start", "hello", 0, 2, "llo"},
		{"all", "hello", 0, 5, ""},
		{"clamp count", "hello", 3, 99, "hel"},
		{"negative count noop", "hello", 1, -1, "hello"},
		{"out of range noop", "hello", 99, 1, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New([]byte(tt.start))
			b.Delete(tt.idx, tt.count)
			if got := b.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInsertDeleteSequence(t *testing.T) {
	b := New([]byte("The quick brown fox"))
	b.Insert(19, " jumps") // "The quick brown fox jumps"
	b.Insert(0, ">> ")     // ">> The quick brown fox jumps"
	b.Delete(3, 4)         // remove "The " -> ">> quick brown fox jumps"
	want := ">> quick brown fox jumps"
	if got := b.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	// Newline accounting must survive edits.
	b.Insert(2, "\nX\n")
	if b.LineCount() != 3 {
		t.Errorf("LineCount() = %d, want 3", b.LineCount())
	}
}

func TestUnicode(t *testing.T) {
	b := New([]byte("héllo, 世界"))
	if b.Len() != 9 { // h é l l o , space 世 界
		t.Fatalf("Len() = %d, want 9", b.Len())
	}
	// Insert an emoji between the two CJK runes.
	b.Insert(8, "🌍")
	want := "héllo, 世🌍界"
	if got := b.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if r, ok := b.RuneAt(8); !ok || r != '🌍' {
		t.Errorf("RuneAt(8) = %q, %v; want 🌍, true", r, ok)
	}
	// Delete the emoji again.
	b.Delete(8, 1)
	if got := b.String(); got != "héllo, 世界" {
		t.Errorf("after delete String() = %q", got)
	}
}

func TestRuneAt(t *testing.T) {
	b := New([]byte("abc"))
	b.Insert(1, "XY") // aXYbc
	for i, want := range []rune("aXYbc") {
		if r, ok := b.RuneAt(i); !ok || r != want {
			t.Errorf("RuneAt(%d) = %q,%v want %q", i, r, ok, want)
		}
	}
	if _, ok := b.RuneAt(5); ok {
		t.Error("RuneAt(5) should be out of range")
	}
	if _, ok := b.RuneAt(-1); ok {
		t.Error("RuneAt(-1) should be out of range")
	}
}

func TestLine(t *testing.T) {
	b := New([]byte("first\nsecond\nthird"))
	cases := map[int]string{0: "first", 1: "second", 2: "third", 3: "", -1: ""}
	for line, want := range cases {
		if got := b.Line(line); got != want {
			t.Errorf("Line(%d) = %q, want %q", line, got, want)
		}
	}
	// A line whose content spans several inserts.
	b2 := New([]byte("ab"))
	b2.Insert(1, "XY") // line 0 = "aXYb"
	b2.Insert(4, "\ntail")
	if got := b2.Line(0); got != "aXYb" {
		t.Errorf("Line(0) = %q, want aXYb", got)
	}
	if got := b2.Line(1); got != "tail" {
		t.Errorf("Line(1) = %q, want tail", got)
	}
}

func TestLineColRoundTrip(t *testing.T) {
	b := New([]byte("ab\ncdef\n\nghi"))
	cases := []struct {
		idx       int
		line, col int
	}{
		{0, 0, 0},
		{2, 0, 2},  // the newline at end of line 0
		{3, 1, 0},  // 'c'
		{7, 1, 4},  // newline at end of "cdef"
		{8, 2, 0},  // the empty line
		{9, 3, 0},  // 'g'
		{12, 3, 3}, // end of buffer
	}
	for _, c := range cases {
		l, col := b.RuneToLineCol(c.idx)
		if l != c.line || col != c.col {
			t.Errorf("RuneToLineCol(%d) = (%d,%d), want (%d,%d)", c.idx, l, col, c.line, c.col)
		}
		if got := b.LineColToRune(c.line, c.col); got != c.idx {
			t.Errorf("LineColToRune(%d,%d) = %d, want %d", c.line, c.col, got, c.idx)
		}
	}
	// Column past end of line clamps to the newline position.
	if got := b.LineColToRune(0, 99); got != 2 {
		t.Errorf("LineColToRune(0,99) = %d, want 2", got)
	}
}

func TestTypingCoalesces(t *testing.T) {
	b := New(nil)
	for _, r := range "hello world" {
		b.Insert(b.Len(), string(r))
	}
	if got := b.String(); got != "hello world" {
		t.Fatalf("String() = %q", got)
	}
	// Sequential appends should collapse into a single piece, not 11.
	if n := b.numPieces(); n != 1 {
		t.Errorf("numPieces() = %d, want 1 (coalesced)", n)
	}
}

func TestInvalidUTF8RoundTrips(t *testing.T) {
	raw := []byte{0xff, 'a', 0xfe, '\n', 'b'}
	b := New(raw)
	if got := b.Bytes(); string(got) != string(raw) {
		t.Errorf("Bytes() = %v, want %v", got, raw)
	}
	if b.LineCount() != 2 {
		t.Errorf("LineCount() = %d, want 2", b.LineCount())
	}
}

func TestLargeBuildConsistency(t *testing.T) {
	// Build a moderately large document via repeated middle inserts and verify
	// totals match a ground-truth string.
	var want strings.Builder
	b := New(nil)
	for i := 0; i < 500; i++ {
		s := "line\n"
		b.Insert(b.Len(), s)
		want.WriteString(s)
	}
	// Insert in the middle. The document is ASCII, so rune index == byte index.
	mid := b.Len() / 2
	b.Insert(mid, "MIDDLE")
	ws := want.String()
	ws = ws[:mid] + "MIDDLE" + ws[mid:]
	if got := b.String(); got != ws {
		t.Errorf("large build mismatch:\n got len %d\nwant len %d", len(got), len(ws))
	}
	if b.LineCount() != strings.Count(ws, "\n")+1 {
		t.Errorf("LineCount() = %d, want %d", b.LineCount(), strings.Count(ws, "\n")+1)
	}
}
