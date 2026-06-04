// Command chiquito is a terminal-based text editor.
//
// Usage:
//
//	chiquito [file]
//
// With a file argument the file is loaded into the buffer; without one a
// scratch buffer is opened. Default keybindings are Emacs-style (see
// internal/config.DefaultKeybindings); C-x C-s saves and C-x C-c quits.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/fileio"
	"argc.dev/chiquito/internal/ui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "chiquito:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, _, err := config.Load()
	if err != nil {
		// A broken config file shouldn't prevent editing; fall back to defaults.
		cfg = config.Default()
		fmt.Fprintln(os.Stderr, "chiquito: using defaults:", err)
	}

	var (
		content []byte
		name    string
	)
	if len(args) > 0 {
		name = args[0]
		// A missing file is fine — open an empty buffer that saves to that path.
		// Any other error (permissions, non-regular file) is fatal.
		data, err := fileio.Read(name)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		content = data
	}

	ed := editor.New(content, name)
	ed.SetTabStops(cfg.Editor.TabWidth, cfg.Editor.ExpandTabs)

	model := ui.New(ed, cfg)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	_, err = prog.Run()
	return err
}
