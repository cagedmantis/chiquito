package editor

import (
	"strings"
	"testing"
)

func TestNoTrailingNewline(t *testing.T) {
	e := newEd("abc")
	if e.LineCount() != 1 {
		t.Errorf("LineCount = %d, want 1", e.LineCount())
	}
	// A trailing newline implies a final empty line.
	e2 := newEd("abc\n")
	if e2.LineCount() != 2 {
		t.Errorf("LineCount = %d, want 2", e2.LineCount())
	}
	if got := e2.Line(1); got != "" {
		t.Errorf("Line(1) = %q, want empty", got)
	}
	// The cursor can sit on the trailing empty line.
	e2.SetCursor(e2.LineColToRuneViaEnd())
	if l, _ := e2.CursorLineCol(); l != 1 {
		t.Errorf("cursor line = %d, want 1", l)
	}
}

// LineColToRuneViaEnd returns the rune index of the document end (test helper).
func (e *Editor) LineColToRuneViaEnd() int { return e.idx.Total() }

func TestCRLFPreserved(t *testing.T) {
	// chiquito splits on '\n' and preserves '\r' bytes (no normalization).
	e := newEd("a\r\nb\r\nc")
	if e.LineCount() != 3 {
		t.Errorf("LineCount = %d, want 3", e.LineCount())
	}
	if got := e.Line(0); got != "a\r" {
		t.Errorf("Line(0) = %q, want %q", got, "a\r")
	}
	// Round-trips byte-exact.
	if got := string(e.Bytes()); got != "a\r\nb\r\nc" {
		t.Errorf("round trip = %q", got)
	}
}

func TestVeryLongLine(t *testing.T) {
	long := strings.Repeat("x", 50000)
	e := newEd(long)
	if e.idx.Total() != 50000 {
		t.Fatalf("Len = %d, want 50000", e.idx.Total())
	}
	e.LineEnd()
	if _, c := e.CursorLineCol(); c != 50000 {
		t.Errorf("LineEnd col = %d, want 50000", c)
	}
	// Insert in the middle of the long line.
	e.SetCursor(25000)
	e.Insert("|")
	if got := e.Line(0); got[25000] != '|' {
		t.Error("insert into long line failed")
	}
	if e.LineCount() != 1 {
		t.Errorf("LineCount = %d, want 1", e.LineCount())
	}
}

func TestEmptyBufferOps(t *testing.T) {
	e := newEd("")
	e.MoveLeft()
	e.MoveRight()
	e.MoveUp()
	e.MoveDown()
	e.DeleteBackward()
	e.DeleteForward()
	e.KillLine()
	if e.idx.Total() != 0 || e.LineCount() != 1 {
		t.Errorf("empty buffer ops changed state: len=%d lines=%d", e.idx.Total(), e.LineCount())
	}
}

func TestManyLinesScrollClamp(t *testing.T) {
	e := newEd(strings.Repeat("line\n", 10))
	e.SetSize(20, 5)
	// Move down well past the end; cursor and top must stay in bounds.
	for i := 0; i < 100; i++ {
		e.MoveDown()
	}
	line, _ := e.CursorLineCol()
	if line > e.LineCount()-1 {
		t.Errorf("cursor line %d exceeds last line %d", line, e.LineCount()-1)
	}
	if e.Top() < 0 || e.Top() >= e.LineCount() {
		t.Errorf("top %d out of bounds", e.Top())
	}
}
