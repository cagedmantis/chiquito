// Command chiquito is a terminal-based text editor.
//
// Usage:
//
//	chiquito [flags] [file]
//
// With a file argument the file is loaded into the buffer; without one a
// scratch buffer is opened. Default keybindings are Emacs-style (see
// internal/config.DefaultKeybindings); C-x C-s saves and C-x C-c quits.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/editor"
	"argc.dev/chiquito/internal/fileio"
	"argc.dev/chiquito/internal/ui"
)

// version is overridable at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses flags and launches the editor, returning a process exit code. It
// takes its arguments and writers explicitly so the flag-handling paths are
// testable without a terminal.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("chiquito", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { usage(stderr) }

	var (
		showVersion bool
		cpuProfile  string
		memProfile  string
	)
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.StringVar(&cpuProfile, "cpuprofile", "", "write a CPU profile to `file`")
	fs.StringVar(&memProfile, "memprofile", "", "write a memory profile to `file` on exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if showVersion {
		fmt.Fprintf(stdout, "chiquito %s\n", version)
		return 0
	}

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			fmt.Fprintln(stderr, "chiquito:", err)
			return 1
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintln(stderr, "chiquito:", err)
			return 1
		}
		defer pprof.StopCPUProfile()
	}

	if err := launch(fs.Arg(0)); err != nil {
		fmt.Fprintln(stderr, "chiquito:", err)
		return 1
	}

	if memProfile != "" {
		if err := writeMemProfile(memProfile); err != nil {
			fmt.Fprintln(stderr, "chiquito:", err)
			return 1
		}
	}
	return 0
}

// launch builds the editor and runs the Bubble Tea program.
func launch(name string) error {
	cfg, _, err := config.Load()
	if err != nil {
		cfg = config.Default() // a broken config shouldn't block editing
	}

	var content []byte
	if name != "" {
		data, rerr := fileio.Read(name)
		if rerr != nil && !os.IsNotExist(rerr) {
			return rerr
		}
		content = data
	}

	ed := editor.New(content, name)
	ed.SetTabStops(cfg.Editor.TabWidth, cfg.Editor.ExpandTabs)

	prog := tea.NewProgram(ui.New(ed, cfg), tea.WithAltScreen())
	_, err = prog.Run()
	return err
}

func writeMemProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	runtime.GC()
	return pprof.WriteHeapProfile(f)
}

func usage(w io.Writer) {
	fmt.Fprint(w, `chiquito — a terminal text editor for code and Markdown

Usage:
  chiquito [flags] [file]

Flags:
  -version            print version and exit
  -cpuprofile file    write a CPU profile to file
  -memprofile file    write a memory profile to file on exit

Default keybindings (Emacs-style; customizable in the config file):
  C-f / C-b           move forward / backward by one character
  C-p / C-n           move to previous / next line
  C-a / C-e           move to start / end of line
  C-d / C-k           delete character / kill to end of line
  C-s                 incremental search   (C-s next, C-r prev, C-t case)
  M-%                 query replace all
  C-x C-f             open a file
  C-x C-s             save (prompts for a name if the buffer is unnamed)
  C-x C-c             quit

Config file (TOML, created on first run):
  $XDG_CONFIG_HOME/chiquito/config.toml  (Linux/BSD)
  ~/Library/Application Support/chiquito/config.toml  (macOS)
  Changes are hot-reloaded while running.
`)
}
