// Package editor holds chiquito's framework-agnostic editing state: the text
// buffer, the cursor, the viewport, and the commands that mutate them. It has
// no knowledge of Bubble Tea or any terminal — the TUI layer (internal/ui)
// drives it — so every command here is directly unit-testable.
package editor

import (
	"strings"
	"unicode/utf8"

	"argc.dev/chiquito/internal/buffer"
)

// Editor is the editing model: a buffer plus a cursor and viewport over it.
// The cursor is stored as a single rune index (pos); line/column are derived
// from the cached LineIndex. The zero value is not usable — construct with New.
type Editor struct {
	buf  *buffer.Buffer
	idx  *LineIndex
	name string

	pos     int // cursor as a rune index, the source of truth
	goalCol int // remembered column for vertical movement

	top    int // first visible line
	width  int
	height int // number of text rows in the viewport

	tabWidth   int
	expandTabs bool
	dirty      bool
}

// New returns an Editor over the given contents, with the cursor at the start.
// name is the file path used for saving (may be empty for a scratch buffer).
func New(content []byte, name string) *Editor {
	e := &Editor{
		buf:        buffer.New(content),
		name:       name,
		height:     1,
		width:      80,
		tabWidth:   4,
		expandTabs: true,
	}
	e.reindex()
	return e
}

// SetTabStops configures tab handling used by Insert when a tab is typed.
func (e *Editor) SetTabStops(width int, expand bool) {
	if width > 0 {
		e.tabWidth = width
	}
	e.expandTabs = expand
}

func (e *Editor) reindex() { e.idx = BuildLineIndex(e.buf) }

// --- accessors -------------------------------------------------------------

// Name returns the file path backing this editor (possibly empty).
func (e *Editor) Name() string { return e.name }

// SetName sets the file path used for saving.
func (e *Editor) SetName(name string) { e.name = name }

// Dirty reports whether there are unsaved changes.
func (e *Editor) Dirty() bool { return e.dirty }

// MarkSaved clears the dirty flag after a successful save.
func (e *Editor) MarkSaved() { e.dirty = false }

// Bytes returns the current buffer contents.
func (e *Editor) Bytes() []byte { return e.buf.Bytes() }

// LineCount returns the number of lines.
func (e *Editor) LineCount() int { return e.idx.Count() }

// Line returns the contents of the given line (no trailing newline).
func (e *Editor) Line(n int) string { return e.buf.Line(n) }

// CursorLineCol returns the cursor's zero-based line and rune column.
func (e *Editor) CursorLineCol() (line, col int) { return e.idx.LineCol(e.pos) }

// CursorPos returns the cursor as a rune index.
func (e *Editor) CursorPos() int { return e.pos }

// SetCursor moves the cursor to a rune index (clamped) and keeps it visible.
func (e *Editor) SetCursor(pos int) {
	e.pos = clamp(pos, 0, e.idx.Total())
	e.syncGoal()
	e.scrollToCursor()
}

// Text returns the whole document as a string.
func (e *Editor) Text() string { return e.buf.String() }

// LineStartPos returns the rune index at which line begins.
func (e *Editor) LineStartPos(line int) int { return e.idx.Start(line) }

// PosToLineCol converts a rune index to a zero-based line and column.
func (e *Editor) PosToLineCol(pos int) (line, col int) { return e.idx.LineCol(pos) }

// Top returns the first visible line index.
func (e *Editor) Top() int { return e.top }

// Height returns the viewport height in rows.
func (e *Editor) Height() int { return e.height }

// Width returns the viewport width in columns.
func (e *Editor) Width() int { return e.width }

// SetSize sets the viewport dimensions (in text rows/columns) and keeps the
// cursor visible.
func (e *Editor) SetSize(width, height int) {
	if width > 0 {
		e.width = width
	}
	if height > 0 {
		e.height = height
	}
	e.scrollToCursor()
}

// --- movement --------------------------------------------------------------

func (e *Editor) syncGoal() {
	_, col := e.idx.LineCol(e.pos)
	e.goalCol = col
}

// MoveLeft moves the cursor one rune left, wrapping to the previous line.
func (e *Editor) MoveLeft() {
	if e.pos > 0 {
		e.pos--
	}
	e.syncGoal()
	e.scrollToCursor()
}

// MoveRight moves the cursor one rune right, wrapping to the next line.
func (e *Editor) MoveRight() {
	if e.pos < e.idx.Total() {
		e.pos++
	}
	e.syncGoal()
	e.scrollToCursor()
}

