# Backlog

A sidebar priority list — known, deliberately-deferred issues that don't
block the current phase. Pick these up opportunistically (when touching the
same context) or reactively (when the issue actually manifests as a real
problem), not proactively. See `CLAUDE.md`'s Roadmap section for the phase
plan this list sits alongside.

This is not the same as a blocker: if something here is actually stopping
the current phase's work, it belongs in `CLAUDE.md`'s Known Issues, not
here.

## `internal/editor` structural refactor (pre-Part-3 architecture review, 2026-09-24)

The user paused before starting Part 3 (go-to-definition/multi-buffer) to
read through `internal/editor` end-to-end and review its structure — not
because anything here is broken, everything listed is fully functional.
The pattern behind most of these: fast, feature-focused implementation
work correctly avoided restructuring unrelated code mid-feature (a real,
legitimate discipline), but that means reorganization never became anyone's
actual task until a dedicated pass like this one made it one. Grouped by
theme, not urgency — none of these block anything:

**State that should be bundled, not scattered:**
- ~~**Bundle the LSP-related `Editor` fields into one dedicated struct**~~
  **Done 2026-09-25**, completed in two steps. First: `LspService`
  bundled `LspServer`/`LspLanguage`/`LspVersion`/`LspURI`/`LspCapabilities`/
  `LspCompletionGeneration` into one `Editor` field (`e.lspService`),
  reconnected across `lsp.go`/`lspfeatures.go`/`dispatch.go` —
  `stopLSPServerLocked`'s reset became one assignment instead of five.
  Second, same day: gave it real methods (`Active`, `IsCurrent`, `Reset`,
  `Start`, `Reopen`, `DidOpen`, `DidChange`, `NextCompletionGeneration`) so
  `Editor` stopped reaching into its fields to decide anything itself —
  then, since it turned out to have zero dependency on `Editor`'s own
  state (only `lspclient`/`types`), moved it to its own package,
  `internal/lspservice` (type renamed `Service` to avoid the
  `lspservice.LspService` stutter, fields dropped their `Lsp` prefix
  accordingly — `Server`/`Language`/`Version`/`URI`/`Capabilities`/
  `CompletionGeneration`). Verified against a real, live `gopls` (not just
  the fast unit suite) since this touched the async staleness-check call
  sites directly — `TestRealDiagnosticsFlowIntoStyleSpans` and
  `TestRealHoverAutocompleteAndFormatEndToEnd` both still pass end-to-end.
  `diagnostics` deliberately stayed a separate `Editor` field rather than
  joining the struct — it's keyed to `URI` but has its own independent
  lifecycle (arrives async, well after the session starts).
- **`SetMode`'s mode-transition cleanup belongs on `Cursor`, not `Editor`**
  — `Editor.SetMode` reaches into `Cursor`'s own fields (`CommandBuffer`,
  `Completion`, `LSPCompletion`) from the outside to decide what to clear
  on a mode change. `Cursor.SetMode` already exists but is a trivial
  one-line setter today; give it the real transition logic instead, and
  `Editor.SetMode` shrinks to lock + delegate + notify.
- **`CommandBuffer` deserves its own type**, not a raw `string` field on
  `Cursor` poked at by `Editor` methods (`AppendCommandBuffer`,
  `BackspaceCommandBuffer`, `ClearCommandBuffer`). `BackspaceCommandBuffer`
  in particular already does real logic (UTF-8-safe rune slicing to avoid
  splitting a multi-byte character) — that belongs encapsulated on the
  type itself (`Append`/`Backspace`/`Clear`/`String` methods), not as
  `Editor` methods reaching into a bare string.
- **`CompletionState`/`LSPCompletionState` should own their own cycling** —
  `Editor` currently does the wraparound index math
  (`((index+direction)%n+n)%n`) externally for both. Give each state type
  its own `Cycle(direction)` method instead.

**Logic that leaked into the wrong layer:**
- ~~**`types.DetectLanguage` shouldn't live in `internal/types`**~~ **Done
  2026-09-25** — moved to `internal/langdetect.Detect`, zero-dependency
  leaf package, same shape `types` already has. `types` is back to pure
  enums.
- ~~**Bracket/quote auto-pairing's decision logic is a real, standalone
  algorithm currently living inline in `Editor`**~~ **Done 2026-09-25** —
  extracted to `internal/autopairs` as a pure function (`Decide(typed,
  before, after rune) (Action, rune)`) taking just the three runes
  involved, no buffer or `Editor` needed. `Editor` kept as the thin
  imperative shell that reads the runes around the cursor and acts on the
  decision. The multi-edit apply-in-reverse-order algorithm
  (`applyTextEditsLocked`, `internal/editor/lspfeatures.go`) is still
  open — same reasoning applies, just not done yet.
