package buffer

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// Line returns the contents of the given zero-based line, excluding the
// trailing newline. Out-of-range lines return "".
//
// The scan stops as soon as the target line has been emitted, so reading the
// first lines of a large file is cheap. (A cached line index for O(log n)
// random line access is a Phase 5 optimization.)
func (b *Buffer) Line(line int) string {
	if line < 0 || line > b.totalNewlines {
		return ""
	}
	var sb strings.Builder
	cur := 0
	for _, p := range b.pieces {
		if cur > line {
			break
		}
		data := b.pieceBytes(p)
		if p.newlines == 0 {
			if cur == line {
				sb.Write(data)
			}
			continue
		}
		for len(data) > 0 {
			i := bytes.IndexByte(data, '\n')
			if i < 0 {
				if cur == line {
					sb.Write(data)
				}
				break
			}
			if cur == line {
				sb.Write(data[:i])
				return sb.String()
			}
			cur++
			data = data[i+1:]
			if cur > line {
				break
			}
		}
	}
	return sb.String()
}

// RuneToLineCol converts a rune index to a zero-based (line, column) pair where
// column is measured in runes from the start of the line. idx is clamped to
// [0, Len()].
//
// This walks the document once (O(n)); Phase 2 introduces a cached line index
// for cursor movement.
func (b *Buffer) RuneToLineCol(idx int) (line, col int) {
	if idx < 0 {
		idx = 0
	}
	if idx > b.totalRunes {
		idx = b.totalRunes
	}
	data := b.Bytes()
	off, r := 0, 0
	for off < len(data) && r < idx {
		c, size := utf8.DecodeRune(data[off:])
		if c == '\n' {
			line++
			col = 0
		} else {
			col++
		}
		off += size
		r++
	}
	return line, col
}

// LineColToRune converts a zero-based (line, column) pair to a rune index,
// clamping the column to the end of the line and the line to the document.
func (b *Buffer) LineColToRune(line, col int) int {
	if line < 0 {
		return 0
	}
	if col < 0 {
		col = 0
	}
	data := b.Bytes()
	off, r := 0, 0
	curLine := 0
	for off < len(data) && curLine < line {
		c, size := utf8.DecodeRune(data[off:])
		if c == '\n' {
			curLine++
		}
		off += size
		r++
	}
	curCol := 0
	for off < len(data) && curCol < col {
		c, size := utf8.DecodeRune(data[off:])
		if c == '\n' {
			break
		}
		off += size
		r++
		curCol++
	}
	return r
}
