// Package fuzzy implements fzf-style fuzzy matching with scoring and match
// positions, used to filter, rank, and highlight picker lists (e.g. the file
// pane). It is pure and framework-agnostic.
//
// A query matches a candidate when every query rune appears in the candidate in
// order, case-insensitively. Unlike a greedy scan, MatchPositions finds the
// optimal alignment with a dynamic program (a Smith-Waterman variant in the
// style of fzf): it rewards matches at word boundaries, camelCase humps, and
// consecutive runs, and penalizes gaps, so the best alignment — and the exact
// matched character positions for highlighting — are returned.
package fuzzy

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

const (
	scoreMatch       = 16
	scoreGap         = -1 // penalty per skipped candidate rune between matches
	bonusBoundary    = 8  // match right after a separator/space, or at the start
	bonusCamel       = 7  // match at a lower→upper or x→digit transition
	bonusConsecutive = 4  // match immediately after another match
)

// character classes for boundary detection.
const (
	clsWhite = iota
	clsNonWord
	clsLower
	clsUpper
	clsDigit
)

func class(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return clsWhite
	case unicode.IsUpper(r):
		return clsUpper
	case unicode.IsLower(r):
		return clsLower
	case unicode.IsDigit(r):
		return clsDigit
	default:
		return clsNonWord
	}
}

// bonusAt returns the bonus for a match whose candidate rune has class cur and
// whose preceding rune has class prev (prev is clsWhite at the start).
func bonusAt(prev, cur int) int {
	if cur == clsLower || cur == clsUpper || cur == clsDigit {
		switch prev {
		case clsWhite, clsNonWord:
			return bonusBoundary
		}
		if prev == clsLower && cur == clsUpper {
			return bonusCamel
		}
		if prev != clsDigit && cur == clsDigit {
			return bonusCamel
		}
	}
	return 0
}

const negInf = math.MinInt32

// MatchPositions reports whether query fuzzily matches candidate and, if so, the
// score (higher is better) and the candidate rune indices that were matched, in
// ascending order. An empty query matches everything with score 0 and no
// positions.
func MatchPositions(query, candidate string) (score int, positions []int, matched bool) {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return 0, nil, true
	}
	t := []rune(candidate)
	M, N := len(q), len(t)
	if N < M {
		return 0, nil, false
	}

	tl := make([]rune, N)
	for i, r := range t {
		tl[i] = unicode.ToLower(r)
	}

	bonus := make([]int, N)
	prev := clsWhite
	for j := 0; j < N; j++ {
		c := class(t[j])
		bonus[j] = bonusAt(prev, c)
		prev = c
	}

	// H: best score; C: length of the consecutive run ending here; move: 1=match,
	// 2=skip, for backtracking.
	H := make([][]int, M+1)
	C := make([][]int, M+1)
	move := make([][]uint8, M+1)
	for i := range H {
		H[i] = make([]int, N+1)
		C[i] = make([]int, N+1)
		move[i] = make([]uint8, N+1)
	}
	for i := 1; i <= M; i++ {
		H[i][0] = negInf
	}

	for i := 1; i <= M; i++ {
		for j := 1; j <= N; j++ {
			matchVal := negInf
			cons := 0
			if q[i-1] == tl[j-1] && H[i-1][j-1] > negInf {
				b := bonus[j-1]
				cons = 1
				if C[i-1][j-1] > 0 {
					cons = C[i-1][j-1] + 1
					b += bonusConsecutive
				}
				add := scoreMatch + b
				if i == 1 {
					add += b // emphasize the first matched character
				}
				matchVal = H[i-1][j-1] + add
			}
			skipVal := H[i][j-1] + scoreGap
			if matchVal >= skipVal {
				H[i][j], C[i][j], move[i][j] = matchVal, cons, 1
			} else {
				H[i][j], C[i][j], move[i][j] = skipVal, 0, 2
			}
		}
	}

	best, bestJ := negInf, -1
	for j := M; j <= N; j++ {
		if H[M][j] > best {
			best, bestJ = H[M][j], j
		}
	}
	if bestJ < 0 || best <= negInf/2 {
		return 0, nil, false
	}

	i, j := M, bestJ
	for i > 0 && j > 0 {
		if move[i][j] == 1 {
			positions = append(positions, j-1)
			i--
			j--
		} else {
			j--
		}
	}
	for a, b := 0, len(positions)-1; a < b; a, b = a+1, b-1 {
		positions[a], positions[b] = positions[b], positions[a]
	}
	return best, positions, true
}

// Match reports whether query matches candidate and the score, discarding
// positions.
func Match(query, candidate string) (int, bool) {
	score, _, ok := MatchPositions(query, candidate)
	return score, ok
}

// Result is a matched candidate's original index and score.
type Result struct {
	Index int
	Score int
}

// Rank returns the candidates that match query, ordered by descending score
// (ties broken by original order). An empty query returns all candidates in
// their original order.
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
