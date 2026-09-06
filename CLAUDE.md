# CLAUDE.md

Guidance for Claude Code (and other agents) working in this repository.

## Project overview

`pieces-store` is an early-stage Go text editor built around a piece-table text
buffer, with two independent front ends sharing one editor core: a GUI built
on [Gio](https://gioui.org) and a TUI built on
[tview](https://github.com/rivo/tview)/tcell. Both front ends drive the same
`internal/editor.Editor` orchestrator, which owns the piece-table buffer,
cursor/position math, and a vim-style modal keybinding engine.

The project is small and mid-refactor (see "Known issues" below) — treat this
file, not `README.md`/`CONCEPTUAL_GUIDE.md`, as the source of truth for
current package structure. See `BACKLOG.md` for known, deliberately-deferred
issues that don't block the current phase — pick those up opportunistically
or when they actually bite, not proactively.

## Vision & direction

Origin: the project started as a reaction to an Electron-based note app that
treated vim keybindings as an afterthought. The author dislikes Electron and
set out to build the same idea from scratch in Go — starting with the piece
table as a small standalone package, then growing it into a full editor.

The goal is a **real, daily-driver editor with nvim as its baseline** — not
primarily a learning exercise on piece-table internals (despite how the
current code reads: casual commit history, a demo-caption typing simulation,
and a narrow feature set all suggest a hobby project, but that's just where
it is early on, not the ceiling). Concretely:

- `cmd/gui` and `cmd/tui` are "the application." Both front ends are meant to
  reach **full parity** over time — divergence between them (like `gui-base`
  bypassing the keymap router today) is a bug to fix, not an acceptable
  stylistic difference.
- **Target platforms**: Gio compiled to Mac/Windows/Linux (and eventually
  Android/iOS), tview for the terminal, and eventually a from-scratch web
  build (Gio has real, if early, WASM support, so this is a plausible
  extension of the same GUI code, not a separate project). No strict
  priority between platforms, but all are always "in scope": desktop +
  terminal are what's actually being built now; mobile/web aren't
  implemented yet, but architecture should stay flexible enough to
  accommodate them later. Between the two, web is considered the more
  valuable future target; mobile is considered more niche (different
  interaction model — keyboard-first modal input doesn't map cleanly to
  touch).
- **Competitive positioning**: the explicit reference points are JetBrains
  (feature-rich but heavy — both under load on large files *and at idle*,
  due to the JVM plus an always-on whole-project background indexer), VS
  Code (approachable and extensible, but shares large-file weaknesses and
  is Electron-heavy at idle), and Neovim (fast, native modal editing, but
  requires heavy plugin assembly to feel IDE-like and is terminal-only, a
  real adoption barrier). The strategy is not to out-feature JetBrains/VS
  Code — that's not a winnable fight for a solo project — but to combine
  Neovim's native (not emulated) modal editing with dual GUI+terminal
  delivery (removing Neovim's biggest adoption wall) while making
  performance **both at idle and on huge files** the axis this editor is
  unambiguously better on than either incumbent.
- **The IDE pivot**: the original goal was a markdown-focused editor. That
  has expanded to a full **IDE ambition** — once the foundation supports
  treesitter/LSP at all, the incremental step from "markdown support" to
  "broad language support" via treesitter's existing language presets plus
  LSP is small, so the plan is to go all the way rather than stop at
  markdown. This is *not* in tension with staying lightweight: LSP's
  protocol design explicitly externalizes expensive language analysis into
  a separate per-project server process (lightweight client + full LSP was
  the protocol's founding goal), and treesitter is built for fast
  incremental re-parsing (the same reason Neovim/Zed/Helix use it) — not
  inherently heavy. The real risk is *scope*: any per-keystroke
  syntax/LSP work should be bounded to the visible offset range (the same
  viewport mechanism already planned for text rendering, see Known issues),
  not applied to the whole buffer. Roadmap staging for this is below —
  **do not let it pull scope or urgency into current work**.
- **Codebase comprehensibility** is an explicit goal, not a side effect:
  the codebase should be understandable by others with few pointers — a
  one-page README laying out structure, with everything else following the
  same repeating pattern. This is the same lever as the action + offset
  principle below pulling double duty: if piece-table ops, motions, text
  objects, the viewport, and eventually treesitter queries all reduce to
  one vocabulary, a newcomer learns one mental model and can navigate
  everything with it. The risk to this goal is the "exceptions" (undo/redo
  shape, mode-only commands, etc.) — treat "does this really need to be an
  exception, or can it fit the existing shape" as a real design gate each
  time one comes up, not a formality.
- The feature set is expected to grow well beyond today's modal
  editing + save/load — search/replace, visual-mode operations, multiple
  buffers/splits, redo, syntax highlighting, and similar are on the table.
  Favor real, robust solutions over quick/minimal patches when extending
  this codebase.
- `cmd/api` is explicitly **not** one of the consumer-facing apps. Its future
  is undetermined, but the current direction is toward making the editor
  "modifiable" — possibly a scripting/extensibility layer (Lua was floated,
  in the spirit of Neovim). Don't treat it as dead code to delete outright;
  today it's just a broken demo harness for exercising the core engine (see
  Known issues), but it may become the home for config/extensibility APIs.
- **Gio**: chosen deliberately for its trade-off — minimal built-in widgets,
  a steep initial learning curve, but complete rendering freedom and a
  genuinely lightweight immediate-mode model (no persistent retained widget
  tree, no work except in direct response to a frame event — this is most
  of why idle-lightness above is close to the default state of this stack
  rather than something to separately engineer). `internal/gui-base` today
  is ad hoc inline rendering; when it's eventually rebuilt, the plan is to
  apply a Flutter-style declarative widget-composition layer (small
  interfaces; params that accept and nest other components) on top of
  Gio's primitives, mirroring a pattern already built this way in a
  separate prior project — not to improvise a new approach at that time.

## Roadmap

Informal — the user's own framing is "not a concrete roadmap, but one can be
derived from it." Governing rule throughout: **keep future stages in mind
architecturally so nothing forecloses them, but don't build them until
reached** (same rule already applied to platform targets above).

### Path to stage 1 (agreed 2026-09-06)

1. **Finish the key engine foundation** (`key_engine.go`). Focus is the
   *mechanism*, not full keybinding coverage — keybindings get added along
   the way, some now, more later. Concrete exit test (not a feeling):
   adding a couple of new keybindings (e.g. a movement verb, a new text
   object) should require only *data* — a table entry plus a resolver — no
   dispatch-logic changes. `u` (undo) needs real backing here, which means
   the still-open undo/redo model decision (see below) will likely get
   forced during this phase rather than staying comfortably deferred.
   While here, also just fix (don't defer) two small, cheap items: wire the
   already-existing `anchors` parameter into `MoveCursorUp`/`MoveCursorDown`
   instead of always rescanning from offset 0, and add the missing bounds
   guard in `MoveCursorDown` (no guard symmetric to `MoveCursorUp`'s).
2. **Audit for contradictions with the core principles** (chiefly
   performance) before frontend work resumes — not a full re-audit, a
   targeted check for anything that actively fights the architecture.
   Known deferred items go in `BACKLOG.md`, not fixed proactively; e.g. the
   piece table's O(P) linear scan (`FindPieceAt`) is deliberately deferred
   because once viewport slicing exists it scales with piece count from
   editing activity, not file size — genuinely low-priority for a
   single-file MVP session, not a blocker being ignored.
3. **Make both front ends actually functional** (`gui-base`, `tui-base`) —
   this is where `gui-base` finally gets rewired through the key engine
   instead of its own inline key handling. A backend without a usable
   front end has no value to an actual user (even if that user is just the
   author, or an alpha release) — this is the point of the whole exercise,
   not a nice-to-have once the backend is "perfect."

Stage 1 (MVP) is reached once the above three hold. From there, proceed
through the numbered stages below, staying value-driven per release rather
than jumping straight to full IDE breadth.

1. **MVP**: single-file editing, full vim bindings, open/save, smooth on a
   500MB file. Find/replace via nvim's command-style (`:s///`), not a new
   modal UI — reuses existing command-mode machinery and stays
   frontend-agnostic.
2. **Single markdown/HTML file with preview**: the deliberately
   low-entry-cost proof case for treesitter + LSP — one file, one grammar,
   no multi-file complexity. HTML specifically because it lets vim motions
   extend into syntax-tree-aware text objects (e.g. "change inside tag"),
   a good stepping stone from character/word-based resolvers to tree-based
   ones. Open question not yet resolved: whether "preview" is a narrow,
   special-cased read-only render or the first real instance of multi-pane
   arriving a stage early — worth a deliberate decision, not a default.
3. **Multi-file / workspace** (explicitly deferred, not designed yet):
   splits, multiple open files, workspace-wide search/replace. The actual
   boundary of the action + offset model against multi-file operations is
   meant to be discovered by building against real requirements here, not
   theorized in advance. Proposed shape for multi-buffer: one self-contained
   buffer unit (today's `Editor`, unchanged) per open file/pane, plus an
   outer "current buffer" selector — matches vim's actual buffer/window
   separation. Already fully accommodated by the current codebase with zero
   changes needed, since `Editor` has no package-level/global state
   (`NewEditor`/`NewEditorFromFile` are already safe to call N times
   independently); what's net-new when built is a `Workspace`/`Session`
   type plus `cmd/gui`/`cmd/tui`'s entry point changing shape to accept it.
   Workspace-wide search is planned via an external/embedded ripgrep-style
   tool (matches VS Code/Helix/Sublime precedent) rather than forcing search
   into the per-buffer offset+action model — search is a separate,
   disk-level subsystem whose only interface to the rest of the system is a
   `(file, offset)` match consumed via existing per-buffer primitives.
4. **Full IDE** (roughly "step 4-5"): broad language support via
   treesitter's presets + LSP, per the IDE pivot above. Debugging
   (breakpoints, stepping, process/call-stack state) hasn't been discussed
   at all yet and is a genuinely separate domain — it won't fit the
   piece-table/offset model the way editing and search do, and will need
   its own subsystem, likely later than stage 3.

**Still genuinely open, not yet addressed by any stage above:**
- **Undo/redo model** — action-log-based (undo replays the inverse of the
  last action, redo re-applies it) vs. staying snapshot-based. Raised early
  on, never settled.
- **Concurrency/async orchestration** — how the synchronous, single-mutex
  editor core reconciles late-arriving async results (LSP diagnostics,
  background indexing, file-watcher events) without freezing the UI or
  racing edits. Partially de-risked by the multi-buffer shape aligning with
  LSP's per-document model, but the scheduling question itself is
  unaddressed — correctly so, until LSP's actual behavior is well
  understood.
- **Plugin API surface breadth** — the uniform blackbox-attachment
  discipline (every capability, frontend or not, attaches to the editor's
  facade the same interchangeable way) covers core editing actions;
  lifecycle hooks, UI extension points, and an API-stability contract
  aren't designed yet, appropriately deferred until real external plugin
  consumers exist.

## Design philosophy

~80% of design decisions here trace back to three principles, in this order:

1. **Performance** first.
2. **Modularity measured by decoupling, not line count** — prefer more,
   smaller, decoupled units over minimizing total lines.
3. **Interfaces/black-box boundaries without deep abstraction layering** —
   components satisfy interfaces or are opaque to each other, without
   stacking indirection on top.

**Delegation model**: `internal/editor.Editor` is the orchestrator and the
spine — the backbone everything connects through, owning all state (mode,
cursor, etc.) and performing actions from key input. `internal/piecetable`
is the crown jewel — the actual reason this project exists; the editor's job
is to connect it to everything else, not to be the interesting part itself.
Front ends (`gui-base`/Gio, `tui-base`/tview) are meant to be **dumb**: they
render whatever text/state the editor exposes, capture raw keystrokes, and
forward them to the editor — no editing logic belongs in a front end.

**Decoupling rule**: every component should have no direct knowledge of any
other component — either it's a self-contained black box, or the
relationship is mediated entirely through an interface. This is why, e.g.,
`viewmanager.Document` duplicates `editor`'s own `Document`-shaped interface
instead of sharing one type — that's intentional decoupling, not accidental
duplication to merge away. This is a **guiding principle for new/touched
code**, not a mandate to retrofit every existing violation immediately
(judge existing cases — like tests reaching into unexported `Editor`
fields — on their own merits).

**Simple, not easy**: the keybinding foundation (`key_engine.go`) is meant to
be genuinely hard to get right once, so that everything built on top of it
stays simple. The concrete test for whether this has actually been achieved:
adding a new keybinding/text-object later should mean adding *data* (a new
verb/modifier/noun table entry plus a resolver function), never touching the
core dispatch logic. If a future feature requires editing the execution path
rather than just registering into the tables, the foundation hasn't reached
"simple" yet. This same decoupled black-box boundary (editor ↔ frontends) is
intended to double as the eventual plugin/scripting API surface (e.g. a
future Lua layer) rather than needing a second, separately-designed API —
mirroring how Neovim's own RPC API is the same API used by both external
GUIs and Lua plugins, not a separate surface bolted on next to it.

**Action + offset**: the piece table only ever knows an offset
(`start, length`) and an action (`Insert`/`Delete`) applied to it. The
keybinding architecture (see `internal/editor/key_engine.go` below) is
designed to reduce every vim-style verb/modifier/noun combination — and
eventually the rendering viewport — to that same shape: a motion/text-object
*resolves* an offset, and a verb *applies* an action to it. The same
resolver is reused whether a noun is reached after an operator verb or
pressed standalone as a "silent verb" (e.g. `w`'s offset logic is identical
for `dw` and for bare `w`-as-move).

## Current phase

Front ends were wired up early "for gratification" and have since been
deliberately backgrounded while the editor/piece-table core got priority. A
full backend audit (piecetable, editor, keymap, viewmanager) was completed
2026-09-06 against the principles above, aimed at reaching "backend works
properly enough for a decent frontend MVP" — see Known issues below for the
resulting findings. **Before frontend work resumes**, the plan is to finish
the keybinding engine (`internal/editor/key_engine.go`, see Architecture
below) and work through the backend punch list. Don't jump straight into
frontend fixes unprompted.

## Module & toolchain

- Module: `github.com/fliplucky/pieces-store`
- Go version: `1.26.2`
- Key dependencies:
  - `gioui.org` — immediate-mode GUI toolkit, used by `internal/gui-base`
  - `github.com/rivo/tview` (+ `github.com/gdamore/tcell/v2`) — terminal UI toolkit, used by `internal/tui-base`
- No CI configuration and no linter config exist yet. `go vet` and `go test`
  are the only automated checks.

## Architecture / package map

```
cmd/gui, cmd/tui  →  gui-base / tui-base  →  editor  →  keymap, piecetable, viewmanager
cmd/api           →  piecestore (deleted — broken, see Known issues)
```

- **`cmd/gui`, `cmd/tui`** — working entry points. Each builds an
  `editor.Editor` (`editor.NewEditor(...)` or `editor.NewEditorFromFile(path)`)
  and hands it to the corresponding front end (`guibase.CreateApp(ed).Run()` /
  `tuibase.Boot(ed)`).
- **`cmd/api`** — not a consumer-facing app; currently just a broken demo
  harness for exercising the core engine directly (see Known issues). Its
  long-term role is undetermined but may become an extensibility/scripting
  entry point (see Vision & direction).
- **`internal/piecetable`** — the core text data structure: `Table` holds
  `Master`/`Add` byte buffers plus a `Pieces` list; `Insert`/`Delete`/`Undo`
  mutate it with history-based undo, `Coalesce` merges adjacent same-buffer
  pieces. Also handles file load/save (`NewPieceTableFromFile`, `Save`,
  `SaveAs`). This is a mutex-protected rewrite of the older, now-deleted
  `internal/piecestore` package (`Store` → `Table`).
- **`internal/editor`** — the orchestrator: `Editor` wraps a
  `*piecetable.Table`, a `*viewmanager.RuneCalculator`/`VirtualGrid`/`Cursor`,
  and (currently) a `*keymap.Router`. Public surface: text ops (`InsertText`,
  `DeleteText`, `GetText`), cursor movement, file I/O, vim-style command mode
  (`:w`, `:e`, `:q`, `:wq` via `ExecuteCommand`), `Undo`, and `HandleKey`
  (forwards to the keymap router — see below, this is expected to change).
  This is "the point of contact for the gui or tui" per its package doc
  comment.
- **`internal/editor/keymap`** — **the old, crude first implementation** of
  the keybinding engine, currently still the live wired path (`Router` walks
  a `map[mode]map[string]*KeyNode` graph to parse verb/modifier/noun
  sequences like `3dw`/`diw`). Confirmed by the author to be slated for
  replacement and deletion, not repair — see `key_engine.go` below and Known
  issues.
- **`internal/editor/key_engine.go`** — **the intended path forward** for
  keybindings, currently unfinished and not yet wired into `Editor`. Built on
  an "action + offset" principle (see Design philosophy above): a
  `CommandContext` parses verb/modifier/noun key sequences against semantic
  tables (`CreateVerbs`/`CreateModifiers`/`CreateNouns`) cross-checked
  against a per-key whitelist (`KeyAction.allowedKeyActions`), producing an
  `ExecutableCommand{Count, Verb, Modifier, Noun}`. Not yet implemented:
  offset resolution (wiring in `viewmanager.RuneCalculator`'s motion
  functions), an execution/dispatch step that applies the resulting action
  via `piecetable`/`editor`, most verb/modifier/noun table entries, and
  removal of `internal/editor/keymap` once this replaces it. See Known
  issues for the concrete gaps found in the 2026-09-06 audit.
- **`internal/viewmanager`** — presentation/position math extracted out of
  `internal/editor`: `Cursor`/`Mode` (Normal/Insert/Visual/Command),
  `RuneCalculator` (UTF-8-aware byte-offset ↔ row/col conversion and vim
  motions like word-forward/back, inner-word ranges), and `VirtualGrid`
  (viewport mapping — currently a no-op passthrough, see Known issues).
- **`internal/gui-base`**, **`internal/tui-base`** — thin, single-file
  (`base.go`) front ends, one per toolkit. Both hold an `*editor.Editor` and
  render a Tokyo Night–themed view + status bar. `tui-base` routes all
  keystrokes through `ed.HandleKey` (i.e. the keymap router); `gui-base` does
  **not** — it hand-rolls its own key handling inline (see Known issues).
- **`platform/`** — empty placeholder for future platform-specific code.

## Build / test / run

Primary interface is the `Taskfile.yaml` (requires [Task](https://taskfile.dev)):

- `task fmt` — `go fmt ./...`
- `task vet` — `go vet ./...`
- `task test` — `go test -v ./...` (run this to execute the test suite)
- `task build` — builds local binaries into `tmp/`
- `task clean` — `go clean && rm -rf tmp/*`
- `task gui` / `task tui` — run the GUI (Docker) or TUI (`go run ./cmd/tui/main.go`, host-interactive)
- `task tui-docker`, `task dev`, `task up`, `task down` — Docker Compose services with Air hot reload

Docker Compose defines `gui`, `tui`, and `dev` services, each running Air with
its own config (`.air.gui.toml`, `.air.tui.toml`, `.air.dev.toml`).

`go build ./...`, `go vet ./...`, and `go test ./...` all pass cleanly as of
2026-09-06 (the two build-breakers in Known Issues are fixed).

## Testing conventions

- Standard library `testing` only — no testify or other assertion libraries.
- Tests are white-box and same-package, often reaching into unexported fields
  directly (see `internal/editor/editor_test.go`).
- For seams that need faking, prefer a hand-written mock/test-double type
  implementing the relevant interface rather than a mocking library — e.g.
  `keymap`'s tests use a `MockEditor` implementing `EditorInterface`, and
  `viewmanager`'s tests use a small `ByteDocument` implementing `Document`.

## Known issues / in-progress refactor

The tree is mid-refactor (piece-table renamed from `piecestore` to
`piecetable`; cursor/grid logic split out of `internal/editor` into
`internal/viewmanager`). Known gaps as of now:

1. ~~`cmd/api/main.go` doesn't compile~~ **Fixed 2026-09-06** — now imports
   `internal/piecetable`/`piecetable.NewPieceTable` instead of the deleted
   `internal/piecestore`. `go build ./...` passes.
2. **`internal/editor/key_engine.go`'s build-breaking unused import was
   fixed 2026-09-06** (the `viewmanager` import was removed — no offset-
   resolution wiring exists yet, so there was nothing to keep it for). This
   is *not* dead code: it's the intended keybinding engine, still very
   incomplete. Only 2 verbs (`d`/`c`), 1 modifier (`i`; `a` is referenced
   but has no table entry), and 1 noun (`w`; `LineMotion`/`RuneMotion`
   exist as enum values with no table entries) are registered; no offset
   resolution or action-execution/dispatch step exists yet; a leading `"0"`
   keystroke is unconditionally swallowed into the count accumulator before
   any verb/noun lookup runs; and the design work for finishing it
   (a `availableDirects` table for immediately-executing keys, generic
   doubled-verb handling for `dd`/`yy`/`cc`, an error path distinguishing
   "not a sequence" from "whitelist says yes but the semantic table has
   nothing") is settled but not yet implemented.
3. **`internal/editor/keymap` is the old, crude first implementation** of
   the keybinding engine — confirmed by the author to be slated for deletion
   once `key_engine.go` is finished and wired into `Editor.HandleKey`. Don't
   invest in fixing bugs inside it (e.g. `diw`/`dw`/`dd`/`x` all collapsing
   onto the same non-directional `DeleteText()` primitive, or Visual mode
   being entirely unreachable — no registered graph, no entry keybinding) —
   that code is being replaced, not repaired.
4. **`internal/editor/interfaces.go`'s `Document` interface is orphaned** —
   unused anywhere in the codebase; `viewmanager.Document` is the one
   actually consumed by `RuneCalculator`.
5. **`gui-base` bypasses the keymap router** — it implements its own inline
   key-handling switch instead of calling `ed.HandleKey` like `tui-base`
   does, and as a result cannot save or quit at all. Since GUI/TUI parity is
   an explicit goal (see Vision & direction), this is a real bug to fix, not
   just a stylistic inconsistency — but it should be fixed against whatever
   replaces `internal/editor/keymap`, not against the old package.
6. **`internal/viewmanager/virtual_grid.go`'s `GetScreenPosition` is a no-op**
   passthrough — scroll offsets, line wrapping, tab expansion, and a
   line-number gutter aren't implemented yet. Both front ends currently do
   naive full-buffer `GetText()` + `strings.Split` on every keystroke *and*
   every cursor move, which is also a significant performance concern (see
   Design philosophy's "performance first").
7. **Cross-cutting backend concerns found in the 2026-09-06 audit**: `piecetable.GetRuneAt`/`FindPieceAt`
   bypass their own mutex on the actual read path (a real, if currently
   latent, data race); `GetRange` panics on out-of-range/reversed input;
   `Insert` with a negative offset silently misplaces instead of erroring;
   no dirty/unsaved-changes tracking exists, so `:q` quits unconditionally;
   there is no Redo (`Undo` destructively pops history); and cursor
   movement/typing recompute row/col by scanning from byte offset 0 on every
   keystroke (the `anchors` parameter `PositionToByteOffset` already exposes
   for this is never used by any caller).
8. **`README.md` and `CONCEPTUAL_GUIDE.md` are stale** — they predate the
   `piecestore` → `piecetable` rename and the `viewmanager` extraction, and
   describe outdated package/file paths. Use this file's architecture section
   instead.
