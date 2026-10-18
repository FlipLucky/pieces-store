# internal/interpreter

**Processor** for both modes (see
[Validator → Processor → Executor](../concepts/validator-processor-executor.md)).
Sits *above* both [keyengine](keyengine.md) and [offset](offset.md),
importing both without either depending back — preserving the
zero-dependency-sibling property those two packages maintain between each
other.

## Layout

- `normal.go` — `ProcessNormal(cmd, doc, at Position, clip PasteClipboard,
  bracket BracketContext) []Action`. Resolution only: never mutates
  anything, never touches `*editor.Editor`. `resolveMotion`/`resolveRange`/
  `resolveLineStep` are pure functions of `offset.Document`, relocated
  near-verbatim from the old `dispatch.go`.

## Text construct vs. short delete (T22, 2026-10-11)

Two deliberately different ranges `resolveRange` can produce, picked by a
`modifier == None && noun != LineMotion` branch at its top:

- **Text construct** (`diw`/`daw`/`dip`/`dd`/`yy`/`cc`) — a complete
  semantic unit regardless of cursor position inside it, resolved by
  `resolveRangeOnce`'s existing chaining (unchanged).
- **Short delete** (`dw`/`cw`/`yw`, `db`/`de`/`dh`/`dl`, and `x`) — a bare
  motion swept from the cursor to wherever it lands, possibly a whole word
  or half of one — the user's own distinction. New `resolveMotionRange`
  reuses `resolveMotion` (the exact single-step resolver plain `Move`
  already calls) rather than duplicating motion logic. `3dw` sweeps three
  whole word-motions (count applied per hop before cutting once) — a
  genuinely different counting rule from `3diw`'s word/whitespace
  alternation, confirmed against real vim docs.

Deliberately uniform across d/c/y with no vim-style `cw`≈`ce`
special-casing. The one real exception: motion **inclusivity** — `e`
lands ON its target character in real vim (so the swept range must
include it), while `w`/`b`/`h` land just past/before theirs and are
exclusive by construction; `l` already resolves to exactly one rune past
`at` and needs no adjustment. Handled once via `isInclusiveMotion` since
it's a structural property of the motion itself, not a one-off
command-pair quirk — the same reasoning `LineMotion`'s own structural
handling already follows.
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
`RuneContext{Before, After rune}`, `PasteClipboard{Value, Linewise, Ok}`,
`BracketContext{Skip}` (T44, 2026-10-18 — a predicate `%`'s `resolveMove`
passes straight through to `offset.MatchBracket`, letting it exclude a
bracket sitting inside a string/comment without `interpreter` itself ever
touching `Highlighter`/`StyledSpan`). `Editor` reads these once under its
lock and passes them in by value. This is what makes resolution safe to
reason about as a pure function, and what lets `Editor` hold one lock
across resolve-then-apply instead of two separately-locked steps.

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
