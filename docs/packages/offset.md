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

  T55 (2026-10-18) added `MotionLineStart`/`MotionFirstNonBlank`/
  `MotionLineEnd`/`MotionLineContentEnd`/`MotionParagraphForward`/`Backward`
  (`0`/`^`/`$`/`A`'s own target/`}`/`{`) plus `RangeJoinLines`/
  `RangeToggleCase` (`J`/`~`) — all thin, composable wrappers over the
  already-private `lineStart`/`lineEnd`/`prevLineSpan`/`nextLineSpan`/
  `isBlankSpan` this file's `RangeLine`/`RangeInnerParagraph` already used
  internally, not new scanning logic. `MotionLineEnd` (lands ON the last
  real character, Normal mode's own `$`) and `MotionLineContentEnd` (one
  past it, `A`'s append target) are genuinely two different primitives, not
  a duplicate pair — Insert mode's cursor can sit past the last character
  the way Normal mode's can't.

  **A real, pre-existing `lineStart` bug, found by this task and fixed at
  the root**: calling it directly with `offset == 0` on a document whose
  first character is itself a newline (an empty first line) wrongly
  concluded the cursor was already *past* that empty line, into line 1.
  Root cause: `FindOffset`'s match predicate only ever tests positions it
  steps *onto*, never its own starting position — so a direct call at
  offset 0 never gets to test offset 0's own content via the normal search
  path at all, only via `lineStart`'s own compensating `GetRuneAt(0)`
  check, whose existing interpretation (there's a newline right at the
  document start, so whatever line I started searching *from* begins right
  after it) is correct for every `offset > 0` call but wrong for `offset ==
  0` itself, where there is no "line I started searching from" other than
  the one actually sitting at 0. Fixed with a two-line early return
  (`offset <= 0` always starts its own line) — confirmed against real vim
  semantics first (an empty line's cursor position is that line, not the
  next one), the same discipline every prior correctness fix in this
  package follows. Never caught before because nothing had ever called
  `RangeLine`/`lineStart`-dependent logic with the cursor sitting exactly
  on a document's own leading blank line.
- `bracket.go` — `MatchBracket` (T44, 2026-10-18), vim's `%`: depth-counts
  same-type brackets from a starting position to find its match. Takes an
  optional `skip func(offset int) bool` so a caller can exclude offsets
  that shouldn't count as real brackets (one sitting inside a string or
  comment) — `offset` itself stays ignorant of what a "string" even is,
  the same interface-not-knowledge decoupling the rest of this package
  uses. See [interpreter](interpreter.md)'s `BracketContext` and
  [editor](editor.md)'s `bracketMatchSkipLocked` for who actually builds
  that predicate.

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
