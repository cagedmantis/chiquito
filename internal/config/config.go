// Package config defines chiquito's configuration schema, its defaults, and the
// platform-correct on-disk location of the config file.
//
// Phase 1 establishes the schema and directory initialization only. Parsing of
// the TOML file and hot-reloading are introduced in Phase 4; the struct tags
// here describe the file format that parsing will target.
package config

// Config is the complete, user-customizable configuration. Advanced features
// are enabled by default (see Default) and every one of them is toggleable.
type Config struct {
	Editor   EditorConfig  `toml:"editor"`
	Theme    ThemeConfig   `toml:"theme"`
	Features FeatureConfig `toml:"features"`
	Spell    SpellConfig   `toml:"spell"`
	Keys     KeyConfig     `toml:"keybindings"`
}

// EditorConfig holds buffer- and viewport-level preferences.
type EditorConfig struct {
	TabWidth    int  `toml:"tab_width"`
	ExpandTabs  bool `toml:"expand_tabs"`
	LineNumbers bool `toml:"line_numbers"`
	WrapLines   bool `toml:"wrap_lines"`
}

// ThemeConfig selects the active color theme.
type ThemeConfig struct {
	Name string `toml:"name"`
}

// FeatureConfig toggles the optional, on-by-default subsystems.
type FeatureConfig struct {
	SyntaxHighlighting bool `toml:"syntax_highlighting"`
	SpellCheck         bool `toml:"spell_check"`
	AutoSave           bool `toml:"auto_save"`
}

// SpellConfig configures the asynchronous spell checker (Phase 4).
type SpellConfig struct {
	Enabled  bool   `toml:"enabled"`
	Language string `toml:"language"`
}

// KeyConfig maps logical editor actions to key chords. Defaults are Emacs-style
// (see DefaultKeybindings); users may rebind any action.
type KeyConfig struct {
	Bindings map[string]string `toml:"bindings"`
}

// Default returns the built-in configuration: advanced features enabled,
// 4-space soft tabs, line numbers on, and Emacs-style keybindings.
func Default() Config {
	return Config{
		Editor: EditorConfig{
			TabWidth:    4,
			ExpandTabs:  true,
			LineNumbers: true,
			WrapLines:   false,
		},
		Theme: ThemeConfig{Name: "default"},
		Features: FeatureConfig{
			SyntaxHighlighting: true,
			SpellCheck:         true,
			AutoSave:           false,
		},
		Spell: SpellConfig{
			Enabled:  true,
			Language: "en_US",
		},
		Keys: KeyConfig{Bindings: DefaultKeybindings()},
	}
}

// DefaultKeybindings returns the default Emacs-style action→chord map. Chords
// follow Bubble Tea's key naming; multi-key sequences are space-separated.
func DefaultKeybindings() map[string]string {
	return map[string]string{
		"cursor-forward":  "ctrl+f",
		"cursor-backward": "ctrl+b",
		"cursor-up":       "ctrl+p",
		"cursor-down":     "ctrl+n",
		"line-start":      "ctrl+a",
		"line-end":        "ctrl+e",
		"delete-forward":  "ctrl+d",
		"kill-line":       "ctrl+k",
		"search":          "ctrl+s",
		"replace":         "alt+%",
		"save":            "ctrl+x ctrl+s",
		"open":            "ctrl+x ctrl+f",
		"quit":            "ctrl+x ctrl+c",
	}
}
