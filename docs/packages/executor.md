# internal/executor

**Executor** (see
[Validator → Processor → Executor](../concepts/validator-processor-executor.md)).
Its only job is dispatch: given an already-resolved `Action`, decide which
of `Editor`'s capabilities to invoke. It never resolves anything itself
and never imports `internal/interpreter` — an `Action`'s producer is
irrelevant to it, only its shape is.

## Shape

- `Action interface{ Kind() types.ActionKind }` — deliberately not shared
  with `interpreter.Action` (which has the identical method set). See
  [Decoupled by shape, not by sender](../concepts/decoupled-by-shape.md).
- `Editor interface{...}` — the narrow slice of `*editor.Editor`'s
  capabilities `Dispatch` needs, satisfied structurally by an internal
  adapter (`editor.lockedEditorView`) without this package ever importing
  `editor`. Every method on it is already real, correct `Editor` logic;
  `Dispatch`'s job is choosing which one an `Action` maps to, never
  reimplementing what they do.
- Small, optional-capability interfaces (`hasOffset`, `hasRange`,
  `hasRanges`, `hasCount`, `hasRune`, `hasBelow`, `hasInsertion`,
  `hasInsertions`, `hasDirection`, `hasRegisterDestination`, `hasLinewise`,
  `hasForward`, `hasNamedRegister`, `hasVisualAnchor`, `hasVisualKind`) for
  pulling a payload back out of an `Action` via type-assertion — one per
  distinct shape actually needed, satisfied structurally by whatever
  concrete type produced the `Action`. `hasNamedRegister` (T24, 2026-10-11)
  threads a `"{letter}` register selection into `DeleteRangeWithRegister`/
  `YankRange`, both widened with a `named rune` parameter alongside the
  existing `RegisterType`. `hasVisualAnchor`/`hasVisualKind` (T52,
  2026-10-18) back `ActionSetVisualAnchor`/`ActionSetVisualKind` — see
  [interpreter](interpreter.md)'s `ProcessVisual` entry for why these are
  two separate, independently-dispatchable actions instead of payload on
  one combined `ActionEnterVisual`.

  `hasRanges`/`hasInsertions` (T54, 2026-10-18) are `hasRange`/`hasInsertion`'s
  block-visual siblings — N independent `offset.TextRange`s or
  `offset.Insertion`s instead of one. `ActionDelete`/`ActionYank`'s own
  cases check `hasRanges` *before* `hasRange`: block's
  `BlockDeleteAction`/`BlockYankAction` (which only satisfy `hasRanges`)
  and char/linewise's `DeleteAction`/`YankAction` (which only satisfy
  `hasRange`) each reach the right branch structurally, with no
  `ActionKind`-level distinction needed — a visual selection is just an
  offset, N of them for block mode, same `ActionKind` either way.
  `ActionBlockInsert` is the one genuinely new `ActionKind` this task
  needed: `hasInsertion`'s one cursor-relative caret can't express N
  independent paste locations the way block-paste needs, so it gets its
  own `Kind()` and its own `Editor.InsertTextAtOffsets` call instead of
  reusing `ActionInsertText`.

`ActionYank`'s case used to hardcode `clipboardregister.Yank` for every
yank rather than reading a register destination off the `Action` the way
`ActionDelete`'s case already did via `hasRegisterDestination` — a real,
pre-existing inconsistency, fixed alongside T52 (not just patched for
Visual mode) since Visual's own `y` needed to land in
`clipboardregister.Visual` instead. Defaults to `clipboardregister.Yank`
when `hasRegisterDestination` isn't satisfied, so nothing that doesn't
implement it changes behavior.
- `Dispatch(a Action, ed Editor)` — a switch on `a.Kind()`. A malformed
  `Action` (missing the payload interface its `Kind` implies, or a
  producer reporting `ok=false`) is a no-op, not a panic — the same
  "this executor doesn't know how, so it does nothing" contract
  `resolveMotion`/`resolveRange` already established in
  [interpreter](interpreter.md).

## Depends on

[offset](offset.md) (for `TextRange`),
[clipboard-register](clipboard-register.md) (for `RegisterType`),
`internal/types`.

## Depended on by

[internal/editor](editor.md) — the only consumer. `editor`'s
`toExecutorActions` bridges `[]interpreter.Action` to `[]executor.Action`
via an explicit loop (Go doesn't allow bulk slice-type conversion between
two differently-named interface types even when structurally identical —
`editor` is the one place that legitimately imports both packages, so the
bridge lives there).

## See also

- [Validator → Processor → Executor](../concepts/validator-processor-executor.md)
- [Decoupled by shape, not by sender](../concepts/decoupled-by-shape.md)
