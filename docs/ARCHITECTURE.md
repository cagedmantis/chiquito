# chiquito — Architecture Design

`chiquito` is a terminal-based text editor for source code and Markdown,
optimized for SSH and local terminal use. It is built in Go on top of the
[Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework
(`lipgloss` for styling, `bubbles` for reusable widgets).

Design priorities, in order: **correctness**, **security**, **low latency on
large files**, and **full customizability**.

---

## Guiding principles

- **The editor core is framework-agnostic.** The text buffer, search, syntax
  tokenizer, spell checker, and file I/O live in `internal/` packages that do
  not import Bubble Tea. The TUI is a thin adapter (a Bubble Tea `Model`) over
  that core. This keeps the hot paths unit-testable and benchmarkable without a
  terminal, and lets the core be reused (e.g. headless tooling, fuzzing).
- **Rune-addressed API, byte-stored data.** All public positions are rune
  indices (or line/column). Data is stored as UTF-8 bytes and never widened to
  `[]rune`, keeping memory close to the file's on-disk size.
- **Never block the UI thread.** Expensive or unbounded work (spell checking,
  large-file syntax passes) runs in goroutines and reports back via Bubble Tea
  messages (`tea.Cmd` / `tea.Msg`).
- **Secure by construction.** File reads reject non-regular files and are size
  capped; saves are atomic (temp file + fsync + rename) with preserved
  permissions and deliberate symlink handling.

---

## Module & package layout

```
argc.dev/chiquito
├── cmd/chiquito/            # main entry point (arg parsing, wiring)
├── internal/
│   ├── buffer/              # piece-table text store (Phase 1)
│   ├── config/              # config schema, defaults, platform paths (Phase 1; parsing Phase 4)
│   ├── fileio/              # secure read + atomic write (Phase 1)
│   ├── editor/              # editor state machine: cursor, viewport, commands (Phase 2)
│   ├── ui/                  # Bubble Tea Model/Update/View adapter (Phase 2)
│   ├── search/              # find & replace engine (Phase 3)
│   ├── syntax/              # tokenizer + highlight themes (Phase 3)
│   └── spell/               # asynchronous spell checker (Phase 4)
└── docs/                    # this document and design notes
```

`internal/` is used deliberately: nothing here is a public API surface yet, so
we keep freedom to refactor.

---

## Core data structure: the piece table

The buffer is a **piece table** rather than a gap buffer. Rationale for the
"large files, low memory" requirement:

- The original file bytes are loaded **once** into a read-only `original` slice
  and **never copied**. A gap buffer of `[]rune` would cost ~4× the file size;
  a gap buffer of bytes still copies on every gap move.
- Inserts append to a grow-only `added` slice; edits only mutate a small,
  ordered list of `piece` descriptors `{source, byteOffset, byteLen}`.
- Each piece caches its **rune count** and **newline count**, so rune- and
  line-oriented navigation never rescans the whole document.

```
original: "package main\n…"     added: "func "        (grow-only)
pieces:   [ {orig, 0, 8} ] [ {add, 0, 5} ] [ {orig, 8, …} ]
```

Trade-off: random access is O(#pieces) today. Editing is localized, so the
piece count stays small in practice; coalescing of adjacent same-source pieces
keeps it bounded. **Future optimization** (Phase 5): index pieces in a balanced
tree and maintain a cached line index for O(log n) position lookup. Both are
internal and do not change the public API.

Invalid UTF-8 is counted byte-per-rune so that load→save is byte-exact.

---

## Concurrency model

Bubble Tea is single-threaded in `Update`; all shared state lives in the
`Model` and is mutated only there. Background work follows the
command/message pattern:

```
Update --(spawns)--> tea.Cmd --(goroutine)--> tea.Msg --(delivered to)--> Update
```

The spell checker and large-file syntax passes own no editor state; they
receive immutable snapshots (a `[]byte` slice of the relevant text) and return
results (spans/diagnostics) as messages. This avoids locks on the hot path.

---

## Security model

- **Reads** (`fileio.Read`): `fstat` the open descriptor (not a pre-open
  `stat`, avoiding TOCTOU), reject anything that is not a regular file
  (devices/FIFOs/sockets can block or mislead), and cap bytes read.
- **Writes** (`fileio.WriteAtomic`): write to a `0600` temp file in the
  destination directory, `fsync`, then `rename` over the target so a crash
  never yields a truncated file; `fsync` the directory for durability. Existing
  permissions are preserved; symlinks are resolved so the *target* is replaced
  and the link is kept intact.
- **Config**: directory created `0700` under the OS-standard location via
  `os.UserConfigDir()`.

---

## Phased delivery

| Phase | Scope | State |
|-------|-------|-------|
| 1 | Project structure, config dir init, piece-table buffer, secure file I/O | **this change** |
| 2 | Bubble Tea loop, viewport rendering, Emacs cursor navigation | planned |
| 3 | Search/replace, syntax tokenizer, benchmarks | planned |
| 4 | Async spell check, TOML config parsing + hot reload | planned |
| 5 | Security audit, edge-case/fuzz tests, profiling & polish | planned |

Each phase must be green (`go test ./... && go vet ./...`) and benchmarked
before the next begins.

---

## Phase 1 deliverables

- `internal/buffer`: piece table with `New`, `Insert`, `Delete`, `Len`,
  `LineCount`, `Line`, `RuneAt`, `Bytes`/`String`, and line/column conversion;
  unit tests + benchmarks.
- `internal/config`: `Config` schema with `toml` tags, `Default()` (advanced
  features on, all toggleable), and platform-correct `Dir()`/`FilePath()`.
  (Parsing/hot-reload deferred to Phase 4.)
- `internal/fileio`: `Read` and `WriteAtomic` per the security model.
- `cmd/chiquito`: runnable stub that loads a file into the buffer and reports
  stats (the TUI arrives in Phase 2).

No third-party dependencies are introduced in Phase 1; Bubble Tea et al. are
added in Phase 2.
