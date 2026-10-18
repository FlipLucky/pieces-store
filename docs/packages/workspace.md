# internal/workspace

The outer "current buffer" selector the Roadmap's multi-file/workspace
stage already proposed (2026-09-24), built 2026-10-18 (T56): one
self-contained [internal/editor](editor.md) `*Editor` per open file
(unchanged — `NewEditor`/`NewEditorFromFile` were already safe to call N
times independently, zero package-level state), plus a thin `Workspace`
tracking which one is current. `Editor` itself stays completely unaware
this package exists — the same decoupling boundary as every other package
in this codebase.

No separate `Buffer{Path, Editor}` wrapper type: `Editor.GetFilePath()`
already covers display/identity reliably, so `Workspace` holds
`[]*editor.Editor` directly rather than introduce a redundant second
source of truth for the same information — [minimal primitives, maximal
composition](../../CLAUDE.md#design-philosophy).

## The interception problem

`Editor.HandleKey` already fully executes a Command-mode `<Enter>`
keystroke the instant it fires — `handleCommandModeKey` calls the public
`ExecuteCommand` directly, with no existing seam for an outer layer to act
*before* Editor's own (different) `:e` would run.

`Workspace.HandleKey` solves this with **zero new Editor API**: it peeks
at `Current().GetCommandBuffer()` before forwarding that one keystroke; if
[exmode](small-packages.md)'s `Parse` resolves it to a command Workspace
itself owns, Workspace handles it entirely and resets Editor's pending
Command-mode state via the already-public
`Current().SetMode(types.ModeNormal)` — which `Cursor.SetMode` already
clears `CommandBuffer`/`Completion`/`LSPCompletion` from (see
[editor](editor.md)'s T4 entry). Every other keystroke, Command-mode or
not, forwards unchanged to `Current().HandleKey`.

## Owned commands

| Command | Effect |
|---|---|
| `:e`/`:edit <path>` | Switch to an already-open buffer for `path` (compared by `filepath.Abs`), or open a new one. Deliberately collides with and fully supersedes Editor's own `:e` case rather than adding a second one. `:e` on a nonexistent path still errors — the identical limitation Editor's own pre-existing `:e` already had, not a new one. |
| `:bn`/`:bnext` | Next buffer, wrapping around. "no other buffers" status message with only one buffer open. |
| `:bp`/`:bprev`/`:bprevious` | Previous buffer, same wraparound/single-buffer behavior. |
| `:bd`/`:bdelete` | Close the current buffer. Refuses on `Editor.IsDirty()` and on the last remaining buffer. |
| `:bd!`/`:bdelete!` | Force-close, discarding unsaved changes. |
| `:ls`/`:buffers` | List every open buffer: `N`, a `%`/space current marker, a `+`/space dirty marker, then its display name (`GetFilePath()`, or `[No Name]`). |

`Editor.IsDirty() bool` (`internal/editor/file.go`) is the one genuinely
new piece of `Editor` surface this task needed — a public wrapper around
the previously package-private `e.table.Dirty`, for `:bd`'s safety check.

## The redraw-binding gap

Both frontends' redraw-trigger goroutines used to capture
`ed.ChangeChan()` once at boot and `range` over that one fixed channel
forever. Switching `Workspace.current` to a different `*editor.Editor`
(its own, separate channel) would leave the goroutine listening to a
buffer no longer on screen — a real gap, found and fixed before it ever
shipped, not discovered live.

`Workspace.Switched() <-chan struct{}` is the fix: the same non-blocking-
send coalescing idiom `Editor.ChangeChan` already established (a burst of
switches collapses to one redraw, not a queue), fired at the end of every
intercepted command. Both frontends restructured their redraw loop from a
bare `for range ed.ChangeChan()` into:

```go
for {
    select {
    case <-ws.Current().ChangeChan():
    case <-ws.Switched():
    }
    // redraw
}
```

`ws.Current()` is re-evaluated fresh on every loop pass — it's a call
inside the `select`, which is itself inside the `for` — so the pass right
after a `Switched()` wakeup automatically listens to the new current
buffer's own channel instead of the stale one. No library/toolkit support
needed; found by tracing the actual goroutine shape in both frontends
before writing the fix.

## Scope cuts, stated not glossed over

- No oil.nvim-style directory-as-buffer file navigator yet — the user's
  explicitly-named *next* want for file navigation, not this pass.
- No splits yet — buffers' own stated reason for being built first, but a
  separate task.
- Real vim's "two windows showing the *same* buffer with independent
  cursor/scroll" isn't expressible in this shape at all: `Editor` still
  fuses cursor/viewport state to buffer content, so v1 splits (when built)
  will only support different buffers side by side. A named future scope
  cut, not an oversight.

## Depends on

[internal/editor](editor.md), [internal/exmode](small-packages.md),
`internal/types`.

## Depended on by

[Front ends (gui-base/tui-base)](frontends.md), via
[Entry points](entry-points.md).

## See also

- [Decoupling Rule / Conveyor-Belt Splitter](../concepts/decoupling-splitter.md)
  — `Editor` stays unaware `Workspace` exists, the same shape as every
  other boundary in this codebase.
