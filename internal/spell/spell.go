// Package spell provides chiquito's spell-checking primitives. It is pure and
// framework-agnostic: Check scans text and returns the rune ranges of words not
// found in a Dictionary. The asynchronous orchestration (running Check off the
// UI thread and delivering results as messages) lives in the UI layer; keeping
// this package pure makes it deterministic and easy to test and benchmark.
package spell

import "unicode"

// Misspelling is a half-open rune range [Start, End) flagged as misspelled.
type Misspelling struct {
	Start int
	End   int
}

// Dictionary reports whether a word is spelled correctly. Implementations should
// treat lookups case-insensitively.
type Dictionary interface {
	Contains(word string) bool
}

// Check returns the misspelled word ranges in text. Words are maximal runs of
// letters and apostrophes. Tokens that look like code or identifiers — those
// containing digits or underscores, written in camelCase, or in ALLCAPS — are
// skipped to avoid drowning source code in false positives. Single-letter words
// are ignored.
func Check(text string, dict Dictionary) []Misspelling {
	if dict == nil {
		return nil
	}
	var out []Misspelling
	runes := []rune(text)
	n := len(runes)
	i := 0
	for i < n {
		if !isWordRune(runes[i]) {
			i++
			continue
		}
		start := i
		for i < n && isWordRune(runes[i]) {
			i++
		}
		word := runes[start:i]
		if shouldCheck(word) && !dict.Contains(string(word)) {
			out = append(out, Misspelling{Start: start, End: i})
		}
	}
	return out
}

// isWordRune defines what stays joined in a single token. Digits and
// underscores are included so identifier-like tokens (snake_case, value2) remain
// whole and are then filtered by shouldCheck, rather than being split into prose
// fragments.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '_'
}

// shouldCheck filters out tokens that are almost certainly not prose words.
func shouldCheck(word []rune) bool {
	if len(word) < 2 {
		return false
	}
	upper, lower := 0, 0
	for i, r := range word {
		switch {
		case unicode.IsDigit(r) || r == '_':
			return false // identifier-like
		case unicode.IsUpper(r):
			upper++
			if i > 0 && unicode.IsLower(word[i-1]) {
				return false // camelCase
			}
		case unicode.IsLower(r):
			lower++
		}
	}
	// ALLCAPS acronyms (no lowercase letters) are skipped.
	if lower == 0 && upper > 1 {
		return false
	}
	return true
}
