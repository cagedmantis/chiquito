package spell

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// WordSet is a case-insensitive in-memory Dictionary.
type WordSet struct {
	words map[string]struct{}
}

// NewWordSet builds a WordSet from the given words.
func NewWordSet(words ...string) *WordSet {
	ws := &WordSet{words: make(map[string]struct{}, len(words))}
	for _, w := range words {
		ws.Add(w)
	}
	return ws
}

// Add inserts a word (folded to lower case).
func (ws *WordSet) Add(word string) {
	ws.words[strings.ToLower(word)] = struct{}{}
}

// Contains reports whether word is present, ignoring case. Trailing possessive
// "'s" is accepted so "Carlos's" matches "carlos".
func (ws *WordSet) Contains(word string) bool {
	w := strings.ToLower(word)
	if _, ok := ws.words[w]; ok {
		return true
	}
	if s, ok := strings.CutSuffix(w, "'s"); ok {
		_, found := ws.words[s]
		return found
	}
	return false
}

// Len returns the number of words.
func (ws *WordSet) Len() int { return len(ws.words) }

// ReadWordList loads a newline-delimited word list (one word per line) into a
// WordSet. Blank lines and comments (#) are ignored.
func ReadWordList(r io.Reader) (*WordSet, error) {
	ws := NewWordSet()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ws.Add(line)
	}
	return ws, sc.Err()
}

// systemWordLists are the conventional locations of a system dictionary.
var systemWordLists = []string{
	"/usr/share/dict/words",
	"/usr/dict/words",
}

// Load builds the default dictionary. It first tries a system word list (for a
// real, comprehensive dictionary) and otherwise falls back to a small built-in
// set of common English words plus chiquito's own vocabulary. This function does
// I/O and is intended to be called off the UI thread.
func Load() *WordSet {
	for _, path := range systemWordLists {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		ws, err := ReadWordList(f)
		f.Close()
		if err == nil && ws.Len() > 0 {
			// Ensure our domain words are present regardless of the system list.
			for _, w := range builtinWords {
				ws.Add(w)
			}
			return ws
		}
	}
	return NewWordSet(builtinWords...)
}
