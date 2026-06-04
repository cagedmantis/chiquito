package spell

import (
	"reflect"
	"strings"
	"testing"
)

func dict(words ...string) *WordSet { return NewWordSet(words...) }

func TestCheckBasic(t *testing.T) {
	d := dict("the", "cat", "sat")
	got := Check("the kat sat", d)
	want := []Misspelling{{4, 7}} // "kat"
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Check = %v, want %v", got, want)
	}
}

func TestCheckCaseInsensitive(t *testing.T) {
	d := dict("hello", "world")
	if got := Check("Hello WORLD", d); len(got) != 0 {
		t.Errorf("expected no misspellings, got %v", got)
	}
}

func TestCheckUnicodeOffsets(t *testing.T) {
	d := dict("café", "ok")
	// "wörd" is unknown; offsets must be rune indices.
	got := Check("café wörd ok", d)
	want := []Misspelling{{5, 9}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Check = %v, want %v", got, want)
	}
}

func TestCheckSkipsCodeLikeTokens(t *testing.T) {
	d := dict("the", "value") // deliberately small
	// camelCase, snake_case, ALLCAPS, digit-bearing, and single letters skipped.
	text := "camelCase snake_case HTTP x value2 the"
	got := Check(text, d)
	// Only "the" and "value" are checkable here; both are in the dict, so none flagged.
	if len(got) != 0 {
		for _, m := range got {
			t.Logf("flagged: %q", []rune(text)[m.Start:m.End])
		}
		t.Errorf("expected code-like tokens skipped, got %d flags", len(got))
	}
}

func TestCheckPossessive(t *testing.T) {
	d := dict("carlos")
	if got := Check("Carlos's editor", dictWith(d, "editor")); len(got) != 0 {
		t.Errorf("possessive should match base word, got %v", got)
	}
}

func dictWith(ws *WordSet, extra ...string) *WordSet {
	for _, w := range extra {
		ws.Add(w)
	}
	return ws
}

func TestCheckNilDictionary(t *testing.T) {
	if got := Check("anything here", nil); got != nil {
		t.Errorf("nil dict should yield nil, got %v", got)
	}
}

func TestReadWordList(t *testing.T) {
	ws, err := ReadWordList(strings.NewReader("apple\n# a comment\n\nBanana\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !ws.Contains("apple") || !ws.Contains("banana") {
		t.Error("expected apple and banana present")
	}
	if ws.Len() != 2 {
		t.Errorf("Len = %d, want 2 (comment/blank skipped)", ws.Len())
	}
}

func TestLoadAlwaysHasBuiltins(t *testing.T) {
	// Whether or not a system list exists, chiquito's own words must be present.
	ws := Load()
	for _, w := range []string{"chiquito", "markdown", "editor"} {
		if !ws.Contains(w) {
			t.Errorf("Load() missing builtin word %q", w)
		}
	}
}
