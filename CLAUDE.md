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
or when they actually bite, not proactively. See `FEATURES.md` for the
product-facing view (what the editor actually does today, how it got here,
and where it's going) — this file's Known Issues section is the
engineering log, `FEATURES.md` is the roadmap.

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

1. ~~**Finish the key engine foundation**~~ **Done 2026-09-13** — see
   Current phase. Exit test was met: `w`/`b`/`e`/`u`/`<C-r>` (and the rest
   of the 14-case list) are all data (table entries + resolvers) on top of
   the same dispatch mechanism, no dispatch-logic changes needed per
   keybinding. The undo/redo model got decided as part of this (symmetric
   redo stack on `piecetable.Table`, not the bigger action-log redesign —
   see `piecetable/writer.go`'s `Undo`/`Redo`). The two cheap fixes
   (`anchors` wiring, `MoveCursorDown` bounds guard) are done too.
2. **Audit for contradictions with the core principles** (chiefly
   performance) before frontend work resumes — not a full re-audit, a
   targeted check for anything that actively fights the architecture.
   Known deferred items go in `BACKLOG.md`, not fixed proactively; e.g. the
   piece table's O(P) linear scan (`FindPieceAt`) is deliberately deferred
   because once viewport slicing exists it scales with piece count from
   editing activity, not file size — genuinely low-priority for a
   single-file MVP session, not a blocker being ignored. (Two real
   correctness bugs were found and fixed as a side effect of building the
   executor, not from a dedicated audit pass — see `internal/offset`'s
   `spanStart`/`RangeInnerWord`/`MotionWordBackward` history.)
3. ~~**Make both front ends actually functional**~~ **Done 2026-09-19** —
   `gui-base` now routes every keystroke through `Editor.HandleKey`/
   `InsertLiteralText` (see Current phase and Known issues history), the
   same as `tui-base` already did. Both front ends can open/save/quit, run
   the full 14-case keybinding list, and undo/redo for real.

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
- ~~**Undo/redo model**~~ **Settled 2026-09-19** — stayed snapshot-based
  (not the full action-log rewrite once floated) but each `historyEntry`
  now also carries the precise `Edit{Offset, OldLength, NewLength}` that
  produced it, so `Undo`/`Redo` report an exact reversible delta instead
  of just restoring a snapshot with no idea what changed. Forced by the
  treesitter/LSP audit below — incremental re-parsing and `didChange`
  both need exactly this shape, and diffing two full snapshots to
  reconstruct it on every undo would have been real per-undo cost on a
  large file. See `piecetable/table.go`'s `Edit`/`historyEntry` and
  `writer.go`'s `Undo`/`Redo`.
- **Concurrency/async orchestration** — how the synchronous, single-mutex
  editor core reconciles late-arriving async results (LSP diagnostics,
  background indexing, file-watcher events) without freezing the UI or
  racing edits. `Editor.ChangeChan()` now carries a real payload
  (`ChangeEvent{Edit}`, settled 2026-09-19 alongside the undo/redo fix
  above) instead of a bare `chan struct{}`, so a future consumer can at
  least learn *what* changed — but it's still explicitly a coalescing,
  best-effort "please redraw" channel (a burst of edits collapses to the
  latest one, by design — see its doc comment), not a lossless ordered
  stream. Anything that must see every edit exactly once, in order (e.g.
  incremental treesitter re-parsing), needs a separate, non-coalescing
  mechanism that doesn't exist yet. Partially de-risked by the
  multi-buffer shape aligning with LSP's per-document model, but the
  scheduling question itself is unaddressed — correctly so, until LSP's
  actual behavior is well
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
keybinding architecture (see `internal/keyengine` below) is
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
properly enough for a decent frontend MVP."

**The keybinding engine is now finished and wired in (as of 2026-09-13)**:
`internal/editor/keymap` (the old, crude first implementation) has been
deleted entirely; `internal/keyengine` (see Architecture below) is the live
path, reached via `Editor.HandleKey` → mode-dispatch → executor
(`internal/editor/dispatch.go`). All 14 originally-agreed keybindings
(`diw`, `daw`, `dip`, `dd`, `w`, `b`, `e`, `u`, `<C-r>`, `r`, `:`, `i`, `o`,
`O`) work end-to-end through a real `Editor`, verified by tests that
actually type sequences through `HandleKey` and check resulting
text/cursor/mode state, not just parser-level unit tests. Undo/Redo is
real (`piecetable.Table.Redo`, a symmetric redo stack — see Known issues
history), and `:q` now refuses on unsaved changes (`:q!` force-quits).

**`gui-base` is now wired through the engine too (as of 2026-09-19)**:
`internal/gui-base/base.go` was rebuilt to route every keystroke through
`Editor.HandleKey` (translating Gio's `key.EditEvent`/`key.Event` into the
same key-string vocabulary `tui-base` already used), call
`Editor.InsertLiteralText` for multi-rune paste/IME chunks (dropped
outside Insert mode, per `InsertLiteralText`'s own contract), and render
via the new `Editor.Viewport`/`viewmanager.ViewportSlice` (a scroll
position that follows the cursor, plus margin) instead of full-buffer
`GetText()` + `strings.Split` on every frame. It also gained a block
cursor in Normal/Visual/Command modes vs. a pipe/bar cursor in Insert
mode, matching vim's own convention, and a status-bar line for
`GetLastCommandError()` (e.g. why `:q` was refused). Stage 1 (MVP) per the
Roadmap is now reached: both front ends are real, functional editors on
top of the same engine, not just `tui-base` alone.

**Next up**: none of the MVP staging items are individually assigned yet
— see the Roadmap's numbered stages (1. MVP polish/find-replace, 2.
markdown/HTML+treesitter/LSP, ...) for what's next in line, plus
`BACKLOG.md`/Known issues for anything that surfaces opportunistically.
Visual mode (Known issues item 2) and `tui-base`'s still-naive full-buffer
render (Known issues item 3) are the most likely near-term picks.

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
cmd/gui, cmd/tui  →  gui-base / tui-base  →  editor  →  keyengine, offset, piecetable, viewmanager
cmd/api           →  piecetable (see Known issues — cmd/api's role is undetermined but no longer broken)
```

`offset` and `keyengine` are both leaf packages (zero internal dependencies
of their own — `keyengine` needs only `types`) — deliberately, so neither
can end up in a dependency cycle with `editor` or with each other.

- **`cmd/gui`, `cmd/tui`** — working entry points. Each builds an
  `editor.Editor` (`editor.NewEditor(...)` or `editor.NewEditorFromFile(path)`)
  and hands it to the corresponding front end (`guibase.CreateApp(ed).Run()` /
  `tuibase.Boot(ed)`).
- **`cmd/api`** — not a consumer-facing app; currently just a broken demo
  harness for exercising the core engine directly (see Known issues). Its
  long-term role is undetermined but may become an extensibility/scripting
  entry point (see Vision & direction).
- **`internal/piecetable`** — the core text data structure: `Table` holds
  `Master`/`Add` byte buffers plus a `Pieces` list; `Insert`/`Delete`
  mutate it, `Coalesce` merges adjacent same-buffer pieces. `Undo`/`Redo`
  are snapshot-based (each unexported `historyEntry` restores a full
  `State` directly, no replay) but each entry also carries the `Edit`
  that produced it (`Edit{Offset, OldLength, NewLength}`), so `Undo`/`Redo`
  return the precise reversible delta rather than leaving callers to
  diff two snapshots to find out what changed (settled 2026-09-19, see
  Roadmap's "Still genuinely open" history). Also handles file load/save
  (`NewPieceTableFromFile`, `Save`, `SaveAs`). This is a mutex-protected
  rewrite of the older, now-deleted `internal/piecestore` package
  (`Store` → `Table`).
- **`internal/editor`** — the orchestrator: `Editor` wraps a
  `*piecetable.Table`, a `*viewmanager.VirtualGrid`, a local `*Cursor`
  (`cursor.go` — position/mode/command-buffer; moved here from
  `viewmanager` since it's editor state, not view state), and a
  `*keyengine.CommandContext`. `HandleKey` (in `dispatch.go`) is the single
  entry point for all keyboard input: it routes on mode — Insert/Command
  mode capture keys literally (no vim grammar applies to "just type this
  character"); Normal (and eventually Visual) mode hands the key to
  `keyengine.CommandContext.ProcessCommand`, and once a sequence resolves,
  `execute()` (also in `dispatch.go`) resolves the command's offset/range
  via `internal/offset` and applies the verb via `piecetable`/`Editor`.
  Also handles file I/O, vim-style command mode (`:w`, `:e`, `:q`/`:q!`,
  `:wq` via `ExecuteCommand`), and `Undo`/`Redo`. This is "the point of
  contact for the gui or tui" per its package doc comment.
- **`internal/keyengine`** — parses vim-style keystroke sequences into a
  resolved `ExecutableCommand{Count, Verb, Modifier, Noun}`. Zero
  dependency on `piecetable`, `offset`, or `Editor` — keystrokes in, a
  command (or an `ErrInconsistentKeymap` error) out. Split across
  `keyengine.go` (core parser: `CommandContext.ProcessCommand`, the
  verb→modifier→noun state machine, the generic doubled-verb check for
  `dd`/future `yy`/`cc`), `normal_mode.go` (the real, populated tables), and
  `visual_mode.go` (placeholder scaffolding — no Visual keybinding is
  implemented yet, see Known issues). A fourth table beyond
  verb/modifier/noun, `availableDirects`
  (`map[string]DirectAction{Verb, Modifier, Noun}`), covers keys that
  execute immediately with no further input: `w`/`b`/`e` (silent motions,
  reusing the same noun identities `diw`/`daw` resolve), `u`/`<C-r>`/`r`
  (editor-level commands), `:`/`i`/`o`/`O` (mode switches / line-opening).
  This replaced `internal/editor/keymap` (deleted 2026-09-13) entirely.
- **`internal/types`** — the shared kernel: small, dependency-free enums
  every other package agrees on, imported freely but never importing
  anything else in the module itself. `Mode` (Normal/Insert/Visual/
  Command), `Style` (a small syntax-highlighting/diagnostic category
  enum — see `viewmanager`'s entry below), and `Language` +
  `DetectLanguage(filePath string)` (extension-only detection — settled
  2026-09-19, a small starter set: Markdown/HTML/Go/JSON/JavaScript/CSS,
  `LanguagePlainText` as the fallback for both an unsaved buffer and an
  unrecognized extension). Exposed via `Editor.Language()`, computed
  fresh each call rather than cached. Nothing consumes it yet — it exists
  for a future syntax analyzer (or LSP client) to know which
  grammar/languageId to use; this was the last treesitter-blocking gap
  from the 2026-09-19 architecture review (see Roadmap history).
- **`internal/offset`** — text-position/structure math, with zero
  dependency on anything else in the module (usable by cursor movement,
  the key engine's resolvers, or anything else that ever needs "where does
  this word/paragraph start and end," e.g. a future double-click-to-select
  word — independent of vim keybindings). `position.go`: byte↔`Position`
  (row/col) conversion, UTF-8-safe rune stepping (`MoveLeft`/`MoveRight`).
  `motions.go`: `FindOffset` (the shared scanning primitive — "the main
  magic for keybindings"), predicates, and the vim motion/text-object
  resolvers built on it (`MotionWordForward`/`Backward`/`End`,
  `RangeInnerWord`/`AroundWord`/`Paragraph`, `RangeLine`,
  `MotionFindCharForward`). `ScanUntil` is still dead code (see Known
  issues, unchanged).
- **`internal/viewmanager`** — screen/viewport mapping, nothing else
  (slimmed 2026-09-13 — `Cursor` moved to `editor`, position math and
  motion resolvers moved to `offset`). `VirtualGrid`/`GetScreenPosition`
  (per-position mapping — currently a no-op passthrough, see Known issues)
  and `viewport.go`'s `ViewportSlice(doc, topRow, visibleRows, margin)` (a
  windowed read: just the lines worth rendering, recomputed fresh on every
  call — deliberately uncached, since `piecetable.GetRange` on a
  viewport-sized range is already cheap regardless of document size).
  `Slice` also carries `LineOffsets`/`EndOffset` (each line's absolute
  byte start, and one past the last line) — not needed by text rendering
  itself, but by placing `StyledSpan`s against specific runes (see below).
  Exposed to frontends via `Editor.Viewport(...)`, not called directly —
  same decoupling reasoning as everywhere else in this package map.
  `style.go` is the styled-rendering contract (settled 2026-09-19): a
  `StyledSpan{offset.TextRange, Style}` (reusing the same range vocabulary
  as motions/delete ranges, not a new one), `StyleAt`/`SpansForRange` as
  the pure lookup/filter functions, and `types.Style` (a small semantic
  category enum — keyword, string, diagnostic-error, etc. — colors are
  each frontend's own concern, matching how `Mode` already works).
  Exposed via `Editor.StyleSpans(start, end)`, which is real and tested
  but always returns empty today — nothing produces a real `StyledSpan`
  yet (no syntax analyzer exists). **Wired into both frontends' actual
  rendering as of 2026-09-19**: `gui-base`'s `renderLine` and `tui-base`'s
  per-line loop both fetch spans and color runes per `StyleAt`, falling
  back to the cheap plain-text path when a line has none (identical
  output to before `StyledSpan` existed, verified by tests) — so the
  moment a real producer populates `Editor.styleSpans`, highlighting
  renders with no further frontend changes needed.
- **`internal/gui-base`**, **`internal/tui-base`** — thin, single-file
  (`base.go`) front ends, one per toolkit. Both hold an `*editor.Editor` and
  render a Tokyo Night–themed view + status bar, and both route every
  keystroke through `ed.HandleKey` (`gui-base` translates Gio's
  `key.EditEvent`/`key.Event` into the same key-string vocabulary
  `tui-base` gets natively from tcell; multi-rune `key.EditEvent` text —
  paste/IME — goes through `ed.InsertLiteralText` instead). `gui-base`
  renders via `Editor.Viewport` (windowed, scroll-follows-cursor) with a
  block/pipe cursor per mode; `tui-base` still does full-buffer
  `GetText()` + `strings.Split` every frame (see Known issues item 3). Both
  now also color runes per `Editor.StyleSpans` (currently always empty —
  see `internal/viewmanager`'s `style.go` entry above), each frontend
  owning its own `Style` → color mapping (`gui-base`'s `styleColor`,
  `tui-base`'s `tviewStyleTag`). Both also render a wildmenu-style
  completion popup for `:e`/`:w` path
  completion above the status bar: `<Tab>` (`Editor.TriggerCompletion`)
  starts a cycle or confirms the current selection — descending into it
  (a fresh candidate list one directory deeper) if it's a directory,
  ending the cycle if it's a file — while `<C-n>`/`<C-p>`
  (`Editor.CycleCompletion`) only move the highlighted selection within
  the current list without ever descending. Windowed to
  `maxCompletionRows` so a large directory doesn't blow up the popup.
  - **Gio-specific gotchas worth knowing before touching `gui-base`'s key
    handling again** (all three cost real debugging time to find,
    2026-09-19): (1) a tag only receives `key.EditEvent` — the event
    carrying actual typed text — once something registers a
    `key.FocusFilter{Target: tag}` for it; without that, the router's
    `keyQueue.Frame` strips focus from the tag every single frame
    regardless of `key.FocusCmd`, so text input silently never arrives.
    (2) a bare `Tab` keypress is intercepted by Gio itself as a reserved
    system key for widget-to-widget focus cycling, and is deliberately
    withheld from a wildcard `key.Filter{}` — it must be claimed by name
    (`key.Filter{Name: key.NameTab}`) or it never reaches the app at all.
    (3) a wildcard `key.Filter{}` only matches events with *zero*
    modifiers — `keyFilterMatch` rejects any modifier bit not covered by
    the filter's `Required`/`Optional` fields, which default to zero — so
    every Ctrl-combo (`<C-r>`, `<C-n>`, `<C-p>`) was being silently
    dropped until the filter declared `Optional: key.ModCtrl`.
    `handleEvents`' `gtx.Event(...)` call registers all three filters
    together for exactly these reasons. General lesson: Gio's key
    filters are opt-in and precise by design — nothing is delivered "by
    default," so audit any wildcard filter for what it's silently
    excluding rather than assuming broad coverage.
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

Resolved history, kept briefly for context: `cmd/api/main.go` failing to
compile (fixed 2026-09-06, now imports `internal/piecetable`), the old
`internal/editor/keymap` engine (deleted entirely 2026-09-13, replaced by
`internal/keyengine`), `internal/editor/interfaces.go`'s orphaned
`Document` interface (deleted), the keybinding engine's incompleteness
(finished 2026-09-13 for the 14-case list — see Current phase), no
Redo (`piecetable.Table.Redo` added, a symmetric redo stack), no
dirty-tracking on `:q` (added — `Dirty` on `piecetable.Table`, `:q!` force-
quits), `MoveCursorDown`'s missing bounds check / unused `anchors`
param (fixed — see `editor.go`, anchoring only helps the Down direction,
Up genuinely can't benefit from it without also tracking each line's start
offset, see the comment there), and `gui-base` bypassing the key engine
with its own inline key-handling switch (rebuilt 2026-09-19 to route
through `Editor.HandleKey`/`InsertLiteralText` and render via
`Editor.Viewport` — see Current phase).

Known gaps as of now:

1. **Visual mode is unreachable** — `internal/keyengine/visual_mode.go` is
   still the original copy-paste placeholder (not fleshed out — see the
   comment there for why), and nothing enters Visual mode (no `v` key is
   registered anywhere). Not in scope until Visual mode is actually needed.
2. **`internal/viewmanager/virtual_grid.go`'s `GetScreenPosition` is a
   no-op** passthrough — scroll offsets, line wrapping, tab expansion, and
   a line-number gutter aren't implemented yet (distinct from
   `ViewportSlice`, which does the line-windowing `gui-base` now uses).
   `tui-base` still does naive full-buffer `GetText()` + `strings.Split`
   on every keystroke *and* every cursor move — `gui-base` no longer does,
   see Current phase — which is a real performance concern on large files
   (see Design philosophy's "performance first") once `tui-base` gets the
   same `Editor.Viewport` treatment.
3. **Cross-cutting backend concerns, still open**: `piecetable.GetRuneAt`/
   `FindPieceAt` bypass their own mutex on the actual read path (a real, if
   currently latent, data race); `Insert` with a negative offset silently
   misplaces instead of erroring (`GetRange` itself was fixed — it now
   clamps out-of-range/reversed input instead of panicking, see
   `piecetable/reader.go`/`table_test.go`'s `TestGetRange`). See
   `BACKLOG.md` for the full sidebar list (most of these are deliberately
   deferred, not overlooked).
4. **`ScanUntil`** (`internal/offset/motions.go`) is still dead code — same
   byte-stepping issue `FindOffset` used to have, looks like an abandoned
   attempt to simplify/replace it that never got finished. Not yet decided
   whether to delete it or complete the replacement.
5. **`README.md` and `CONCEPTUAL_GUIDE.md` are stale** — they predate the
   `piecestore` → `piecetable` rename and the whole `keyengine`/`offset`/
   `viewmanager` split. Use this file's architecture section instead.
