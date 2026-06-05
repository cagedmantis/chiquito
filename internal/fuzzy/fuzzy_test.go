package fuzzy

import "testing"

func TestMatchBasics(t *testing.T) {
	cases := []struct {
		query, cand string
		want        bool
	}{
		{"", "anything", true},
		{"abc", "abc", true},
		{"abc", "aXbYc", true}, // subsequence
		{"abc", "ab", false},   // missing 'c'
		{"abc", "cba", false},  // wrong order
		{"ABC", "abc", true},   // case-insensitive
		{"go", "main.go", true},
		{"xyz", "main.go", false},
	}
	for _, c := range cases {
		if _, ok := Match(c.query, c.cand); ok != c.want {
			t.Errorf("Match(%q, %q) matched=%v, want %v", c.query, c.cand, ok, c.want)
		}
	}
}

func TestMatchScoringPrefersBoundaryAndConsecutive(t *testing.T) {
	// "main" as a whole prefix should beat scattered matches.
	whole, _ := Match("main", "main.go")
	scattered, ok := Match("main", "domain_helper.go")
	if !ok {
		t.Fatal("expected scattered match")
	}
	if whole <= scattered {
		t.Errorf("prefix match (%d) should outscore scattered (%d)", whole, scattered)
	}
}

func TestMatchPrefersShorter(t *testing.T) {
	short, _ := Match("ab", "ab")
	long, _ := Match("ab", "axxxxxxxxb")
	if short <= long {
		t.Errorf("shorter candidate (%d) should outscore longer (%d)", short, long)
	}
}

func TestRankOrders(t *testing.T) {
	cands := []string{"domain.go", "main.go", "readme.md", "main_test.go"}
	res := Rank("main", cands)
	if len(res) != 3 {
		t.Fatalf("matched %d, want 3 (%v)", len(res), res)
	}
	// "main.go" should rank first (prefix, shortest).
	if cands[res[0].Index] != "main.go" {
		t.Errorf("top match = %q, want main.go", cands[res[0].Index])
	}
	// readme.md does not contain m-a-i-n in order, so it is excluded.
	for _, r := range res {
		if cands[r.Index] == "readme.md" {
			t.Error("readme.md should not match 'main'")
		}
	}
}

func TestRankEmptyQueryKeepsOrder(t *testing.T) {
	cands := []string{"a", "b", "c"}
	res := Rank("", cands)
	if len(res) != 3 {
		t.Fatalf("got %d results, want 3", len(res))
	}
	for i, r := range res {
		if r.Index != i {
			t.Errorf("empty query should preserve order: pos %d -> index %d", i, r.Index)
		}
	}
}

func TestMatchPositions(t *testing.T) {
	cases := []struct {
		query, cand string
		want        []int
	}{
		{"go", "main.go", []int{5, 6}},   // the "go" after the dot
		{"mn", "main.go", []int{0, 3}},   // m...n
		{"abc", "aXbYc", []int{0, 2, 4}}, // subsequence
		{"", "anything", nil},            // empty query → no positions
	}
	for _, c := range cases {
		_, pos, ok := MatchPositions(c.query, c.cand)
		if !ok && c.query != "" {
			t.Errorf("MatchPositions(%q, %q) did not match", c.query, c.cand)
			continue
		}
		if len(pos) != len(c.want) {
			t.Errorf("MatchPositions(%q, %q) = %v, want %v", c.query, c.cand, pos, c.want)
			continue
		}
		for i := range c.want {
			if pos[i] != c.want[i] {
				t.Errorf("MatchPositions(%q, %q) = %v, want %v", c.query, c.cand, pos, c.want)
				break
			}
		}
	}
}

func TestMatchPositionsPrefersBoundary(t *testing.T) {
	// "ml" should match the boundary 'm' and 'l' (main_lib), not the earlier 'm'.
	_, pos, ok := MatchPositions("ml", "main_lib.go")
	if !ok || len(pos) != 2 {
		t.Fatalf("got pos=%v ok=%v", pos, ok)
	}
	if pos[0] != 0 || pos[1] != 5 { // 'm' at 0, 'l' at 5 (after '_')
		t.Errorf("positions = %v, want [0 5]", pos)
	}
}

func FuzzMatch(f *testing.F) {
	f.Add("main", "main.go")
	f.Add("", "x")
	f.Add("abc", "")
	f.Fuzz(func(t *testing.T, query, cand string) {
		// Must never panic; an empty query always matches.
		score, ok := Match(query, cand)
		if query == "" && !ok {
			t.Error("empty query should always match")
		}
		if !ok && score != 0 {
			t.Errorf("non-match should score 0, got %d", score)
		}
	})
}
