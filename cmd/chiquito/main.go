// Command chiquito is a terminal-based text editor.
//
// Phase 1 ships the editor core (piece-table buffer, secure file I/O, config
// directory) without the interactive TUI, which arrives in Phase 2. Running the
// binary loads an optional file into the buffer and reports what it found, so
// the foundation is exercisable end-to-end.
package main

import (
	"fmt"
	"os"

	"argc.dev/chiquito/internal/buffer"
	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/fileio"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "chiquito:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	cfg := config.Default()

	dir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}

	var (
		buf  *buffer.Buffer
		name string
	)
	if len(args) > 0 {
		name = args[0]
		data, err := fileio.Read(name)
		if err != nil {
			return err
		}
		buf = buffer.New(data)
	} else {
		buf = buffer.New(nil)
	}

	fmt.Fprintln(out, "chiquito — phase 1 (editor core)")
	fmt.Fprintf(out, "config dir:           %s\n", dir)
	fmt.Fprintf(out, "syntax highlighting:  %v (default)\n", cfg.Features.SyntaxHighlighting)
	fmt.Fprintf(out, "spell check:          %v (default)\n", cfg.Features.SpellCheck)
	if name != "" {
		fmt.Fprintf(out, "opened %q: %d runes, %d lines\n", name, buf.Len(), buf.LineCount())
	} else {
		fmt.Fprintln(out, "no file given; started with an empty buffer")
	}
	fmt.Fprintln(out, "(interactive TUI arrives in phase 2)")
	return nil
}
