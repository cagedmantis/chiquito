# chiquito

A fast, terminal-based text editor for source code and Markdown, built in Go on
the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework.
Optimized for SSH and local terminals; Unicode/UTF-8 throughout.

## Features

- **Piece-table buffer** — original file bytes are never copied; low memory,
  fast edits, full Unicode.
- **Emacs-style keybindings**, fully customizable (Emacs or Bubble Tea notation).
- **File-browser pane** — `C-x C-f` opens a selectable list of the current
  directory; **type to fuzzy-filter** (fzf-style ranking with matched-character
  highlighting), navigate, enter directories, open files.
- **Incremental search** and **query-replace** with a case-sensitivity toggle.
- **Syntax highlighting** via [Chroma](https://github.com/alecthomas/chroma):
  ~250 languages and themeable colors (with a small builtin fallback).
- **Asynchronous spell checking** that never blocks typing.
- **TOML configuration** with **live hot-reload**.
- **Secure file I/O**: non-blocking open, rejects non-regular files, atomic
  saves (temp + fsync + rename) that preserve permissions and symlinks.

## Install / build

```sh
go build -o chiquito ./cmd/chiquito
# or
go install argc.dev/chiquito/cmd/chiquito@latest
```

Requires Go 1.26+. Supported: Linux and macOS (tier 1); BSDs best-effort.

## Usage

```sh
chiquito [flags] [file]      # open a file, or a scratch buffer with no argument

  -version            print version and exit
  -cpuprofile file    write a CPU profile
  -memprofile file    write a memory profile on exit
```

### Default keybindings

| Keys | Action | | Keys | Action |
|------|--------|-|------|--------|
| `C-f` / `C-b` | forward / back char | | `C-s` | incremental search |
| `C-p` / `C-n` | previous / next line | | `C-r` | (in search) previous match |
| `C-a` / `C-e` | start / end of line | | `C-t` | (in search) toggle case |
| `C-d` | delete char | | `M-%` | query replace all |
| `C-k` | kill to end of line | | `C-x C-f` | open file |
| arrows / Home / End / PgUp / PgDn | navigate | | `C-x C-s` | save (prompts if unnamed) |
| `Esc` / `C-g` | cancel a prompt | | `C-x C-c` | quit |

With `file_pane` enabled (the default), `C-x C-f` opens a **file-browser pane**
in the bottom rows: **just start typing to fuzzy-filter** the listing (best
matches rank first); `↑`/`↓` (or `C-p`/`C-n`) to move, `Enter` to open a file or
enter a directory, `⌫` to delete a filter character (or go up when the filter is
empty), `←` to go up, `Esc` to cancel. With `file_pane = false`
it falls back to a text path prompt prefilled with a starting directory (the
current file's directory, else the working directory, else home); there, as in
Emacs, typing `//` resets to the filesystem root and `/~` resets to home.

## Configuration

A TOML config is created on first run (and hot-reloaded on change) at:

- `~/.config/chiquito/config.toml` (Linux/BSD)
- `~/Library/Application Support/chiquito/config.toml` (macOS)

Every field is optional; keybindings are merged over the defaults. See
[`docs/config.example.toml`](docs/config.example.toml) for the full annotated
schema.

## Project layout & development

The editor core (`internal/{buffer,editor,search,syntax,spell,config,fileio,fuzzy}`)
is framework-agnostic and does not import Bubble Tea; only `internal/ui` does.
See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the design and
[`docs/ROADMAP.md`](docs/ROADMAP.md) for the phased build history.

```sh
go test ./...                                   # unit + integration tests
go test -race ./...                             # race detector
go test -run=NONE -bench=. ./...                # benchmarks
go test -run=NONE -fuzz=FuzzInsertDelete ./internal/buffer/   # fuzzing
go vet ./...
```
