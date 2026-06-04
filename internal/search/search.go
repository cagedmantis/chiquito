// Package search implements chiquito's find and replace engine. It is pure and
// framework-agnostic: it operates on plain strings and reports matches as rune
// index ranges, so it is trivially unit-testable and benchmarkable.
//
// Matching is exact (literal) with an optional case-insensitivity toggle that
// folds runes with unicode.ToLower, so it is Unicode-correct for the common
// cases. Matches are non-overlapping, found left to right.
package search

import (
	"strings"
	"unicode"
)

// Options controls how matching is performed.
type Options struct {
	CaseSensitive bool
}

// Match is a half-open range [Start, End) measured in rune indices.
type Match struct {
	Start int
	End   int
}

func foldRunes(rs []rune) {
	for i, r := range rs {
		rs[i] = unicode.ToLower(r)
	}
}

// FindAll returns every non-overlapping match of query in text, left to right.
// An empty query yields no matches.
func FindAll(text, query string, opts Options) []Match {
	needle := []rune(query)
	if len(needle) == 0 {
		return nil
	}
	hay := []rune(text)
	if !opts.CaseSensitive {
		foldRunes(hay)
		foldRunes(needle)
	}

	var matches []Match
	n, m := len(hay), len(needle)
	for i := 0; i+m <= n; {
		if equalAt(hay, i, needle) {
			matches = append(matches, Match{Start: i, End: i + m})
			i += m // non-overlapping
		} else {
			i++
		}
	}
	return matches
}

func equalAt(hay []rune, off int, needle []rune) bool {
	for j, r := range needle {
		if hay[off+j] != r {
			return false
		}
	}
	return true
}

// FindNext returns the first match whose start is at or after from, wrapping to
// the first match in the document if none follow. ok is false when there are no
// matches at all.
func FindNext(text, query string, from int, opts Options) (Match, bool) {
	all := FindAll(text, query, opts)
	for _, mt := range all {
		if mt.Start >= from {
			return mt, true
		}
	}
	if len(all) > 0 {
		return all[0], true // wrap around
	}
	return Match{}, false
}

// FindPrev returns the last match whose start is strictly before before,
// wrapping to the final match if none precede it.
func FindPrev(text, query string, before int, opts Options) (Match, bool) {
	all := FindAll(text, query, opts)
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Start < before {
			return all[i], true
		}
	}
	if len(all) > 0 {
		return all[len(all)-1], true // wrap around
	}
	return Match{}, false
}

// ReplaceAll replaces every non-overlapping match of query with repl and returns
// the new text together with the number of replacements made. The replacement
// is literal (no capture groups).
func ReplaceAll(text, query, repl string, opts Options) (string, int) {
	matches := FindAll(text, query, opts)
	if len(matches) == 0 {
		return text, 0
	}
	hay := []rune(text)
	var b strings.Builder
	prev := 0
	for _, mt := range matches {
		b.WriteString(string(hay[prev:mt.Start]))
		b.WriteString(repl)
		prev = mt.End
	}
	b.WriteString(string(hay[prev:]))
	return b.String(), len(matches)
}
