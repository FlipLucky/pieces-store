# internal/editor

**The orchestrator.** `Editor` wraps a `*piecetable.Table`, a
`*viewmanager.VirtualGrid`, a local `*Cursor` (position/mode/command-buffer
state — lives here, not in `viewmanager`, since it's editor state, not view
state), and a `*keyengine.CommandContext`. This is "the point of contact
for the gui or tui" per its own package doc comment — see
[Dumb Front Ends / Editor as Spine](../concepts/dumb-front-ends.md).

## Entry point: `HandleKey`

`HandleKey` (`dispatch.go`) is the single entry point for all keyboard
input. It routes on mode:

- **Insert/Command mode** capture keys close to literally — though Insert
  mode's capture still does two real things of its own: auto-closing
  brackets/quotes (`internal/autopairs`, a pure decision function) and
  tree-based auto-indent on `<Enter>` (`internal/syntax`).
- **Normal (and eventually Visual) mode** hands the key to
  `keyengine.CommandContext.ProcessCommand`; once a sequence resolves,
  `interpreter.ProcessNormal` resolves it into one or more `Action`s,
  which `executor.Dispatch` applies — see
  [Validator → Processor → Executor](../concepts/validator-processor-executor.md).

Resolution and application happen inside **one held lock** per keystroke
(`e.mu`) — not two separately-locked steps — so async work (LSP results,
etc.) can never mutate state in the gap between "resolve this motion" and
"apply it." `dispatch.go`'s `lockedEditorView` is the adapter satisfying
`executor.Editor`'s interface by calling `*Editor`'s own `*Locked` method
bodies directly, assuming the lock is already held — see the
`*Locked`-method-splitting convention below.

## The `*Locked` method-splitting convention

A public method locks, then calls a private `*Locked` body that assumes
the lock is already held (e.g. `Undo`/`undoLocked`, `SetMode`/
`setModeLocked`, `RequestHover`/`requestHoverLocked`). Exists specifically
to avoid a reentrant-mutex deadlock: a public, self-locking method called
from somewhere that already holds `e.mu` deadlocks on first use. Caught
twice for real in this project's history (`:Format`/`StartFormat`, and
again while building the Validator→Processor→Executor pipeline) — treat it
as a load-bearing pattern, not incidental style, before adding a new
locked method.

**The split itself is load-bearing; the `Locked` suffix in the name is
not** (revisited 2026-10-11): the precondition — "caller must already hold
the lock" — is a calling convention, not behavior, and doesn't belong
baked into the name. These existing names are grandfathered, not due for a
mass rename; new lock-free helpers (see `piecetable`'s planned `pieceAt`,
`ATOMIC_TASKS.md`'s T16) use a plain, behavior-named identifier instead —
and when every call site already satisfies the precondition structurally
(not a per-function concern at all, just an architectural fact about the
package), the explanation lives once in the *package's* own doc comment,
not repeated on each helper. No external package should ever need to know
or care that locking happens at all. See CLAUDE.md's Documentation
conventions section.

## Structural refactor, T4–T9 (2026-10-17)

The mechanical remainder of the 2026-09-24 review (`langdetect`/
`autopairs`/`exmode`/`lspservice` were done earlier) — each item pulled
logic out of `Editor` into the type that should own it, per
[the orchestrator doesn't play an instrument](../concepts/orchestrator-instrument.md):

- `Cursor.SetMode` resets its own `CommandBuffer`/`Completion`/
  `LSPCompletion` fields on a mode transition, instead of `Editor`
  reaching into `Cursor`'s fields from outside. `Editor`-level cleanup a
  transition also needs (`completionLineHead`, dismissing hover,
  notifying frontends) stays in `setModeLocked`.
- `CommandBuffer` is its own type (`cursor.go`) — `Append`/`Backspace`/
  `Clear`/`Set`/`String`, a named `string` under the hood so `%s`
  formatting and string-literal comparisons keep working unchanged.
- `CompletionState`/`LSPCompletionState` each have a `Cycle(direction)`
  method for the wraparound math, kept separate since the two hold
  different candidate slices.
- `internal/syntax.Highlighter.IndentStringAt` does the tab-formatting;
  `Editor.indentStringAtLocked` is a pure passthrough.
- File-related methods (`NewEditorFromFile`, `OpenFile`, `SaveFile`,
  `SaveFileAs`, `GetFilePath`, `Language()`) live in `file.go`.
- `MoveCursorLeft/Right/Up/Down` (+ `*Locked` bodies) live in `cursor.go`
  — still `*Editor` methods (they need `e.table`/`e.mu`), a pure file
  move, not a receiver-type change.

## Other responsibilities

- File I/O, vim-style command mode (`:w`, `:e`, `:q`/`:q!`, `:wq`,
  `:LspInstall`/`:LspUninstall`/`:LspStatus`, `:register`/`:registers`),
  `Undo`/`Redo`.
- `registers.go` (T25, 2026-10-11) — `:register`'s overview. Not a real
  `Mode` (same reasoning as `HoverState`'s own doc comment): the very next
  keystroke dismisses it, consumed by a dedicated branch at the top of
  `handleNormalModeKey` rather than fed to `keyengine`. Builds the display
  per the user's own proposed layout (unnamed first, then up to 3 of each
  of yank/delete/visual, then named registers alphabetically) and
  centralizes line formatting here rather than duplicating it per
  frontend — see [clipboard-register](clipboard-register.md).
- `async.go` — the answer to async orchestration: an `asyncResults`
  channel drained by one dedicated goroutine that takes `e.mu.Lock()`
  around each queued closure. Every async producer (LSP install,
  diagnostics, completion, hover) funnels through
  `Editor.runAsync(work func() asyncApply)` instead of separately
  reasoning about locking correctly.
- `lsp.go` — one lazily-started LSP server per buffer language, wired at
  `moveCursorToLocked` (incremental `didChange`, synchronous, same
  reasoning as the tree-sitter highlighter's own inline `Update` call —
  see [internal/syntax](syntax.md)).

## Depends on

[keyengine](keyengine.md), [interpreter](interpreter.md),
[executor](executor.md), [offset](offset.md), [piecetable](piecetable.md),
[viewmanager](viewmanager.md), [syntax](syntax.md),
[clipboard-register](clipboard-register.md), the [LSP stack](lsp-stack.md).

## Depended on by

[Front ends](frontends.md) (`cmd/gui`/`cmd/tui` via `gui-base`/`tui-base`)
— the only consumers; nothing else in the module imports `editor`.

## See also

- [Validator → Processor → Executor](../concepts/validator-processor-executor.md)
- [The orchestrator doesn't play an instrument](../concepts/orchestrator-instrument.md)
  — a real, current, *not yet fixed* example (`moveCursorToLocked`) lives
  right here.