// MoveUp moves to the previous line, keeping the remembered goal column.
func (e *Editor) MoveUp() {
	line, _ := e.idx.LineCol(e.pos)
	if line > 0 {
		e.pos = e.idx.RuneIndex(line-1, e.goalCol)
	}
	e.scrollToCursor()
}

// MoveDown moves to the next line, keeping the remembered goal column.
func (e *Editor) MoveDown() {
	line, _ := e.idx.LineCol(e.pos)
	if line < e.idx.Count()-1 {
		e.pos = e.idx.RuneIndex(line+1, e.goalCol)
	}
	e.scrollToCursor()
}

// LineStart moves to the first column of the current line (Emacs C-a).
func (e *Editor) LineStart() {
	line, _ := e.idx.LineCol(e.pos)
	e.pos = e.idx.Start(line)
	e.goalCol = 0
	e.scrollToCursor()
}

// LineEnd moves to the end of the current line (Emacs C-e).
func (e *Editor) LineEnd() {
	line, _ := e.idx.LineCol(e.pos)
	e.pos = e.idx.Start(line) + e.idx.LineLen(line)
	e.goalCol = e.idx.LineLen(line)
	e.scrollToCursor()
}

// --- editing ---------------------------------------------------------------

// Insert inserts text at the cursor and advances the cursor past it.
func (e *Editor) Insert(text string) {
	if text == "" {
		return
	}
	e.buf.Insert(e.pos, text)
	e.pos += utf8.RuneCountInString(text)
	e.dirty = true
	e.reindex()
	e.syncGoal()
	e.scrollToCursor()
}

// InsertRune inserts a single rune, expanding tabs to spaces when configured.
func (e *Editor) InsertRune(r rune) {
	if r == '\t' && e.expandTabs {
		e.Insert(strings.Repeat(" ", e.tabWidth))
		return
	}
	e.Insert(string(r))
}

// InsertNewline inserts a line break at the cursor.
func (e *Editor) InsertNewline() { e.Insert("\n") }

// DeleteBackward deletes the rune before the cursor (Backspace).
func (e *Editor) DeleteBackward() {
	if e.pos == 0 {
		return
	}
	e.buf.Delete(e.pos-1, 1)
	e.pos--
	e.dirty = true
	e.reindex()
	e.syncGoal()
	e.scrollToCursor()
}

// DeleteForward deletes the rune at the cursor (Emacs C-d / Delete).
func (e *Editor) DeleteForward() {
	if e.pos >= e.idx.Total() {
		return
	}
	e.buf.Delete(e.pos, 1)
	e.dirty = true
	e.reindex()
	e.syncGoal()
	e.scrollToCursor()
}

// KillLine deletes from the cursor to the end of the line; if the cursor is
// already at the end of the line, it deletes the newline, joining the next line
// (Emacs C-k).
func (e *Editor) KillLine() {
	line, col := e.idx.LineCol(e.pos)
	llen := e.idx.LineLen(line)
	if col < llen {
		e.buf.Delete(e.pos, llen-col)
	} else if e.pos < e.idx.Total() {
		e.buf.Delete(e.pos, 1) // delete the newline
	} else {
		return
	}
	e.dirty = true
	e.reindex()
	e.syncGoal()
	e.scrollToCursor()
}

// Replace replaces the rune range [start, end) with text and places the cursor
// just after the inserted text. The range is clamped to the document.
func (e *Editor) Replace(start, end int, text string) {
	start = clamp(start, 0, e.idx.Total())
	end = clamp(end, start, e.idx.Total())
	if end > start {
		e.buf.Delete(start, end-start)
	}
	if text != "" {
		e.buf.Insert(start, text)
	}
	e.pos = start + utf8.RuneCountInString(text)
	e.dirty = true
	e.reindex()
	e.syncGoal()
	e.scrollToCursor()
}

// SetText replaces the entire document, clamping the cursor into the new bounds.
// Used by replace-all.
func (e *Editor) SetText(s string) {
	e.buf = buffer.New([]byte(s))
	e.dirty = true
	e.reindex()
	e.pos = clamp(e.pos, 0, e.idx.Total())
	e.syncGoal()
	e.scrollToCursor()
}

// --- viewport --------------------------------------------------------------

func (e *Editor) scrollToCursor() {
	line, _ := e.idx.LineCol(e.pos)
	if line < e.top {
		e.top = line
	}
	if e.height > 0 && line >= e.top+e.height {
		e.top = line - e.height + 1
	}
	if e.top < 0 {
		e.top = 0
	}
}
