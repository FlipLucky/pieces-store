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

## Other responsibilities

- File I/O, vim-style command mode (`:w`, `:e`, `:q`/`:q!`, `:wq`,
  `:LspInstall`/`:LspUninstall`/`:LspStatus`), `Undo`/`Redo`.
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
