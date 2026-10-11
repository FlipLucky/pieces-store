# internal/offset

Text-position/structure math, with **zero dependency** on anything else in
the module — a leaf package, usable by cursor movement, the key engine's
resolvers, or anything else that ever needs "where does this word/
paragraph start and end" (e.g. a future double-click-to-select-word),
independent of vim keybindings specifically.

## Layout

- `position.go` — the `Document` interface (`GetRuneAt`, `Len` — the
  minimal read contract everything here needs; `*piecetable.Table`
  satisfies it), byte↔`Position` (row/col) conversion, UTF-8-safe rune
  stepping (`MoveLeft`/`MoveRight`).
- `motions.go` — `FindOffset` (the shared scanning primitive underneath
  almost everything else here), predicates, and the vim motion/text-object
  resolvers built on it: `MotionWordForward`/`Backward`/`End`,
  `RangeInnerWord`/`AroundWord`/`Paragraph`, `RangeLine`,
  `MotionFindCharForward`/`MotionFindCharBackward` (`f`/`F`, wired into a
  real keybinding 2026-10-11 — see
  [interpreter](interpreter.md)'s `BeginFindCharAction`/`ResolveFindChar`).
  Neither has a line-boundary guard yet (real vim's `f`/`F`/`t`/`T` stop at
  the current line's end; `FindOffset` has no concept of lines at all) —
  a known, documented simplification, not a silent gap.
  `ScanUntil` is dead code — an abandoned attempt to simplify/replace
  `FindOffset` that never got finished; not yet decided whether to delete
  it or complete the replacement.

## Depends on

Nothing in this module.

## Depended on by

[internal/interpreter](interpreter.md) (Processor's resolvers),
[internal/keyengine](keyengine.md) indirectly has no dependency on it (the
two stay siblings — see [Simple, Not Easy](../concepts/simple-not-easy.md)),
[internal/viewmanager](viewmanager.md), [internal/editor](editor.md).

## See also

- [Action + Offset](../concepts/action-offset.md) — this package *is* the
  "resolves an offset" half of that shape.
