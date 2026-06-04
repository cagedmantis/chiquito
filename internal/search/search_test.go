package search

import (
	"reflect"
	"testing"
)

func TestFindAllCaseSensitive(t *testing.T) {
	got := FindAll("ababab", "ab", Options{CaseSensitive: true})
	want := []Match{{0, 2}, {2, 4}, {4, 6}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindAll = %v, want %v", got, want)
	}
}

func TestFindAllOverlapNonGreedy(t *testing.T) {
	// "aa" in "aaaa" yields non-overlapping matches at 0 and 2.
	got := FindAll("aaaa", "aa", Options{CaseSensitive: true})
	want := []Match{{0, 2}, {2, 4}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindAll = %v, want %v", got, want)
	}
}

func TestFindAllCaseInsensitive(t *testing.T) {
	got := FindAll("Hello HELLO hello", "hello", Options{CaseSensitive: false})
	want := []Match{{0, 5}, {6, 11}, {12, 17}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindAll = %v, want %v", got, want)
	}
	// Case-sensitive finds only the exact one.
	got = FindAll("Hello HELLO hello", "hello", Options{CaseSensitive: true})
	want = []Match{{12, 17}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindAll cs = %v, want %v", got, want)
	}
}

func TestFindAllUnicode(t *testing.T) {
	// Matches are rune indices, not byte offsets.
	got := FindAll("héllo 世界 héllo", "héllo", Options{})
	want := []Match{{0, 5}, {9, 14}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindAll = %v, want %v", got, want)
	}
}

func TestFindAllEmptyQuery(t *testing.T) {
	if got := FindAll("anything", "", Options{}); got != nil {
		t.Errorf("empty query should yield nil, got %v", got)
	}
}

func TestFindNextWraps(t *testing.T) {
	text := "x foo y foo z"
	// from after the first match -> second match.
	mt, ok := FindNext(text, "foo", 3, Options{})
	if !ok || mt.Start != 8 {
		t.Fatalf("FindNext from 3 = %v,%v want start 8", mt, ok)
	}
	// from past the last match -> wrap to first.
	mt, ok = FindNext(text, "foo", 9, Options{})
	if !ok || mt.Start != 2 {
		t.Fatalf("FindNext wrap = %v,%v want start 2", mt, ok)
	}
	if _, ok := FindNext(text, "zzz", 0, Options{}); ok {
		t.Error("expected no match for absent query")
	}
}

func TestFindPrevWraps(t *testing.T) {
	text := "foo a foo b foo"
	mt, ok := FindPrev(text, "foo", 12, Options{})
	if !ok || mt.Start != 6 {
		t.Fatalf("FindPrev before 12 = %v,%v want start 6", mt, ok)
	}
	// before the first match -> wrap to last.
	mt, ok = FindPrev(text, "foo", 0, Options{})
	if !ok || mt.Start != 12 {
		t.Fatalf("FindPrev wrap = %v,%v want start 12", mt, ok)
	}
}

func TestReplaceAll(t *testing.T) {
	got, n := ReplaceAll("the cat sat on the mat", "at", "AT", Options{CaseSensitive: true})
	if n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
	if got != "the cAT sAT on the mAT" {
		t.Errorf("ReplaceAll = %q", got)
	}
}

func TestReplaceAllDifferentLength(t *testing.T) {
	got, n := ReplaceAll("a.b.c", ".", "::", Options{})
	if n != 2 || got != "a::b::c" {
		t.Errorf("ReplaceAll = %q, %d", got, n)
	}
	// Unicode replacement.
	got, n = ReplaceAll("hi hi", "hi", "👋", Options{})
	if n != 2 || got != "👋 👋" {
		t.Errorf("ReplaceAll unicode = %q, %d", got, n)
	}
}

func TestReplaceAllNoMatch(t *testing.T) {
	got, n := ReplaceAll("hello", "xyz", "q", Options{})
	if n != 0 || got != "hello" {
		t.Errorf("ReplaceAll = %q, %d; want unchanged", got, n)
	}
}
