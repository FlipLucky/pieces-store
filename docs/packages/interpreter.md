# internal/interpreter

**Processor** for both modes (see
[Validator → Processor → Executor](../concepts/validator-processor-executor.md)).
Sits *above* both [keyengine](keyengine.md) and [offset](offset.md),
importing both without either depending back — preserving the
zero-dependency-sibling property those two packages maintain between each
other.

## Layout

- `normal.go` — `ProcessNormal(cmd, doc, at Position, clip PasteClipboard)
  []Action`. Resolution only: never mutates anything, never touches
  `*editor.Editor`. `resolveMotion`/`resolveRange`/`resolveLineStep` are
  pure functions of `offset.Document`, relocated near-verbatim from the
  old `dispatch.go`.
- `insert.go` — `ProcessInsert(key, doc, at, ctx RuneContext,
  completionActive bool) []Action`. Folds Validator+Processor into one
  function for Insert mode, since Insert mode's "grammar" really is close
  to a no-op — confirmed, not just predicted, once built. Reuses
  `MoveAction`/`DeleteAction` from `normal.go` directly for cursor motion
  and backspace-with-pair-collapse — see
  [Decoupled by shape, not by sender](../concepts/decoupled-by-shape.md).

## The snapshot pattern

`ProcessNormal`/`ProcessInsert` never read live `Editor` state — they take
value snapshots instead: `Position{ByteOffset, Row, Col}`,
`RuneContext{Before, After rune}`, `PasteClipboard{Value, Linewise, Ok}`.
`Editor` reads these once under its lock and passes them in by value. This
is what makes resolution safe to reason about as a pure function, and
what lets `Editor` hold one lock across resolve-then-apply instead of two
separately-locked steps.

## `[]Action`, not `Action`

Both `ProcessNormal` and `ProcessInsert` return `[]Action` (widened from a
single `Action`) — needed for real once a verb/key resolves into more than
one sequential step: `Change` is "delete this range" then "enter Insert
mode"; linewise `Paste` is "move to the line boundary" then "insert here."
A small `one(a Action) []Action` helper wraps the common single-action
case. Applied by `executor.Dispatch` in order, under the same lock.

## Depends on

[keyengine](keyengine.md), [offset](offset.md),
[clipboard-register](clipboard-register.md) (for `RegisterType`),
`internal/autopairs`, `internal/types`.

## Depended on by

[internal/editor](editor.md) (via the `toExecutorActions` bridge —
see [executor](executor.md)).

## See also

- [Validator → Processor → Executor](../concepts/validator-processor-executor.md)
- [Action + Offset](../concepts/action-offset.md)
