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

## Filling out Normal mode's motions (T55, 2026-10-18)

`resolveMotion` (the single-step resolver both bare `Move` and
`resolveMotionRange`'s short-delete path call) gained five new cases —
`LineStart`/`FirstNonBlank`/`LineEnd`/`ParagraphForward`/`Backward` — each
a thin wrapper over a new `offset` primitive (`MotionLineStart`/
`MotionFirstNonBlank`/`MotionLineEnd`/`MotionParagraphForward`/`Backward`).
Registering them here, not just in `resolveMove`'s own special-case switch,
is what makes `d0`/`d^`/`d$`/`d{`/`d}` fall out for free — the exact same
mechanism the T22 short-delete section above already documents for
`dw`/`db`/`dh`/`dl`.

`$` (`LineEnd`) needed one real fix to `resolveMotionRange`'s inclusive-
motion handling: the function's existing `target == at` branch meant "the
motion didn't move at all, nothing to act on" for every noun it had ever
seen — correct for exclusive motions, but wrong for an inclusive one
landing exactly where the cursor already sits (`d$` with the cursor
already on the line's last character) — there, `target == at` still has
one real character to act on. Fixed by checking `isInclusiveMotion` first
in that branch, with a real-character guard (`r != '\n'` and `target <
doc.Len()`) so `d$` on a blank line or at the document's end still
correctly no-ops instead of phantom-deleting the line's own newline.

`gg`/`G` (`BufferStart`/`BufferEnd`) are handled directly in
`ProcessNormal`'s switch and `resolveMove`'s own special-case section
respectively — not through `resolveMotion` — since both need a target *row*
computed from `offset.PositionToByteOffset`/`ByteOffsetToPosition`, not a
pure function of the current byte offset. `G` reads the new
`ExecutableCommand.HasCount` to distinguish "no count typed" (last line)
from "an explicit count, even `1G`" (that line) — see
[keyengine](keyengine.md)'s own T55 entry for why `Count` alone can't tell
those apart. A shared `resolveGotoLine(doc, row) MoveAction` lands both on
the target row's first-non-blank (`offset.MotionFirstNonBlank`), matching
real vim.

`a`/`A`/`I` (`AppendAfterCursor`/`AppendEndOfLine`/`InsertAtFirstNonBlank`)
each resolve to a `MoveAction` + `EnterInsertAction` pair, directly in
`ProcessNormal`'s switch — the same two-action composition
[CLAUDE.md's "Minimal primitives, maximal composition"](../../CLAUDE.md)
entry already established for Visual mode's own entry sequence. `A`
specifically needed a new primitive, `offset.MotionLineContentEnd` (one
past the last real character — the row's own newline, or `doc.Len()` for
the last line with none), genuinely distinct from `$`'s `MotionLineEnd`
(lands ON the last character): Insert mode's cursor can sit one past the
last character the way Normal mode's can't, so the two targets are a real
one-rune apart, not interchangeable.

`~`/`J` (`ToggleCase`/`JoinLines`) each resolve to a `DeleteAction` +
`InsertTextAction` pair — the same "replace this span with computed text"
shape `Change` already uses, just with the replacement text coming from a
new `offset` resolver (`RangeToggleCase`/`RangeJoinLines`) instead of
Insert mode's own typed keystrokes. Both return `ok=false` for their
respective no-op cases (`~` at the document's end, `J` on the last line)
rather than ever constructing an empty edit.

**`cc`/`S`'s trailing-newline bug, found by this task, not caused by it**:
`Change`'s existing `LineMotion` case (shared with `dd`/`yy` via
`resolveRange`) included the line's trailing newline in the resolved
range — correct for `Delete` (the line structurally disappears) and
`Yank` (reproducing it on paste needs it), but wrong for `Change`, which
should leave the newline in place so typing replaces a line's content
without merging into whatever follows. `cc` itself had no end-to-end test
before `S` (a pure alias, see keyengine's entry) exposed it. Fixed with a
new `trimTrailingNewline(doc, rng) offset.TextRange` helper, called only
from the `Change` + `LineMotion` case — trims exactly one trailing
newline regardless of count, confirmed to generalize correctly to `3cc`
via a dedicated test rather than just asserted.

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

## ProcessVisual and ResolveVisualRange (T52, 2026-10-18)

`normal.go` also holds `ProcessVisual(cmd, doc, at, visual VisualContext,
bracket) []Action` — Visual mode's own Processor, not a branch inside
`ProcessNormal`, since the same keyengine verb identity means something
different depending which one is active (`v` from Normal means "enter";
`v` from Visual's own table, read through `ProcessVisual`, means "exit if
already char-visual, else switch kind"). Motions reuse `resolveMove`
unchanged. `d`/`x`/`y`/`c` need a genuinely new (small) resolver,
`ResolveVisualRange(doc, visual, cursor) (offset.TextRange, bool)` — not
`resolveRange`, since there's no Modifier/Noun left to read once a
selection already exists as live state (`VisualContext{Anchor, Kind}`,
same read-only snapshot pattern as `PasteClipboard`/`BracketContext`) —
but it converges into the exact same `DeleteAction`/`YankAction`
constructors `resolveRange`-driven verbs already use. Selection is
inclusive of wherever the cursor lands (confirmed against real vim, not
assumed symmetric with Normal's exclusive bare-motion sweeps).

`o` (swap anchor/cursor) needed no new action type: it's just
`MoveAction{Target: visual.Anchor}` plus
`SetVisualAnchorAction{Offset: at.ByteOffset}` — the same two primitives
entering Visual already introduced. See CLAUDE.md's "Minimal primitives,
maximal composition" for the design reasoning and
[executor](executor.md)'s entry for the new `ActionKind`s this needed
(`ActionEnterVisual`/`ActionExitVisual`/`ActionSetVisualAnchor`/
`ActionSetVisualKind`).

`ProcessVisual`'s `Inside`/`Around` case (T53, 2026-10-18) is `iw`/`ip`/
`aw`: calls `resolveRange` directly — the same resolver, count-chaining
included, `diw`/`daw`/`dip`/`3diw` already use — but reads `cmd.Verb`
instead of `cmd.Modifier`, since [keyengine](keyengine.md)'s state machine
promotes `i`/`a` into the Verb slot when they start a Visual-mode sequence
with no leading operator (see keyengine's own entry on why). The resolved
range becomes a selection, not an immediate action: `SetVisualAnchorAction`
at the range's start, `MoveAction` to its last rune — Visual mode stays
active afterward so a following `d`/`y`/`c` acts on exactly what was just
selected.

## Block-visual mode (T54, 2026-10-18)

`ProcessVisual`'s `EnterVisualBlock` case mirrors `EnterVisualChar`/`Line`'s
own live-state decision (switch kind in place if a different kind is
already active, else exit) — no new pattern needed. `d`/`y`/`c` each branch
on `visual.Kind == types.VisualBlock` before falling through to the
existing char/line path, converging into two new resolvers:

- `ResolveBlockRanges(doc, visual, cursor) []offset.TextRange` — the one
  genuinely new resolution logic this task needed. Anchor and cursor each
  convert to `(row, col)` via `offset.ByteOffsetToPosition`; row range and
  column range are computed independently per axis (unlike char/line's
  single coupled range), then each spanned row converts back to a
  `offset.TextRange` via `offset.PositionToByteOffset` at that row's
  start/one-past-end column. A row shorter than the selection's column
  range clips to its own actual length for free — `PositionToByteOffset`
  already clamps there, no extra logic needed (real vim's own block-select
  behavior).
- `resolveBlockPaste(doc, at, values, before) []Action` — `Paste`/
  `PasteBefore`'s block-mode branch, taken whenever `len(clip.Values) > 1`.
  Inserts each row's text at the same column on N consecutive rows via a
  new `offset.Insertion{Offset, Text}` type (`TextRange`'s sibling for "insert
  this text here," same zero-dependency-leaf-package pattern). The opposite
  edge case from selection: a target row *shorter* than the paste column
  gets padded with spaces up to it (preserving the rectangle's shape,
  matching real vim), computed by comparing the actual landed `(Row, Col)`
  against the intended one. A row that doesn't exist at all (past the
  document's last line) is skipped outright — smaller in scope than real
  vim's full vertical-growth behavior, a deliberate cut.

`BlockDeleteAction`/`BlockYankAction` carry `[]offset.TextRange` instead of
one `offset.TextRange`, reached via a new `hasRanges` capability in
[executor](executor.md) rather than a new `ActionKind` — see executor's own
entry for why block delete/yank don't need one. `BlockInsertAction` *does*
get a new `ActionKind` (`ActionBlockInsert`), since the existing
`ActionInsertText`/`hasInsertion` shape is tied to one cursor-relative caret
and can't express N independent insertion points.

Block `c` (Change) is a deliberate, documented scope cut: rather than real
vim's multi-row replay (type once, applied to every row — cut from this
whole project phase, parked for a later joint design with multi-cursor),
it resolves to a block delete followed by `EnterInsertAction` — one
insertion point, not N. Smaller and honest, not silently wrong.

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
