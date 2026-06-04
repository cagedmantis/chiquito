# chiquito — Roadmap & Progress

Continuation log for the phased build. If a session is interrupted, start here:
read the **Current status**, then pick up the first ☐ item in the active phase.
Each phase must pass `go build ./... && go vet ./... && go test ./...` and be
benchmarked before the next begins. See `docs/ARCHITECTURE.md` for the design.

---

## Current status

- **Active phase:** Phase 5 (not started)
- **Last completed:** Phase 4 — async spell check, TOML parsing + hot-reload,
  keybinding parser, generalized minibuffer + file open/save-as
- **Tree is green:** `go test -race ./...` passes; benchmarks run.
- **Dependencies:** `bubbletea`, `lipgloss`, `go-runewidth`, `BurntSushi/toml`
  (direct). Hot-reload uses polling (no fsnotify dep). `bubbles` still unused.

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

## Phase 2 — Bubble Tea TUI Integration ✅ DONE

Goal: an interactive editor with viewport rendering and Emacs cursor movement.

- ☑ Added deps: `bubbletea`, `lipgloss`, `go-runewidth`.
- ☑ `internal/editor`: framework-agnostic state — cursor (rune index source of
  truth + derived line/col), viewport (top line, w/h), edit commands. Cached
  `LineIndex` (`lineindex.go`) gives O(log n) movement (benchmark: 70 ns/op,
  0 allocs). Files: `editor.go`, `lineindex.go`, `*_test.go`, `*_bench_test.go`.
- ☑ `internal/ui`: Bubble Tea `Model`/`Update`/`View` over `editor`; all
  charmbracelet imports confined here. Files: `model.go`, `keymap.go`, `view.go`.
- ☑ Emacs nav `C-f C-b C-p C-n C-a C-e`, `C-d`, `C-k`, self-insert, enter,
  backspace, tab; arrows/home/end/pgup/pgdn as built-ins. Multi-key sequences
  (`C-x C-s` save, `C-x C-c` quit) via a pending-prefix state machine, driven by
  `config.DefaultKeybindings()`.
- ☑ Rendering with `lipgloss`; vertical + horizontal scrolling; display-width
  (wcwidth) aware clipping/cursor so CJK/emoji align; line-number gutter;
  reverse-video status bar with name/dirty/position and transient messages.
- ☑ Integration tests drive `Model.Update` with `tea.KeyMsg` and assert on
  buffer/cursor/`View()` (no terminal). Save/quit-confirm covered.
- ☑ `cmd/chiquito` runs `tea.NewProgram(model, tea.WithAltScreen())`.

**Notes / deferred to later phases:** `open` (needs a minibuffer prompt) and
`search`/`replace` are stubbed with status messages; tab reindex is O(n) per
edit (Phase 5); no syntax colors yet (Phase 3).

Validation: `go test -race ./...` green; binary builds and exits cleanly when no
TTY is present. (Interactive smoke test must be run by a human in a terminal.)

---

## Phase 3 — Search, Replace & Syntax Highlighting ✅ DONE

- ☑ `internal/search`: pure engine over strings, rune-index matches. `FindAll`,
  `FindNext`/`FindPrev` (wrapping), `ReplaceAll`; case-sensitivity toggle via
  Unicode folding. Tests + benchmarks (~0.4 ms/0.9 ms over 215 KB).
- ☑ Incremental search UI (C-s start / next, C-r prev, C-t case toggle, Enter
  accept, Esc cancel+restore) and a two-step replace-all prompt (alt+%), in
  `internal/ui/search.go`. Live match highlight (current vs others).
- ☑ `internal/syntax`: line-oriented tokenizer engine with carried `State`;
  `Go` (keywords/strings/numbers/comments incl. multi-line block & raw strings)
  and `Markdown` (headings, fences, inline code/emphasis/links); `Plain`
  fallback; `ForFilename`. The UI caches per-line entering states (`enterStates`,
  rebuilt once per edit, not per frame) → viewport-only tokenization per frame.
- ☑ `internal/ui/theme.go`: token type → `lipgloss` style; honors
  `config.Theme.Name` (only "default" defined; more in Phase 4).
- ☑ Benchmarks for search and tokenization; unit tests for matchers and both
  tokenizers (incl. cross-line state threading).

**Deferred to Phase 5:** `search.FindAll` converts text to `[]rune` per call
(~880 KB/alloc); fine now, optimize for big-file incremental search later.

---

## Phase 4 — Spell Checker & Customization ✅ DONE

- ☑ **Generalized minibuffer + interactive file open.** Extracted a reusable
  `lineInput` (`internal/ui/prompt.go`); search/replace now use it too.
  - ☑ `open` (`C-x C-f`): prompts for a path, `fileio.Read`, loads a fresh
    editor, re-selects the syntax `Language`; missing path → empty buffer bound
    to the name. Replaced the old stub.
  - ☑ Save-as: `C-x C-s` on a nameless buffer prompts for a path, binds + writes.
- ☑ `internal/spell`: pure checker (`Check` → misspelling rune spans) with a
  case-insensitive `WordSet`, system word-list loader (`/usr/share/dict/words`)
  + built-in fallback, and code-token heuristics (skip camelCase/snake_case/
  ALLCAPS/digits). UI runs it **off-thread**: `Init` loads the dictionary async;
  edits schedule a debounced (`250ms`) check; a `docVersion` guard discards
  stale results; misspellings render with a red underline.
- ☑ TOML parsing (`BurntSushi/toml`): `config.Load`/`Parse`/`Marshal`/`Save`;
  layers the file over `Default()` (scalars override, keybindings merge),
  validates/repairs, and writes a default file on first run.
- ☑ Hot-reload: `Init` starts a `1.5s` poll (`configTickMsg`); on mtime change
  the config reloads and `applyConfig` rebinds keys, re-themes, updates editor
  settings, and toggles spell checking live. (Polling, not fsnotify — no dep,
  portable.)
- ☑ Keybinding parser: `config.NormalizeChord` accepts Emacs (`C-x C-s`, `M-%`)
  or Bubble Tea (`ctrl+x ctrl+s`) notation; the UI keymap normalizes on build.
- ☑ Tests: config round-trip/merge/validate/normalize; spell offsets/heuristics/
  concurrency (race); hot-reload + key rebinding; open/save-as/cancel flows.

---

## Phase 5 — Hardening, Benchmarking & Polish ☐

- ☐ **CLI: `--help` / `--version` / flag parsing.** `cmd/chiquito` currently does
  raw `os.Args[1:]` with no flag handling — `chiquito --help` tries to open a
  file named `--help`. Add a `flag.FlagSet` with usage text (synopsis, options,
  default keybindings, config path). Dependency-free; could be pulled forward at
  any time if wanted sooner.
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
