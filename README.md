# chiquito

A fast, terminal-based text editor for source code and Markdown, built in Go on
the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework.
Optimized for SSH and local terminals; Unicode/UTF-8 throughout.

## Features

- **Piece-table buffer** — original file bytes are never copied; low memory,
  fast edits, full Unicode.
- **Emacs-style keybindings**, fully customizable (Emacs or Bubble Tea notation).
- **Incremental search** and **query-replace** with a case-sensitivity toggle.
- **Syntax highlighting** for Go and Markdown (viewport-only, incremental).
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

## Configuration

A TOML config is created on first run (and hot-reloaded on change) at:

- `~/.config/chiquito/config.toml` (Linux/BSD)
- `~/Library/Application Support/chiquito/config.toml` (macOS)

Every field is optional; keybindings are merged over the defaults. See
[`docs/config.example.toml`](docs/config.example.toml) for the full annotated
schema.

## Project layout & development

The editor core (`internal/{buffer,editor,search,syntax,spell,config,fileio}`)
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
