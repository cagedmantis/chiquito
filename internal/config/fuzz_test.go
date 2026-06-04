package config

import "testing"

// FuzzParse checks that parsing arbitrary input never panics, and that whenever
// parsing succeeds the result satisfies the validation invariants (so the rest
// of the program can trust a parsed Config unconditionally).
func FuzzParse(f *testing.F) {
	f.Add("[editor]\ntab_width = 4\n")
	f.Add("garbage = = =")
	f.Add("[features]\nspell_check = false\n")
	f.Add("[editor]\ntab_width = -99\n[theme]\nname = \"\"\n")

	f.Fuzz(func(t *testing.T, data string) {
		cfg, err := Parse([]byte(data))
		if err != nil {
			return // a parse error is an acceptable outcome
		}
		if cfg.Editor.TabWidth < 1 {
			t.Errorf("validated config has TabWidth %d", cfg.Editor.TabWidth)
		}
		if cfg.Theme.Name == "" {
			t.Error("validated config has empty theme name")
		}
		if cfg.Spell.Language == "" {
			t.Error("validated config has empty spell language")
		}
		if cfg.Keys.Bindings == nil {
			t.Error("validated config has nil keybindings")
		}
	})
}
