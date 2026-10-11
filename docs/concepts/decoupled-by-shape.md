# Decoupled by Shape, Not by Sender

`Action` (and `KeySequence`) are interfaces, not shared structs. Each
producer — [internal/interpreter](../packages/interpreter.md) today, a
future LSP-driven edit or plugin/Lua call tomorrow — defines its own
concrete type satisfying the small method set
[internal/executor](../packages/executor.md) needs, with **zero import**
of Executor's package.

This is free in Go specifically because interfaces are satisfied
structurally: no `implements` declaration, no import of wherever the
interface is defined. The one shared vocabulary item across the whole
pipeline is `types.ActionKind` (in `internal/types`, the dependency-free
kernel every package can safely import) — the dispatch key Executor
switches on. Everything else about an `Action` (its payload, its concrete
Go type) is deliberately *not* shared.

A second layer of the same trick: small, optional-capability interfaces
(`hasOffset`, `hasRange`, `hasCount`, `hasRegisterDestination`,
`hasLinewise`, ...) declared inside `internal/executor` itself, satisfied
structurally by whatever concrete Action type was produced. Executor
type-asserts against these to pull a payload back out, without ever
knowing or caring which package built the value.

## Concretely proven, not just designed-for

Insert mode's cursor motion and backspace-with-pair-collapse resolve into
the *same* `MoveAction`/`DeleteAction` concrete types Normal mode's
`w`/`dw` already produce — Executor's handlers for those didn't need to
change at all when Insert mode was added (see
[Validator → Processor → Executor](validator-processor-executor.md)'s
history). Proven a second time by the clipboard-register feature: `p`/`P`
reuse `MoveAction`/`InsertTextAction` as-is, and the linewise-paste fix
only needed a new `Linewise() bool` capability interface, not a new
Action type.

## See also

- [Validator → Processor → Executor](validator-processor-executor.md)
- [internal/types](../packages/small-packages.md#internaltypes) — where
  `ActionKind` lives.
