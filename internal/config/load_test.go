package config

import (
	"testing"
)

func TestParseOverridesDefaults(t *testing.T) {
	data := `
[editor]
tab_width = 8
line_numbers = false

[features]
spell_check = false
`
	cfg, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// Overridden fields.
	if cfg.Editor.TabWidth != 8 {
		t.Errorf("TabWidth = %d, want 8", cfg.Editor.TabWidth)
	}
	if cfg.Editor.LineNumbers {
		t.Error("LineNumbers should be false")
	}
	if cfg.Features.SpellCheck {
		t.Error("SpellCheck should be false")
	}
	// Untouched fields keep their defaults.
	if !cfg.Editor.ExpandTabs {
		t.Error("ExpandTabs should keep default true")
	}
	if !cfg.Features.SyntaxHighlighting {
		t.Error("SyntaxHighlighting should keep default true")
	}
}

func TestParseMergesKeybindings(t *testing.T) {
	data := `
[keybindings.bindings]
save = "C-s"
`
	cfg, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// The override is applied...
	if cfg.Keys.Bindings["save"] != "C-s" {
		t.Errorf("save = %q, want C-s", cfg.Keys.Bindings["save"])
	}
	// ...and unrelated default bindings are preserved (merge, not replace).
	if cfg.Keys.Bindings["cursor-forward"] != "ctrl+f" {
		t.Errorf("cursor-forward = %q, want default ctrl+f", cfg.Keys.Bindings["cursor-forward"])
	}
}

func TestParseValidatesBadValues(t *testing.T) {
	cfg, err := Parse([]byte("[editor]\ntab_width = -3\n[theme]\nname = \"\"\n"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if cfg.Editor.TabWidth != 4 {
		t.Errorf("invalid TabWidth not repaired: %d", cfg.Editor.TabWidth)
	}
	if cfg.Theme.Name != "default" {
		t.Errorf("empty theme not repaired: %q", cfg.Theme.Name)
	}
}

func TestParseInvalidTOML(t *testing.T) {
	if _, err := Parse([]byte("this is = = not toml")); err == nil {
		t.Error("expected parse error for invalid TOML")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	original := Default()
	original.Editor.TabWidth = 2
	original.Theme.Name = "solarized"

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got.Editor.TabWidth != 2 || got.Theme.Name != "solarized" {
		t.Errorf("round trip lost data: %+v", got.Editor)
	}
	if got.Keys.Bindings["quit"] != original.Keys.Bindings["quit"] {
		t.Error("round trip lost keybindings")
	}
}

func TestLoadCreatesDefaultFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)

	cfg, path, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Editor.TabWidth != 4 {
		t.Errorf("default TabWidth = %d, want 4", cfg.Editor.TabWidth)
	}
	// The file should now exist and re-parse to the same defaults.
	cfg2, _, err := Load()
	if err != nil {
		t.Fatalf("second Load error: %v", err)
	}
	if cfg2.Theme.Name != cfg.Theme.Name {
		t.Error("re-load mismatch")
	}
	if path == "" {
		t.Error("Load returned empty path")
	}
}

func TestNormalizeChord(t *testing.T) {
	cases := map[string]string{
		"C-s":           "ctrl+s",
		"C-x C-s":       "ctrl+x ctrl+s",
		"ctrl+f":        "ctrl+f",
		"M-%":           "alt+%",
		"Alt-x":         "alt+x",
		"S-tab":         "shift+tab",
		"ctrl+x ctrl+c": "ctrl+x ctrl+c",
		"C-x C-c":       "ctrl+x ctrl+c",
		"enter":         "enter",
		"a":             "a",
	}
	for in, want := range cases {
		if got := NormalizeChord(in); got != want {
			t.Errorf("NormalizeChord(%q) = %q, want %q", in, got, want)
		}
	}
}
