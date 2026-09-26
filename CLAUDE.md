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
  stream. `internal/syntax`'s incremental re-parsing (added 2026-09-19)
  sidesteps this rather than needing the non-coalescing mechanism this
  item originally called for: `Editor.moveCursorToLocked` calls
  `Highlighter.Update` directly and synchronously, inline with the edit
  itself, never through `ChangeChan` at all — so it never has anything to
  lose to coalescing in the first place. That escape hatch is specific to
  same-process consumers living inside `Editor`'s own locked methods; an
  out-of-process consumer (a real LSP client, eventually) can't use it and
  would still need a real solution here. Partially de-risked by the
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

The mental picture worth keeping for this (the author's own, from building
factory-automation games, and it holds surprisingly well): a **conveyor-belt
splitter**. A value arrives labeled with its type; the splitter reads the
label and routes it down the matching chute; nothing upstream ever needs to
know how many chutes exist or where they physically lead. Adding a new chute
later doesn't require rewiring the belt or touching anything already
connected to it — the classic open/closed shape (open to a new case, closed
against needing to change any existing caller to add it). Concretely,
`internal/clipboard-register`'s `getRegister(RegisterType) *Register`
*is* one of these splitters at a small scale: a caller supplies a type and a
value and gets a result, with zero knowledge of which of five struct fields
actually backs it. `Editor` is the same shape at a much larger scale,
routing to `keyengine`/`offset`/`piecetable`/`lspservice` instead of struct
fields — same splitter, different size.

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

**LSP integration, Part 1 shipped (2026-09-20)** — a real language-server
installer/manager (`internal/lspmanager`, `platform/`, `internal/editor`'s
new async bridge — see Architecture below for the full detail), the first
of a two-part plan (Part 2 — the LSP client itself plus format/diagnostics/
autocomplete/hover — is next; go-to-definition and everything it drags in
(multi-buffer/workspace, workspace-wide search) is a deliberately separate,
not-yet-started third plan). `:LspInstall <language>` genuinely installs a
real language server from mason-registry's live data — verified end-to-end
against real toolchains and the real registry, not mocked. Known, honestly-
scoped trims from the full plan, not yet done: HTML/CSS/JSON's shared npm
source (`vscode-langservers-extracted`) installs three separate times
instead of being deduplicated across catalog entries (correct, just
wasteful); `:LspUninstall` only removes the index record, not the
install directory itself (deliberately deferred — real filesystem deletion
of a version-pinned tree deserves its own attention); npm-installed
servers' Windows binary path (a `.cmd`/`.ps1` shim, not directly
`exec`-able) is a known, documented gap, not silently mishandled.