- **`IndentStringAt`'s tab-formatting belongs in `internal/syntax`, not
  `Editor`** — `Editor.indentStringAtLocked` asks the highlighter for a
  depth (fine, that's orchestration) but then also decides *how* to render
  it (`strings.Repeat("\t", depth)`, a formatting policy). Give
  `Highlighter` its own `IndentStringAt(at) string` returning the already-
  formatted whitespace, so `Editor`'s version becomes a pure passthrough.

**File organization** (no logic changes, just moving code to where its name says it should be):
- Pull file-related `Editor` methods (`NewEditorFromFile`, `OpenFile`,
  `SaveFile`, `SaveFileAs`, `GetFilePath`, `Language()`) out of `editor.go`
  into their own `internal/editor/file.go` — mirrors the split
  `piecetable` already has (table/reader/writer/file).
- Move `MoveCursorLeft`/`Right`/`Up`/`Down` (and their `*Locked` bodies) out
  of `editor.go` into `cursor.go` — `cursor.go` today only holds the
  `Cursor` *type*, no cursor behavior; the file's name should match its
  actual job.
- Remove the `SIMULATE=true` typing-demo goroutine in `NewEditor`
  (`internal/editor/editor.go`) — a leftover from the project's earliest
  prototype phase, no longer needed. Note for whoever does this: it's not
  the *only* reason `Editor`'s state needs its mutex on every access,
  including plain getters — the async/LSP work introduced real concurrent
  access independent of this demo code, so removing it doesn't relax any
  locking.

**A real missing package, mirroring one that already exists:**
- ~~**Command-mode (`:...`) parsing deserves its own package, the same way
  Normal-mode keystrokes got `internal/keyengine`**~~ **Done 2026-09-25** —
  `internal/exmode.Parse(raw string) Command{Name, Args}` now owns parsing
  only, the same "resolve, never act" discipline `keyengine` already
  follows. `executeCommandLocked` (`internal/editor/editor.go`) still owns
  all 9 cases' actual dispatch logic — that part correctly stays in
  `Editor`, since every case needs its own state (`e.table`, `e.mu`, the
  LSP methods) directly.

**Resolved since the last pass (2026-09-13), removed from this list**: the
undo/redo model decision (settled — symmetric redo stack on
`piecetable.Table`, see `writer.go`'s `Undo`/`Redo`); `:q`'s missing dirty
guard (added — `Table.Dirty`, `:q!` force-quits); `internal/editor/keymap`
(deleted entirely, replaced by `internal/keyengine`); `internal/editor/
interfaces.go`'s orphaned `Document` (deleted); the `Around` modifier and
`LineMotion` noun missing table entries (both real now, in
`internal/keyengine/normal_mode.go`). `RuneMotion` stays deliberately
unregistered — it's for a future `x` keybinding, not in the agreed 14-case
list, not a gap.

## Deferred performance (viewport-scoping changes the calculus here)

- **`piecetable.FindPieceAt` is an O(P) linear scan** (`internal/piecetable/reader.go`)
  with no index/tree over pieces. Deferred: once viewport slicing exists,
  this scales with piece count from editing activity, not file size — for a
  single-file MVP session, piece count should stay small enough that this
  doesn't matter in practice. Revisit if a long, heavily-edited session
  makes it measurably slow.
- **`Table.History` (and now `RedoStack`) grow unbounded** — every
  `Insert`/`Delete` pushes a full snapshot forever, no cap or eviction.
  Revisit once real session lengths are known; a simple depth cap or
  coalescing strategy would fix it.
- **`internal/viewmanager/virtual_grid.go`'s `GetScreenPosition` is a no-op**
  — no scroll offsets, line wrapping, tab expansion, or gutter width yet.
  Needed for real files with tabs/long lines; not needed for a minimal
  plain-text MVP.

## Correctness/robustness (latent — not currently triggered, but real)

- **`piecetable.GetRuneAt`/`FindPieceAt` bypass their own mutex** on the
  actual read path — a genuine data race if ever called concurrently from
  outside `Editor`'s coarser lock. Not triggered today (single-goroutine
  usage per `Editor`), but a broken promise on a type that advertises
  thread safety via `sync.RWMutex`.
- **`GetRange` panics on out-of-range/reversed input**; `Insert` with a
  negative offset silently misplaces instead of erroring. No consistent
  failure discipline across `Insert`/`Delete`/`GetRange`/`FindPieceAt`. Will
  matter more once viewport work starts calling `GetRange` for real.
- **No rune-boundary guard on `Insert`/`Delete`/`GetRange`** — a
  non-rune-aligned byte offset would silently corrupt UTF-8 text. Not
  currently reachable (all current callers stay rune-aligned via
  `internal/offset`), but no defense-in-depth exists.

## API/interface cleanup (cheap, no urgency)

- **`internal/offset/motions.go`'s `ScanUntil` is dead code** — nothing
  calls it, has the same byte-stepping bug `FindOffset` used to have, looks
  like an abandoned attempt to simplify/replace `FindOffset` that never got
  finished. Not yet decided whether to delete it or complete the
  replacement.
- **`internal/keyengine/visual_mode.go`** is still the original copy-paste
  placeholder — real Visual-mode operator semantics differ from Normal's
  (an operator acts on the current selection directly, no following noun
  needed), so this needs real design work, not just filling in the
  existing table shape, whenever Visual mode is actually in scope.
- **Counted delete isn't implemented** — `executeDelete` (`internal/editor/dispatch.go`)
  ignores `Count`, applying once regardless (e.g. `3diw` behaves like
  `diw`). Not in the agreed 14-case list; `executeMove` already loops on
  `Count` if a similar loop is wanted here later.
