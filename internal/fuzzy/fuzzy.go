// Package fuzzy implements lightweight fuzzy subsequence matching with scoring,
// used to filter and rank picker lists (e.g. the file pane). It is pure and
// framework-agnostic.
//
// A query matches a candidate when every query rune appears in the candidate in
// order, case-insensitively. The score rewards consecutive matches and matches
// at word boundaries (after a separator or a camelCase hump), and lightly
// prefers shorter candidates, so the most relevant items rank first.
package fuzzy

import (
	"sort"
	"strings"
	"unicode"
)

const (
	scoreMatch       = 16
	bonusConsecutive = 8
	bonusBoundary    = 10
	bonusFirstChar   = 4
)

// Match reports whether query fuzzily matches candidate and, if so, a score
// (higher is better). An empty query matches everything with score 0. Matching
// is greedy, which is fast and good enough for file names.
func Match(query, candidate string) (score int, matched bool) {
	if query == "" {
		return 0, true
	}
	q := []rune(strings.ToLower(query))
	c := []rune(candidate)

	qi := 0
	prevMatch := -2
	for ci := 0; ci < len(c) && qi < len(q); ci++ {
		if unicode.ToLower(c[ci]) != q[qi] {
			continue
		}
		s := scoreMatch
		if ci == prevMatch+1 {
			s += bonusConsecutive
		}
		if ci == 0 {
			s += bonusFirstChar + bonusBoundary
		} else if isBoundary(c, ci) {
			s += bonusBoundary
		}
		score += s
		prevMatch = ci
		qi++
	}
	if qi != len(q) {
		return 0, false
	}
	// Prefer shorter candidates among otherwise equal matches.
	score -= len(c)
	return score, true
}

func isBoundary(c []rune, i int) bool {
	prev := c[i-1]
	switch prev {
	case '/', '\\', '_', '-', '.', ' ':
		return true
	}
	return unicode.IsLower(prev) && unicode.IsUpper(c[i])
}

// Result is a matched candidate's original index and score.
type Result struct {
	Index int
	Score int
}

// Rank returns the candidates that match query, ordered by descending score
// (ties broken by original order for stability). An empty query returns all
// candidates in their original order.
func Rank(query string, candidates []string) []Result {
	out := make([]Result, 0, len(candidates))
	for i, cand := range candidates {
		if score, ok := Match(query, cand); ok {
			out = append(out, Result{Index: i, Score: score})
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].Score > out[b].Score
	})
	return out
}
