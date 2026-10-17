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
  `hasCount`, `hasRune`, `hasBelow`, `hasInsertion`, `hasDirection`,
  `hasRegisterDestination`, `hasLinewise`, `hasForward`,
  `hasNamedRegister`) for pulling a payload back out of an `Action` via
  type-assertion — one per distinct shape actually needed, satisfied
  structurally by whatever concrete type produced the `Action`.
  `hasNamedRegister` (T24, 2026-10-11) threads a `"{letter}` register
  selection into `DeleteRangeWithRegister`/`YankRange`, both widened with
  a `named rune` parameter alongside the existing `RegisterType`.
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
