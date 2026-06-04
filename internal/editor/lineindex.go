package editor

import (
	"sort"

	"argc.dev/chiquito/internal/buffer"
)

// LineIndex caches, for a snapshot of a buffer, the rune offset at which each
// line begins. It turns line/column ↔ rune-index conversions and cursor
// movement into O(log n) (binary search) or O(1) lookups, which a raw piece
// table cannot offer cheaply.
//
// The index is immutable; rebuild it with BuildLineIndex after any edit. (The
// editor rebuilds on edit; incremental maintenance is a Phase 5 optimization.)
type LineIndex struct {
	starts []int // starts[i] = rune index of the first rune of line i
	total  int   // total rune count
}

// BuildLineIndex records the start of every line by walking the buffer's piece
// table (without copying the whole document).
func BuildLineIndex(b *buffer.Buffer) *LineIndex {
	starts, total := b.LineStartRunes(make([]int, 0, b.LineCount()))
	return &LineIndex{starts: starts, total: total}
}

// Count returns the number of lines (always >= 1).
func (li *LineIndex) Count() int { return len(li.starts) }

// Total returns the total number of runes in the indexed buffer.
func (li *LineIndex) Total() int { return li.total }

// Start returns the rune index at which line begins. line is clamped.
func (li *LineIndex) Start(line int) int {
	line = clamp(line, 0, len(li.starts)-1)
	return li.starts[line]
}

// LineLen returns the number of runes on line, excluding the trailing newline.
func (li *LineIndex) LineLen(line int) int {
	line = clamp(line, 0, len(li.starts)-1)
	start := li.starts[line]
	end := li.total
	if line+1 < len(li.starts) {
		end = li.starts[line+1] - 1 // drop the '\n'
	}
	return end - start
}

// RuneIndex converts a (line, col) pair to a rune index, clamping line to the
// document and col to the end of that line.
func (li *LineIndex) RuneIndex(line, col int) int {
	line = clamp(line, 0, len(li.starts)-1)
	col = clamp(col, 0, li.LineLen(line))
	return li.starts[line] + col
}

// LineCol converts a rune index to a (line, col) pair. pos is clamped to
// [0, total].
func (li *LineIndex) LineCol(pos int) (line, col int) {
	pos = clamp(pos, 0, li.total)
	// Largest i with starts[i] <= pos.
	line = sort.Search(len(li.starts), func(i int) bool { return li.starts[i] > pos }) - 1
	if line < 0 {
		line = 0
	}
	return line, pos - li.starts[line]
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
