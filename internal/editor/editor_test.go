package editor

import "testing"

func newEd(content string) *Editor {
	return New([]byte(content), "test.txt")
}

func TestMovementHorizontal(t *testing.T) {
	e := newEd("abc\ndef")
	e.MoveRight()
	e.MoveRight()
	if l, c := e.CursorLineCol(); l != 0 || c != 2 {
		t.Fatalf("after 2x right: (%d,%d), want (0,2)", l, c)
	}
	// Right past end of line 0 wraps to start of line 1.
	e.MoveRight() // col 3 (end of "abc")
	e.MoveRight() // wrap to (1,0)
	if l, c := e.CursorLineCol(); l != 1 || c != 0 {
		t.Fatalf("wrap forward: (%d,%d), want (1,0)", l, c)
	}
	// Left from (1,0) wraps back to end of line 0.
	e.MoveLeft()
	if l, c := e.CursorLineCol(); l != 0 || c != 3 {
		t.Fatalf("wrap backward: (%d,%d), want (0,3)", l, c)
	}
}

func TestMovementVerticalGoalColumn(t *testing.T) {
	e := newEd("hello\nhi\nworld")
	// Put cursor at column 4 on line 0.
	for i := 0; i < 4; i++ {
		e.MoveRight()
	}
	// Down onto the short line "hi" (len 2) clamps to col 2...
	e.MoveDown()
	if l, c := e.CursorLineCol(); l != 1 || c != 2 {
		t.Fatalf("down to short line: (%d,%d), want (1,2)", l, c)
	}
	// ...but the goal column is remembered, so moving down again restores col 4.
	e.MoveDown()
	if l, c := e.CursorLineCol(); l != 2 || c != 4 {
		t.Fatalf("goal column not preserved: (%d,%d), want (2,4)", l, c)
	}
}

func TestLineStartEnd(t *testing.T) {
	e := newEd("hello world\nsecond")
	e.MoveRight()
	e.MoveRight()
	e.LineEnd()
	if _, c := e.CursorLineCol(); c != 11 {
		t.Fatalf("LineEnd col = %d, want 11", c)
	}
	e.LineStart()
	if _, c := e.CursorLineCol(); c != 0 {
		t.Fatalf("LineStart col = %d, want 0", c)
	}
}

func TestInsertAndNewline(t *testing.T) {
	e := newEd("")
	e.Insert("hi")
	e.InsertNewline()
	e.Insert("there")
	if got := string(e.Bytes()); got != "hi\nthere" {
		t.Fatalf("Bytes = %q, want %q", got, "hi\nthere")
	}
	if l, c := e.CursorLineCol(); l != 1 || c != 5 {
		t.Fatalf("cursor = (%d,%d), want (1,5)", l, c)
	}
	if !e.Dirty() {
		t.Error("editor should be dirty after insert")
	}
}

func TestInsertRuneExpandsTabs(t *testing.T) {
	e := newEd("")
	e.SetTabStops(4, true)
	e.InsertRune('\t')
	if got := string(e.Bytes()); got != "    " {
		t.Fatalf("tab expansion = %q, want 4 spaces", got)
	}
	e2 := newEd("")
	e2.SetTabStops(4, false)
	e2.InsertRune('\t')
	if got := string(e2.Bytes()); got != "\t" {
		t.Fatalf("literal tab = %q, want \\t", got)
	}
}

func TestDeleteBackward(t *testing.T) {
	e := newEd("abc")
	e.LineEnd()
	e.DeleteBackward()
	if got := string(e.Bytes()); got != "ab" {
		t.Fatalf("Bytes = %q, want ab", got)
	}
	if _, c := e.CursorLineCol(); c != 2 {
		t.Fatalf("cursor col = %d, want 2", c)
	}
	// Backspace at start of buffer is a no-op.
	e.LineStart()
	e.DeleteBackward()
	if got := string(e.Bytes()); got != "ab" {
		t.Fatalf("no-op delete changed buffer: %q", got)
	}
}

func TestDeleteBackwardJoinsLines(t *testing.T) {
	e := newEd("ab\ncd")
	e.MoveDown() // (1,0)... goal col 0, but start at (0,0) then down -> (1,0)
	e.LineStart()
	e.DeleteBackward() // remove the newline, join
	if got := string(e.Bytes()); got != "abcd" {
		t.Fatalf("Bytes = %q, want abcd", got)
	}
	if l, c := e.CursorLineCol(); l != 0 || c != 2 {
		t.Fatalf("cursor = (%d,%d), want (0,2)", l, c)
	}
}

func TestDeleteForward(t *testing.T) {
	e := newEd("abc")
	e.DeleteForward()
	if got := string(e.Bytes()); got != "bc" {
		t.Fatalf("Bytes = %q, want bc", got)
	}
	e.LineEnd()
	e.DeleteForward() // at end: no-op
	if got := string(e.Bytes()); got != "bc" {
		t.Fatalf("Bytes = %q, want bc", got)
	}
}

func TestKillLine(t *testing.T) {
	e := newEd("hello world\nnext")
	for i := 0; i < 5; i++ {
		e.MoveRight() // cursor after "hello"
	}
	e.KillLine() // removes " world"
	if got := e.Line(0); got != "hello" {
		t.Fatalf("Line(0) = %q, want hello", got)
	}
	// Now at end of line 0; KillLine again joins next line.
	e.KillLine()
	if got := string(e.Bytes()); got != "hellonext" {
		t.Fatalf("Bytes = %q, want hellonext", got)
	}
}

func TestUnicodeMovement(t *testing.T) {
	e := newEd("a世🌍b")
	e.MoveRight() // after 'a'
	e.MoveRight() // after '世'
	if _, c := e.CursorLineCol(); c != 2 {
		t.Fatalf("col = %d, want 2", c)
	}
	e.Insert("X")
	if got := string(e.Bytes()); got != "a世X🌍b" {
		t.Fatalf("Bytes = %q", got)
	}
}

func TestScrolling(t *testing.T) {
	var sb []byte
	for i := 0; i < 100; i++ {
		sb = append(sb, []byte("line\n")...)
	}
	e := New(sb, "big.txt")
	e.SetSize(80, 10)
	if e.Top() != 0 {
		t.Fatalf("initial top = %d, want 0", e.Top())
	}
	// Move down past the viewport; top should follow.
	for i := 0; i < 15; i++ {
		e.MoveDown()
	}
	line, _ := e.CursorLineCol()
	if line != 15 {
		t.Fatalf("cursor line = %d, want 15", line)
	}
	if e.Top() != 6 { // line - height + 1 = 15 - 10 + 1
		t.Fatalf("top = %d, want 6", e.Top())
	}
	// Scroll back up.
	for i := 0; i < 15; i++ {
		e.MoveUp()
	}
	if e.Top() != 0 {
		t.Fatalf("top after scroll up = %d, want 0", e.Top())
	}
}
