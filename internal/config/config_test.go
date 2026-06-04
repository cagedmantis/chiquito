package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.Editor.TabWidth != 4 {
		t.Errorf("TabWidth = %d, want 4", c.Editor.TabWidth)
	}
	if !c.Editor.ExpandTabs || !c.Editor.LineNumbers {
		t.Error("expected ExpandTabs and LineNumbers enabled by default")
	}
	// Advanced features should be on by default.
	if !c.Features.SyntaxHighlighting || !c.Features.SpellCheck {
		t.Error("expected syntax highlighting and spell check on by default")
	}
	if c.Features.AutoSave {
		t.Error("AutoSave should default off")
	}
	if c.Spell.Language != "en_US" {
		t.Errorf("Spell.Language = %q, want en_US", c.Spell.Language)
	}
}

func TestDefaultKeybindings(t *testing.T) {
	kb := DefaultKeybindings()
	want := map[string]string{
		"cursor-forward":  "ctrl+f",
		"cursor-backward": "ctrl+b",
		"cursor-up":       "ctrl+p",
		"cursor-down":     "ctrl+n",
		"line-start":      "ctrl+a",
		"line-end":        "ctrl+e",
	}
	for action, chord := range want {
		if kb[action] != chord {
			t.Errorf("binding %q = %q, want %q", action, kb[action], chord)
		}
	}
	// Mutating the returned map must not affect a freshly built default.
	kb["save"] = "tampered"
	if DefaultKeybindings()["save"] == "tampered" {
		t.Error("DefaultKeybindings returned shared mutable state")
	}
}

func TestDirAndFilePath(t *testing.T) {
	tmp := t.TempDir()
	// os.UserConfigDir consults XDG_CONFIG_HOME on Unix and HOME on macOS.
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("AppData", tmp)
	}

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if !strings.HasSuffix(dir, appName) {
		t.Errorf("Dir() = %q, want suffix %q", dir, appName)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("config dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", dir)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o700 {
			t.Errorf("config dir perm = %o, want 700", perm)
		}
	}

	fp, err := FilePath()
	if err != nil {
		t.Fatalf("FilePath() error: %v", err)
	}
	if got := filepath.Base(fp); got != FileName {
		t.Errorf("FilePath base = %q, want %q", got, FileName)
	}
}
