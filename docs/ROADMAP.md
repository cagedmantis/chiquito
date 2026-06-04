# chiquito — Roadmap & Progress

Continuation log for the phased build. If a session is interrupted, start here:
read the **Current status**, then pick up the first ☐ item in the active phase.
Each phase must pass `go build ./... && go vet ./... && go test ./...` and be
benchmarked before the next begins. See `docs/ARCHITECTURE.md` for the design.

---

## Current status

- **Active phase:** Phase 2 (not started)
- **Last completed:** Phase 1 — editor core (buffer, config, fileio, runnable stub)
- **Tree is green:** `go test -race ./...` passes; benchmarks run.
- **Dependencies:** none yet (stdlib only). Bubble Tea is introduced in Phase 2.

---

## Phase 1 — Foundation & Buffer Core ✅ DONE

- ☑ Project layout (`cmd/chiquito`, `internal/{buffer,config,fileio}`, `docs/`)
- ☑ `internal/buffer`: Unicode piece table — rune-indexed API, byte storage,
  cached rune/newline counts, append coalescing. Files: `buffer.go`, `lines.go`,
  `buffer_test.go`, `buffer_bench_test.go`.
- ☑ `internal/config`: schema + `Default()` (advanced features on, toggleable),
  Emacs `DefaultKeybindings()`, platform paths via `os.UserConfigDir()`.
  (TOML parsing intentionally deferred to Phase 4.)
- ☑ `internal/fileio`: `Read` (non-blocking open, rejects non-regular files,
  size cap, fstat to avoid TOCTOU) and `WriteAtomic` (temp→fsync→rename, perm
  preservation, symlink resolution). Platform split: `open_unix.go`/`open_other.go`.
- ☑ `cmd/chiquito`: runnable stub that loads a file and reports stats.
- ☑ Unit tests + benchmarks; `docs/ARCHITECTURE.md` and `CLAUDE.md` written.

**Notable decision/bug:** `Read` must open with `O_NONBLOCK` — a plain `os.Open`
on a FIFO/device blocks before the regular-file check can reject it.

---

## Phase 2 — Bubble Tea TUI Integration ☐ NEXT

Goal: an interactive editor with viewport rendering and Emacs cursor movement.

- ☐ Add deps: `charmbracelet/bubbletea`, `lipgloss`, `bubbles` (`go get`); commit
  `go.mod`/`go.sum`. (First third-party deps — needs network.)
- ☐ `internal/editor`: framework-agnostic editor state — cursor (rune index +
  line/col), viewport (top line, height/width), and edit commands operating on
  the `buffer`. Add a **cached line index** here for O(log n)/O(1) cursor moves
  (the Phase-5 buffer-tree note can wait; the line index belongs with the cursor).
- ☐ `internal/ui`: Bubble Tea `Model`/`Update`/`View` adapter over `editor`.
  Keep all Bubble Tea imports confined to this package.
- ☐ Emacs navigation: `C-f C-b C-p C-n C-a C-e`, plus self-insert, backspace,
  newline. Wire from `config.DefaultKeybindings()` (parse chords; multi-key
  sequences like `C-x C-s` can be stubbed until Phase 4 config parsing).
- ☐ Viewport rendering with `lipgloss`; horizontal/vertical scrolling; UTF-8
  display width (wcwidth) for CJK/emoji so the cursor aligns.
- ☐ Integration tests driving `Model.Update` with `tea.KeyMsg` sequences and
  asserting on `View()` / cursor state (no real terminal needed).
- ☐ Replace the `cmd/chiquito` stub with `tea.NewProgram(...)`.

Validation: build/vet/test green; manual smoke test over SSH/local terminal.

---

## Phase 3 — Search, Replace & Syntax Highlighting ☐

- ☐ `internal/search`: forward/backward find; case-sensitivity toggle; exact
  match now (regex optional later). Operate over buffer bytes/runes efficiently.
- ☐ Find-and-replace (single + all), incremental search UI in `ui`.
- ☐ `internal/syntax`: tokenizer engine + language definitions (start: Go,
  Markdown). Tokenize only the visible viewport + a margin for latency; design
  for incremental re-tokenization on edit.
- ☐ Themes via `lipgloss` styles keyed by token type; honor `config.Theme`.
- ☐ Benchmarks: search execution, tokenization throughput; unit tests for
  matchers and each language's tokenizer.

---

## Phase 4 — Spell Checker & Customization ☐

- ☐ `internal/spell`: asynchronous checker. Runs in a goroutine over immutable
  text snapshots; returns misspelling spans as `tea.Msg`. Must never block
  `Update` during typing or load. Dictionary load is lazy/async.
- ☐ TOML config **parsing** (`BurntSushi/toml` or `pelletier/go-toml`): load
  `config.toml`, merge over `Default()`, validate, write a default file if none.
- ☐ Hot-reload: watch the config file (fsnotify or poll) and apply changes live,
  including keybindings and feature toggles.
- ☐ Keybinding parser for full chords / multi-key sequences (`C-x C-s` etc.).
- ☐ Tests: config round-trip/merge/validation; spell checker concurrency
  (race detector); hot-reload behavior.

---

## Phase 5 — Hardening, Benchmarking & Polish ☐

- ☐ Security audit: revisit symlink/TOCTOU handling, temp-file lifecycle, large/
  malformed/binary file behavior, path handling.
- ☐ Fuzz tests: `buffer` insert/delete invariants, UTF-8 round-trip, config
  parser, search.
- ☐ Performance: piece table → balanced tree of pieces + cached line index for
  O(log n) random access (internal only, no API change); profile render/input
  hot paths (`pprof`); reduce allocations.
- ☐ Edge cases: huge files, very long lines, no-trailing-newline, CRLF, mixed
  encodings, terminal resize.
- ☐ Docs polish, example config, README usage.

---

## How to resume

1. `go build ./... && go vet ./... && go test -race ./...` to confirm green.
2. Open this file; find the active phase and its first ☐ item.
3. Update the checkboxes and **Current status** as you complete work.
