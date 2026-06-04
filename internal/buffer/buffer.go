// Package buffer implements chiquito's editable text store.
//
// The buffer is a piece table: the original file contents are kept in a
// read-only "original" slice, all inserted text is appended to a grow-only
// "added" slice, and the visible document is described by an ordered list of
// pieces that reference byte ranges within those two slices. This gives
// amortised O(1) appends, cheap inserts/deletes near existing edits, and—
// crucially for large files—means the original bytes are never copied.
//
// Every offset exposed by the public API is a rune index (or a line/column
// pair). Internally each piece caches its rune and newline counts, so rune-
// and line-oriented navigation does not require rescanning the whole document.
// Invalid UTF-8 bytes are counted as one rune each so that a load→save cycle is
// byte-exact.
package buffer

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// source identifies which underlying slice a piece reads from.
type source uint8

const (
	srcOriginal source = iota
	srcAdded
)

// piece describes a contiguous run of bytes within one of the buffers.
type piece struct {
	src      source
	off      int // byte offset into the source slice
	length   int // byte length
	runes    int // cached rune count of the run
	newlines int // cached count of '\n' bytes in the run
}

// Buffer is a piece-table text store. The zero value is not usable; create one
// with New. A Buffer is not safe for concurrent mutation.
type Buffer struct {
	original []byte
	added    []byte
	pieces   []piece

	totalRunes    int
	totalNewlines int
}

var newline = []byte{'\n'}

// New returns a Buffer whose contents are the supplied bytes. The slice is
// retained as the read-only original and is never modified or copied.
func New(original []byte) *Buffer {
	b := &Buffer{original: original}
	if len(original) > 0 {
		p := piece{
			src:      srcOriginal,
			off:      0,
			length:   len(original),
			runes:    utf8.RuneCount(original),
			newlines: bytes.Count(original, newline),
		}
		b.pieces = append(b.pieces, p)
		b.totalRunes = p.runes
		b.totalNewlines = p.newlines
	}
	return b
}

// Len returns the number of runes in the buffer.
func (b *Buffer) Len() int { return b.totalRunes }

// LineCount returns the number of lines, i.e. one more than the number of
// newline characters. An empty buffer has one (empty) line.
func (b *Buffer) LineCount() int { return b.totalNewlines + 1 }

// Bytes returns a copy of the buffer contents as UTF-8 bytes.
func (b *Buffer) Bytes() []byte {
	out := make([]byte, 0, b.byteLen())
	for _, p := range b.pieces {
		out = append(out, b.pieceBytes(p)...)
	}
	return out
}

// String returns the buffer contents as a string.
func (b *Buffer) String() string { return string(b.Bytes()) }

func (b *Buffer) byteLen() int {
	n := 0
	for _, p := range b.pieces {
		n += p.length
	}
	return n
}

func (b *Buffer) srcBytes(s source) []byte {
	if s == srcAdded {
		return b.added
	}
	return b.original
}

func (b *Buffer) pieceBytes(p piece) []byte {
	return b.srcBytes(p.src)[p.off : p.off+p.length]
}

// runeByteOffset returns the byte offset of the r-th rune in data. r must be in
// [0, RuneCount(data)].
func runeByteOffset(data []byte, r int) int {
	off := 0
	for i := 0; i < r && off < len(data); i++ {
		_, size := utf8.DecodeRune(data[off:])
		off += size
	}
	return off
}

// splitAt ensures a piece boundary exists exactly before rune index idx and
// returns the index in b.pieces of the piece beginning at idx (== len(pieces)
// if idx is at end). idx must be in [0, totalRunes]. No zero-length pieces are
// produced.
func (b *Buffer) splitAt(idx int) int {
	if idx <= 0 {
		return 0
	}
	acc := 0
	for i := 0; i < len(b.pieces); i++ {
		p := b.pieces[i]
		if acc == idx {
			return i
		}
		if acc+p.runes > idx {
			r := idx - acc
			data := b.pieceBytes(p)
			bo := runeByteOffset(data, r)
			leftNL := bytes.Count(data[:bo], newline)
			left := piece{src: p.src, off: p.off, length: bo, runes: r, newlines: leftNL}
			right := piece{src: p.src, off: p.off + bo, length: p.length - bo, runes: p.runes - r, newlines: p.newlines - leftNL}
			b.pieces = append(b.pieces, piece{})
			copy(b.pieces[i+1:], b.pieces[i:])
			b.pieces[i] = left
			b.pieces[i+1] = right
			return i + 1
		}
		acc += p.runes
	}
	return len(b.pieces)
}

// Insert inserts text at rune index idx. idx is clamped to [0, Len()].
func (b *Buffer) Insert(idx int, text string) {
	if text == "" {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx > b.totalRunes {
		idx = b.totalRunes
	}

	off := len(b.added)
	b.added = append(b.added, text...)
	np := piece{
		src:      srcAdded,
		off:      off,
		length:   len(text),
		runes:    utf8.RuneCountInString(text),
		newlines: strings.Count(text, "\n"),
	}

	pos := b.splitAt(idx)

	// Coalesce with the preceding piece when this insert continues an append to
	// the end of the added slice — the common case while typing. This keeps the
	// piece count from growing one-per-keystroke.
	if pos > 0 {
		prev := &b.pieces[pos-1]
		if prev.src == srcAdded && prev.off+prev.length == np.off {
			prev.length += np.length
			prev.runes += np.runes
			prev.newlines += np.newlines
			b.totalRunes += np.runes
			b.totalNewlines += np.newlines
			return
		}
	}

	b.pieces = append(b.pieces, piece{})
	copy(b.pieces[pos+1:], b.pieces[pos:])
	b.pieces[pos] = np
	b.totalRunes += np.runes
	b.totalNewlines += np.newlines
}

// Delete removes count runes starting at rune index idx. The range is clamped
// to the buffer; out-of-range or non-positive requests are no-ops.
func (b *Buffer) Delete(idx, count int) {
	if count <= 0 || idx >= b.totalRunes {
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx+count > b.totalRunes {
		count = b.totalRunes - idx
	}

	start := b.splitAt(idx)
	end := b.splitAt(idx + count)
	for _, p := range b.pieces[start:end] {
		b.totalRunes -= p.runes
		b.totalNewlines -= p.newlines
	}
	b.pieces = append(b.pieces[:start], b.pieces[end:]...)
}

// RuneAt returns the rune at index idx and true, or (0, false) if idx is out of
// range.
func (b *Buffer) RuneAt(idx int) (rune, bool) {
	if idx < 0 || idx >= b.totalRunes {
		return 0, false
	}
	acc := 0
	for _, p := range b.pieces {
		if acc+p.runes > idx {
			data := b.pieceBytes(p)
			bo := runeByteOffset(data, idx-acc)
			r, _ := utf8.DecodeRune(data[bo:])
			return r, true
		}
		acc += p.runes
	}
	return 0, false
}

// numPieces reports the number of pieces; used by tests and benchmarks.
func (b *Buffer) numPieces() int { return len(b.pieces) }
