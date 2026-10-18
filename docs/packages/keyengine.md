# internal/keyengine

Normal mode's **Validator** (see
[Validator → Processor → Executor](../concepts/validator-processor-executor.md)).
Parses vim-style keystroke sequences into a resolved
`ExecutableCommand{Count, Verb, Modifier, Noun}`. Zero dependency on
`piecetable`, `offset`, or `Editor` — keystrokes in, a command (or an
`ErrInconsistentKeymap` error) out. A **leaf package** (needs only
`internal/types`) — deliberately, so it can never end up in a dependency
cycle with `editor` or with [offset](offset.md).

## Layout

- `keyengine.go` — the core parser: `CommandContext.ProcessCommand`, the
  verb→modifier→noun state machine, and the generic doubled-verb check
  (a verb's own key pressed again means "linewise" — built for `dd`,
  reused with zero changes for `yy`/`cc`).
- `normal_mode.go` — the real, populated tables. A fourth table beyond
  verb/modifier/noun, `availableDirects`, covers keys that execute
  immediately with no further input (`w`/`b`/`e`, `h`/`j`/`k`/`l` +
  arrows, `u`/`<C-r>`/`r`, `:`/`i`/`o`/`O`, `p`/`P`, `x`, `%` — T44,
  2026-10-18, the `BracketMatch` noun). `b`/`e`/`h`/`l`
  are also registered as **nouns** (2026-10-11, T22) — reachable bare
  after a verb with no modifier (`dw`/`db`/`de`/`dh`/`dl`, a "short
  delete," see [interpreter](interpreter.md)) — alongside their existing
  Move-only direct entries. `%` is bare-only, deliberately not composable
  with an operator yet (`d%` isn't wired) — same documented scope limit
  `f`/`F` already carry.
- `visual_mode.go` — placeholder scaffolding only; no Visual keybinding is
  implemented yet.

## Named register prefix (`"{letter}`, T24, 2026-10-11)

A one-keystroke literal capture — same shape as Replace/FindChar's second
keystroke, just resolved entirely inside the parser rather than needing
any buffer state. `cc.registerPending`/`cc.register` sit ahead of digit
accumulation in `ProcessCommand`'s priority order, but the quote itself is
only recognized when `cc.verb.name == None` — so both `"a3yw` and `3"ayw`
parse to the same command, matching real vim. Resolves into
`ExecutableCommand.Register rune` (0 means none selected).

## The "directs vs. nouns" non-collision

`p`/`P` are registered as **directs** (`Paste`/`PasteBefore`, fire with no
verb pending), while `p` is *also* registered as a **noun** (`Paragraph`,
only reachable after `di`/`da`). These don't actually collide:
`ProcessCommand`'s state machine checks the `directs` map only when no
verb is pending, and checks the `nouns` map only inside a verb/modifier
continuation — mutually exclusive parser states. Confirmed by tracing the
actual control flow, not assumed — see
[clipboard-register](clipboard-register.md)'s history for where this came
up.

## Depends on

`internal/types` only.

## Depended on by

[internal/editor](editor.md) (the only consumer — `keyengine` itself
never imports `editor`, `offset`, or `piecetable`).

## See also

- [Simple, Not Easy](../concepts/simple-not-easy.md) — the test this
  package is built to satisfy.
- [Action + Offset](../concepts/action-offset.md)