**LSP Part 2 core + diagnostics shipped, same day**: new package
`internal/lspclient` — a hand-rolled (not a library; the real message
surface, confirmed via a throwaway probe against real `gopls` first,
turned out small enough not to need one) JSON-RPC/LSP client:
`transport.go` (Content-Length framing), `client.go` (request/response
correlation, notification dispatch, auto-replies "method not found" to any
unsupported server-to-client request rather than hanging the server),
`lifecycle.go` (spawn, real `initialize`/`initialized` handshake, bounded
`Stop()`), `position.go` (byte-offset ↔ LSP `Position`, real UTF-16
code-unit math — confirmed empirically that `gopls` defaults to UTF-16
even when UTF-8 is offered), `sync.go`
(`didOpen`/`didChange`/`didClose`/`hover`/`completion`/`formatting`
wrappers, diagnostics parsing). `didChange` is full-sync (whole document
each time) — a deliberate v1 call since incremental sync needs pre-edit
content `Editor` doesn't snapshot — accepted explicitly by the user
("if you have a 500MB file with a complex LSP running, you're doing
something wrong").

Wired into `Editor` via new `internal/editor/lsp.go`: one lazily-started
server per buffer language (mirrors `setupHighlighterLocked`'s own
lazy-per-buffer pattern), spawned/initialized asynchronously via the
Part-1 `runAsync` bridge, `didChange` sent synchronously inline from
`moveCursorToLocked` right next to the tree-sitter highlighter's own
`Update` call (same "local pipe write, not a network call" reasoning).
Real diagnostics now flow into `Editor.StyleSpans` — placed first so they
override syntax color at the same byte — tagged against the currently-open
document's URI so a notification for a since-closed buffer is discarded.
**Needed zero new frontend rendering code**: both `gui-base` and
`tui-base` already had real color mappings for `StyleDiagnosticError`/
`StyleDiagnosticWarning` from the original styled-rendering-contract work
(2026-09-19) — only the producer was missing.

A real, if minor, test-isolation bug was caught the same session: wiring
LSP startup into every file-open path meant existing tests started
touching `platform.LSPServersDir()` — real `os.UserCacheDir()` — confirmed
to actually create `~/.cache/pieces-store/` on the real machine during a
plain `go test ./...`. Fixed with a package-wide `TestMain` in
`internal/editor` isolating the cache/config env vars for the whole test
binary.

Verified end-to-end against a real, spawned `gopls`: a real `.go` file
with an actual compile error, opened through a live `Editor`, produces a
real `StyleDiagnosticError` span in `StyleSpans` — install through
rendering-ready state, all real, not mocked.

**LSP Part 2 completed the same day** — formatting, hover, autocomplete,
and the cursor-anchored floating-popup UI all landed:

- **Formatting** (`:Format`): requests `textDocument/formatting`, applies
  the returned edits via new `Editor.applyTextEditsLocked` (multi-edit
  batch apply, sorted end-to-start so an earlier edit's offset is never
  invalidated by a later one already applied — a real, previously-
  nonexistent piece of `Editor` surface, since it only ever exposed
  single Insert/Delete before this).
- **Hover** (`K` in Normal mode, a new `keyengine.Hover` verb): async
  `textDocument/hover` request, result shown via new `Cursor.HoverState`,
  auto-dismissed on the next cursor move or mode change
  (`dismissHoverLocked`, called from `moveCursorToLocked`/`SetMode`).
- **Autocomplete**: triggers automatically on the server's own advertised
  trigger characters (e.g. `.` for `gopls`) while in Insert mode; browsed
  with `<C-n>`/`<C-p>` (already-existing key translations in both
  frontends needed zero changes — they were built for `:e`/`:w` path
  completion but are mode-agnostic), confirmed with `<Enter>`. A new
  `Cursor.LSPCompletionState` reuses `lspclient.CompletionItem` directly
  rather than the thinner `[]string`-based `CompletionState` path
  completion already uses — confirmed against real `gopls` output that
  real servers return `TextEdit`-based completions (a replace-this-range
  instruction), not plain insert text, and the apply logic handles both.
  A generation counter (`lspCompletionGeneration`) discards a slow
  response if the user kept typing past it.
- **Floating popups, new UI in both frontends** (the user's explicit
  choice over reusing the docked completion bar): `tui-base` uses
  `tview.Pages` with **exact terminal-cell positioning** (no
  approximation needed — `editorView.GetInnerRect()` plus the
  already-tracked cursor row/col give real integer coordinates).
  `gui-base` needed a new `layout.Stack` wrapping the whole top-level
  layout (previously a plain `layout.Flex`) so `op.Offset` could position
  an overlay — **honestly approximate, not pixel-exact**: Gio's own
  per-rune layout here never computes real pixel coordinates for the
  cursor (only byte offsets), so position is derived from
  `cursor.Row`/`Col` and font-size-derived cell dimensions
  (`charWidthPx`/`lineHeightPx` — the same approximation `followCursor`'s
  scroll math already used), not real op-recording. Flagged as a real,
  documented limitation worth revisiting if it's visually off enough to
  matter in daily use, not silently claimed as exact.

**Caught a real reentrant-mutex bug before it ever ran**, matching this
codebase's established `moveCursorUpLocked`/`moveCursorDownLocked`
precedent: the first `:Format` implementation called a public, locking
`StartFormat()` from inside `executeCommandLocked`, which itself runs
under `ExecuteCommand`'s already-held lock — a guaranteed deadlock on
first use. Caught by re-checking the actual call chain before running
anything, not by hitting the hang; split into a public `StartFormat()`
(locks, for any future caller that isn't already inside a locked method)
and `startFormatLocked()` (assumes the lock is already held, what
`executeCommandLocked` actually calls).

**Verified end-to-end against real `gopls` once more** — one comprehensive
test (`TestRealHoverAutocompleteAndFormatEndToEnd`) drives a single real
`gopls` session through hover (real text about `fmt.Println`), autocomplete
(29 real candidates from typing `fmt.`, confirming one actually mutates the
buffer correctly), and formatting (a deliberately mis-indented line — a
space instead of a tab — gets genuinely fixed) — all through a live
`Editor`, not mocked, and specifically chosen to be reentrant-lock-
sensitive (a locking mistake would hang the test rather than fail it
cleanly). Both real tagged production binaries (`gui-local`/`tui-local`,
built with the full `GRAMMAR_TAGS` set) still build clean. Frontend
rendering itself follows this codebase's existing, documented asymmetry:
`tui-base`'s new `wrapText` (pure logic) got real unit tests; `gui-base`'s
equivalent is its **first test file ever** for the same reason (pure
`wrapText` logic, no Gio context needed) — but the actual popup rendering
in both frontends is build+vet+smoke-run verified only, same as all
existing gui-base rendering code, since exercising a live popup requires
real keyboard input into a running window that isn't scriptable in this
environment.

**Diagnostics rendering changed from color-override to underline (same
day)**, after the user asked about squiggly underlines: `Editor.StyleSpans`
no longer includes diagnostics at all (reverted the earlier merge) — a new
`Editor.DiagnosticSpans(start, end)` exposes them separately, so a
frontend renders a token's syntax color *and* an independent diagnostic
underline rather than one replacing the other. `gui-base` draws a real,
independently-colored underline (`withDiagnosticUnderline`/
`diagnosticColor`) since it draws every glyph itself — no toolkit
limitation there. `tui-base` uses tview's real `u`/`U` tag attribute
(confirmed via reading tview's actual tag parser source, not assumed) —
but genuinely can't color the underline separately from the text color
through that API, a real, checked limitation of tview's tag-string
convenience layer (`tcell.Style.Underline` itself does support a color
parameter, just not reachable through tags) — documented in
`tviewStyleColor`'s doc comment rather than silently accepted. A real
attribute-leakage bug class was avoided by design: tview tags are sticky
across a styled string unless explicitly reset, so every emitted tag now
always states the underline attribute explicitly (`u` or `U`), never
relying on omission to mean "unchanged."

**Diagnostic severity color + virtual text, same day, after live user
testing**: user asked for the underline to actually show its severity
color (not inherit the text's own color) and for the diagnostic's message
to show as inline virtual text (vim/nvim-style). The color ask exposed the
real reason for `tui-base`'s "monochrome underline" limitation noted
above: tview ties underline to the same foreground color as the text, so
there's no way to underline in red while keeping a token's own syntax
color in the terminal. Resolved by accepting that constraint deliberately
— a diagnosed rune's color becomes its severity color in `tui-base`
(matching how most terminal tools show diagnostics anyway), while
`gui-base` keeps doing both independently (its own drawn underline was
never subject to this limitation).

Virtual text needed real new plumbing, not just a rendering tweak: added
`viewmanager.DiagnosticSpan{TextRange, Style, Message}` (a new file,
`diagnostic.go`) — deliberately **not** folded into `StyledSpan`, which
stays general-purpose syntax-rendering vocabulary with no room for a
message string. `Editor.diagnostics`/`DiagnosticSpans` changed type
accordingly; `lsp.go`'s diagnostics handler now preserves the real
`lspclient.Diagnostic.Message` instead of discarding it. Both frontends
show a dim, truncated (rune-safe, not byte-truncated), newline-collapsed
one-line summary after the diagnosed line's own content — real message
text from the server, not a placeholder.

Deliberately deferred, per the user's own instinct that it's bigger scope:
code actions / quick-fixes (a picker UI + `workspace/applyEdit`) — agreed
this is real, separate, "Part 4"-shaped work, not a quick add on top of
virtual text.

**Real performance bug found and fixed (2026-09-21)**, reported by the
user as stutter scrolling down through a large file with `j`: confirmed,
measured, and fixed — not a guess. `internal/syntax.Highlighter.Spans`
was doing a linear scan over *every* highlight range in the whole
document on every call, not just the visible window; a synthetic
2000-function Go file produced over 16,000 ranges, and `Spans` is called
once per render frame — including pure cursor movement (`j`/`k` never
touch the highlighter's incremental-update path, since no text changes),
so scrolling paid an O(total document tokens) cost on every keystroke,
contradicting this project's own "per-keystroke work stays bounded to the
visible range" principle. Fixed exactly, not heuristically: confirmed
empirically that `h.ranges` comes back sorted by `StartByte` from
`gotreesitter`, then added a `maxEndSoFar` prefix-max array (kept in sync
via a new `setRanges` — the one place `h.ranges` is ever assigned) so
`Spans` can binary-search a real, correct lower bound even for a long
range (e.g. a block comment) starting well before the query window —
plain binary search on `StartByte` alone would have missed that case.
`Spans` is now O(log n + k) instead of O(n), k being the tokens actually
in view. Measured, not assumed: 2.7x faster at 500 functions, 4.1x at
2000, **29x at 8000** — the gap widens with file size exactly as the
complexity analysis predicts, confirmed via a real before/after benchmark
(not kept in the shipped test suite — the fix itself is, along with a
correctness test for the long-range-before-the-window case and a
`BenchmarkSpansOnLargeDocument` giving a real absolute number for the
current implementation). `viewmanager.DiagnosticSpansForRange` has the
same *class* of linear-scan issue, deliberately not fixed here — real
diagnostic counts are orders of magnitude smaller than syntax token
counts in practice, so it's a real but much lower-priority version of the
same thing, not overlooked.

Visual mode (Known issues item 1) remains open and unrelated to any of the
above. Go-to-definition and everything it drags in (multi-buffer/
workspace, workspace-wide search) is the deliberately separate, not-yet-
started Part 3.

**Deliberate pause before Part 3, 2026-09-24**: rather than starting
Part 3 on top of `internal/editor` as it stood after the LSP work above,
the user read through the package end-to-end and did a real structural
review — not because anything is broken (it all works, and is verified),
but because `editor.go` in particular had grown organically to 738 lines
across a lot of fast, feature-focused sessions. The concrete result is a
real refactor list in `BACKLOG.md`'s "`internal/editor` structural
refactor" section — bundling scattered LSP fields into one struct, moving
a few standalone algorithms (auto-pairing, multi-edit apply) out of
`Editor` into places that don't need its state to reason about them, and
giving command-mode (`:...`) parsing its own package the same way
Normal-mode already got `internal/keyengine`. None of it is done yet —
Part 3 should either wait for it or at least not make it harder.

**Structural refactor in progress, 2026-09-25**: four of the review's items
are done — `internal/langdetect` (extension→`types.Language` detection,
out of `internal/types`), `internal/autopairs` (bracket/quote pairing
decision logic, out of `Editor`), `internal/exmode` (command-mode parsing,
mirroring `internal/keyengine`), and `internal/lspservice` (the LSP session
type, first given its own methods so `Editor` stopped touching its fields
directly, then moved to its own package once that made it dependency-free
of `Editor`'s own state — see `BACKLOG.md` for the exact method list).
Each followed the same discipline: real current code read first, extract,
reconnect every call site, then `go build`/`vet`/`gofmt`/`test` clean
before moving to the next one — and for `lspservice` specifically, also a
real run against a live `gopls` (`LSPMANAGER_INTEGRATION=1`), since the
staleness-check call sites it touched are exactly the kind of concurrency
bug the fast unit suite wouldn't catch. Remaining items are tracked in
`BACKLOG.md`, not repeated here.

## Module & toolchain

- Module: `github.com/fliplucky/pieces-store`
- Go version: `1.26.2`
- Key dependencies:
  - `gioui.org` — immediate-mode GUI toolkit, used by `internal/gui-base`
  - `github.com/rivo/tview` (+ `github.com/gdamore/tcell/v2`) — terminal UI toolkit, used by `internal/tui-base`
  - `github.com/odvcencio/gotreesitter` — pure-Go tree-sitter runtime (no
    cgo), used by `internal/syntax` for real syntax highlighting. Only the
    curated language subset is compiled in via build tags — see
    Taskfile.yaml's `GRAMMAR_TAGS` var; building without those tags
    (e.g. a bare `go build ./...`, as `go vet`/tests already do) links in
    the library's full ~200-language fleet instead, which still works but
    produces a much larger binary.
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
  character") — though Insert mode's literal capture still does two real
  things of its own, not just plain insertion: auto-closing brackets and
  quotes (`insertRuneWithAutoPairing`/`bracketPairs`/`quoteRunes` — typing
  `{`/`(`/`[` inserts the matching close too and lands the cursor between
  them, typing the close yourself skips over an already-auto-inserted one
  instead of doubling up, and `<BS>` collapses an empty pair in one edit,
  added 2026-09-19; `"`/`'`/`` ` `` get the same treatment via
  `shouldPairQuoteHere`, a heuristic that only pairs when neither
  neighboring character is a word character, so typing a contraction like
  `don't` doesn't trigger an unwanted pair on the apostrophe) and
  tree-based auto-indent on `<Enter>` (see `internal/syntax` below).
  Normal (and eventually Visual) mode hands the
  key to `keyengine.CommandContext.ProcessCommand`, and once a sequence
  resolves, `execute()` (also in `dispatch.go`) resolves the command's
  offset/range via `internal/offset` and applies the verb via
  `piecetable`/`Editor`. Also handles file I/O, vim-style command mode
  (`:w`, `:e`, `:q`/`:q!`, `:wq`, and — added 2026-09-20 —
  `:LspInstall`/`:LspUninstall`/`:LspStatus`, all via `ExecuteCommand`), and
  `Undo`/`Redo`. This is "the point of contact for the gui or tui" per its
  package doc comment.

  `async.go` (added 2026-09-20, alongside the LSP-installer commands above)
  is the concrete answer to the long-open "Concurrency/async orchestration"
  item: an `asyncResults chan asyncApply` drained by one dedicated goroutine
  (started once, from both constructors) that takes `e.mu.Lock()` around
  each queued closure before invoking it — every async producer
  (`:LspInstall` today; diagnostics/completion/hover once Part 2 of the LSP
  plan lands) funnels through `Editor.runAsync(work func() asyncApply)`
  instead of separately reasoning about locking `Editor` correctly. Paired
  with a new `StatusMessage()` accessor — deliberately not a reuse of
  `GetLastCommandError()`, which is explicitly error-shaped (skips
  `ErrQuit`) — for non-error progress text like "installing gopls..."; both
  frontends show it in the status bar right next to the existing error
  segment. Verified end-to-end, not just unit-tested: a real
  `:LspInstall go` through a live `Editor` genuinely installs `gopls` and
  `StatusMessage()` reports the real result once the background goroutine
  finishes (`internal/editor/lspcommands_test.go`'s
  `TestLspInstallGoRealEndToEndThroughEditor`, gated behind
  `LSPMANAGER_INTEGRATION=1`).
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
  reusing the same noun identities `diw`/`daw` resolve), `h`/`j`/`k`/`l`
  and their arrow-key equivalents (plain cursor motion, added 2026-09-19
  — see Known issues history), `u`/`<C-r>`/`r` (editor-level commands),
  `:`/`i`/`o`/`O` (mode switches / line-opening).
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
  Exposed via `Editor.StyleSpans(start, end)`. **Wired into both
  frontends' actual rendering as of 2026-09-19**: `gui-base`'s
  `renderLine` and `tui-base`'s per-line loop both fetch spans and color
  runes per `StyleAt`, falling back to the cheap plain-text path when a
  line has none (identical output to before `StyledSpan` existed,
  verified by tests). `Editor.StyleSpans` now returns real spans for every
  curated `types.Language` — see `internal/syntax` below, the first real
  producer.
- **`internal/syntax`** — real syntax highlighting, added 2026-09-19.
  Wraps `github.com/odvcencio/gotreesitter` (a pure-Go tree-sitter
  runtime — no cgo anywhere, works on every platform including WASM;
  chosen deliberately over the canonical C library specifically to avoid
  this project's cgo/cross-compile/WASM tension, see Vision & direction
  and the Roadmap's treesitter history). `Highlighter` keeps one
  incrementally-updated parse tree per buffer: `Reparse` for a full parse
  (new buffer, or the table replaced wholesale), `Update(edit, source)`
  for an incremental one (fed the same `piecetable.Edit` shape
  `Editor.ChangeChan` already carries), `Spans(start, end)` translating
  the tree into `viewmanager.StyledSpan`s via a small capture-name→`Style`
  prefix mapping, and `IndentAt(offset)` (added 2026-09-19, same day as a
  live bug report: `o` opened lines with no indent at all) — real
  tree-based auto-indent, deliberately not brace-counting the raw text
  (which misfires on a `{`/`}` sitting inside a string or comment, exactly
  the class of bug a real parse tree avoids). Counts nested
  `indentContainerTypes` ancestors at a position — a per-language table of
  which node type means "one more indent level" (Go/CSS/SCSS/Dart:
  `block`; C/C++/PHP: `compound_statement`; JS/TS/TSX: `statement_block`;
  JSON: `object`/`array`; YAML: `block_mapping`/`block_sequence`; HTML:
  `element` — each verified empirically against a real parse, not
  guessed; Dockerfile/Markdown have no entry, correctly returning 0/flush
  left, since neither nests via a brace-like container). Wired into
  `Editor.IndentStringAt` (one tab per level), consumed by `o`/`O`
  (`dispatch.go`'s `openLine`) and `<Enter>` in Insert mode — the two
  needed different insert orderings (`indent+"\n"` for `openLine`, since
  its insertion point already sits at the *next* row's boundary;
  `"\n"+indent` for `<Enter>`, which splits a line's existing content
  instead) — see `openLine`'s doc comment for the exact reasoning, worth
  reading before touching this again. `grammarName` is the one place
  mapping `types.Language` to the library's grammar registry names —
  adding a language is adding a case there (the grammar itself is almost
  certainly already bundled, see below). Wired into `Editor` at
  `moveCursorToLocked` (every incremental edit) and
  `setupHighlighterLocked` (called on construction and whenever the
  table is replaced wholesale — `OpenFile`, `:e`). Bundles only the
  curated language set via Go build tags (`grammar_subset_<lang>` — see
  Taskfile.yaml's `GRAMMAR_TAGS` var and the two `.air.*.toml` build
  commands), roughly halving binary size versus the library's full
  ~200-language fleet. A 2026-09-19 prototype verified, for every language
  in the curated set (Go, TypeScript, TSX, JavaScript, HTML, CSS, SCSS,
  PHP, Dart, C, C++, YAML, JSON, Dockerfile, Markdown): full-quality parse
  support (no missing external-scanner gaps), correct real parses
  (including TSX's notoriously ambiguous JSX-in-TypeScript grammar), a
  working `GOOS=js GOARCH=wasm` build, and real subset-tag binary-size
  reduction — see `internal/syntax/syntax_test.go`'s
  `TestGrammarNameCoversEveryCuratedLanguage` for the same guarantee kept
  honest in CI.
- **`internal/gui-base`**, **`internal/tui-base`** — thin, single-file
  (`base.go`) front ends, one per toolkit. Both hold an `*editor.Editor` and
  render a Tokyo Night–themed view + status bar, and both route every
  keystroke through `ed.HandleKey` (`gui-base` translates Gio's
  `key.EditEvent`/`key.Event` into the same key-string vocabulary
  `tui-base` gets natively from tcell; multi-rune `key.EditEvent` text —
  paste/IME — goes through `ed.InsertLiteralText` instead). `gui-base`
  renders via `Editor.Viewport` (windowed, scroll-follows-cursor) with a
  block/pipe cursor per mode; `tui-base` now also fetches via
  `Editor.Viewport` (a window around the cursor, `tview.TextView.ScrollTo`
  keeping it a fixed distance from the top — settled 2026-09-19, see
  Known issues item 2) instead of full-buffer `GetText()` +
  `strings.Split` every frame. Both
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
- **`platform/`** — no longer an empty placeholder as of 2026-09-20: OS/
  arch-specific concerns nothing else owned. `paths.go`'s `CacheDir`/
  `ConfigDir` (both `os.UserCacheDir`/`os.UserConfigDir`-backed) are the
  first config/cache-directory convention this codebase has ever had — used
  today by `internal/lspmanager` for installed language servers, reusable
  later for settings/plugin state. `arch.go`'s `MasonTargetCandidates`/
  `CurrentTargets` map Go's `GOOS`/`GOARCH` onto mason-registry's own target
  vocabulary (`linux_x64_gnu`, `darwin_arm64`, etc.), most-specific-first,
  with a best-effort musl-vs-glibc detection on Linux.
- **`internal/lspmanager`** — a real, working language-server installer
  (added 2026-09-20, Part 1 of the LSP integration plan), consuming
  `mason-registry`'s live published data
  (`github.com/mason-org/mason-registry`) rather than a hand-rolled
  manifest — same data Neovim's Mason plugin uses, confirmed against a real
  fetch rather than assumed (its release publishes a `registry.json.zip`
  asset; `registry.go` downloads and parses it, filtered to just the
  curated language list's ~12 package names, cached to disk so `:LspInstall`
  doesn't need network on every call). `catalog.go` is the curated
  `types.Language` → mason-registry package table, every entry read
  directly out of a real registry dump: Go→`gopls` (`go install`),
  TypeScript/JS/TSX→`typescript-language-server`, HTML/CSS/SCSS/JSON→the
  `vscode-langservers-extracted` trio, PHP→`intelephense`, YAML→
  `yaml-language-server` (all five npm-installed), C/C++→`clangd`,
  Markdown→`marksman`, Dockerfile→`docker-language-server` (all three
  direct GitHub-release binaries, no runtime dependency), Dart→detected on
  `PATH` only (ships with its own SDK, nothing to install). `install.go`
  checks a package's declared runtime dependency (Node, a Go toolchain, or
  none) via `exec.LookPath` and fails with an actionable message if
  missing — checked lazily, only for the specific server being installed,
  never as a blanket requirement, per an explicit product decision. Real
  archive extraction (zip/tar.gz, with zip-slip protection) and a small
  mason-registry-template resolver (`{{version}}`,
  `{{ version | strip_prefix "v" }}`, and the `{{source.asset.*}}`
  self-reference forms) round out the GitHub-binary path. `index.go`
  persists what's actually installed (`~/.cache/pieces-store/lsp-servers/
  index.json` on Linux) so `:LspStatus` reflects real, restart-durable
  state rather than in-memory-only bookkeeping. **Zero new third-party
  dependencies** — everything above is stdlib (`net/http`, `archive/zip`,
  `archive/tar`, `os/exec`). Verified end-to-end against the live registry
  and real toolchains, not just unit-tested: real `gopls`, a real npm-
  installed `yaml-language-server`, and a real downloaded-and-extracted
  `marksman` binary were each installed for real and made to run
  (`internal/lspmanager`'s `*_integration_test.go` files, gated behind
  `LSPMANAGER_INTEGRATION=1` so normal `go test ./...` stays network-free).
  One real bug this caught before it shipped: `marksman`'s actual
  per-target asset objects carry no `bin` field at all (only
  `docker-language-server`'s do, as the literal template
  `"{{source.asset.file}}"`) — an empty `asset.Bin` needs the identical
  "the downloaded file itself is the binary" fallback, not just the literal
  template string.

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
offset, see the comment there), `gui-base` bypassing the key engine
with its own inline key-handling switch (rebuilt 2026-09-19 to route
through `Editor.HandleKey`/`InsertLiteralText` and render via
`Editor.Viewport` — see Current phase), and — found live by the user
after everything above shipped — `h`/`j`/`k`/`l` and the arrow keys had no
table entries at all in the rebuilt `internal/keyengine` (the original
14-case list never included them, so the only way to move the cursor at
all in Normal mode was repeated `w`). Fixed 2026-09-19: `h`/`l` and their
arrow-key equivalents resolve through the same generic `Move`-verb
pipeline as `w`/`b`/`e` (new `CharBackward`/`CharForward` nouns); `j`/`k`
are special-cased in `executeMove` to call `moveCursorUpLocked`/
`moveCursorDownLocked` directly, since preserving the cursor's column
across differently-sized lines needs current Row/Col state, not just a
byte offset the way every other motion works. Arrow keys also now move
the cursor in Insert mode (handled directly in `handleInsertModeKey`,
bypassing the key engine entirely, same as `<BS>`/`<Enter>` already did) —
previously not wired into either frontend's key translation at all, in
any mode.

Known gaps as of now:

1. **Visual mode is unreachable** — `internal/keyengine/visual_mode.go` is
   still the original copy-paste placeholder (not fleshed out — see the
   comment there for why), and nothing enters Visual mode (no `v` key is
   registered anywhere). Not in scope until Visual mode is actually needed.
2. **`internal/viewmanager/virtual_grid.go`'s `GetScreenPosition` is a
   no-op** passthrough — scroll offsets, line wrapping, tab expansion, and
   a line-number gutter aren't implemented yet (distinct from
   `ViewportSlice`, which both front ends now use for windowed reads —
   `tui-base` got the same `Editor.Viewport` treatment as `gui-base` as of
   2026-09-19, fetching a window around the cursor — see Current phase —
   instead of full-buffer `GetText()` + `strings.Split` on every
   keystroke). `tui-base`'s cursor-follow scrolling is deliberately
   simpler than `gui-base`'s minimal-scroll `followCursor` (a fixed
   top-padding via `tview.TextView.ScrollTo`, not exact on-screen-row
   tracking — that widget doesn't expose visible-row count cheaply before
   its first real draw) — still a real improvement over no auto-follow at
   all, which is what existed before.
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
