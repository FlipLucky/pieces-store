# Backlog

A sidebar priority list — known, deliberately-deferred issues that don't
block the current phase. Pick these up opportunistically (when touching the
same context) or reactively (when the issue actually manifests as a real
problem), not proactively. See `CLAUDE.md`'s Roadmap section for the phase
plan this list sits alongside.

This is not the same as a blocker: if something here is actually stopping
the current phase's work, it belongs in `CLAUDE.md`'s Known Issues, not
here.

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
- `internal/gui-base`/`internal/tui-base` still do naive full-buffer
  `GetText()` + `strings.Split` on every render — the actual fix is the
  planned viewport-as-offset-slice work, not a local patch here.
