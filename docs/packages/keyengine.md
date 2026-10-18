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

  T55 (2026-10-18, "fill out Normal mode") added: `0`/`^`/`$`/`{`/`}` as
  both Directs and Nouns, the same dual-registration shape `h`/`l` already
  established (so `d0`/`d^`/`d$`/`d{`/`d}` compose for free — see
  [interpreter](interpreter.md)); `G` as a Direct-only Noun (`BufferEnd`,
  deliberately *not* a Noun — see below); a new `g` verb (`GPrefix`, no
  `allowedKeyActions`) purely to reuse the existing doubled-verb mechanism
  for `gg`, costing zero parser changes; `D`/`C`/`Y`/`X`/`s`/`S` as pure
  Direct aliases of existing Verb+Noun pairs (`Y` is a true alias for `yy`,
  not `y$`, matching real vim); three new standalone verbs for insert-entry
  (`AppendAfterCursor`/`AppendEndOfLine`/`InsertAtFirstNonBlank`, i.e.
  `a`/`A`/`I`) and two more for standalone edits (`ToggleCase`/`JoinLines`,
  i.e. `~`/`J`). `ExecutableCommand` also gained `HasCount bool` (set from
  the parser's own already-tracked `countStarted`) — the one genuinely new
  primitive this pass needed, since bare `G` (last line) and `5G` (line 5)
  both read `Count` as its default of 1 otherwise. `BufferStart`/`BufferEnd`
  are deliberately **not** registered as Nouns (no `dgg`/`dG`): real vim's
  version of that is linewise (whole lines), a structurally different shape
  than every other Noun here, which all resolve through the charwise
  `resolveMotionRange` path — a scope cut, not an oversight.
- `visual_mode.go` — Visual mode's real tables (T52, 2026-10-18), not
  scaffolding anymore. `CreateVisualModeDirects` is flat: every motion/
  `d`/`x`/`y`/`c`/`o`/`v`/`V`/`<Esc>` fires immediately with no following
  noun, since Visual's grammar never composes an operator with a target
  the way Normal's `diw` does. Required one real `modeTables()` fix beyond
  just populating this table: it used to hardcode `nil` for Visual mode's
  direct table (nothing had ever needed it) — now returns the real map.

  `CreateVisualModeVerbs`/`Nouns` (T53, 2026-10-18) back `iw`/`ip`/`aw`:
  `i`/`a` are registered as Visual's own "verbs" — not semantically verbs,
  just because keyengine's state machine always promotes whichever key
  starts a sequence into the Verb slot, with no way to start directly in
  the Modifier slot the way Normal's `diw` does. Same whitelist shape
  Normal's real `Inside`/`Around` modifiers already use (`i`→`w`/`p`,
  `a`→`w`); `w`/`p` land in `CreateVisualModeNouns`, reusing the same
  `Word`/`Paragraph` identities Normal's own nouns use. `w` staying a bare
  Direct too is the same non-collision shape as the directs-vs-nouns
  section below, not a new exception.

  `<C-v>` (T54, 2026-10-18) is a plain `CreateVisualModeDirects` entry too
  (`{Verb: EnterVisualBlock}`), registered in both `normal_mode.go` (enters
  block-visual from Normal) and `visual_mode.go` (switches kind in place,
  or exits, if already in Visual — same live-state decision
  `interpreter.ProcessVisual` already makes for `v`/`V`). No keyengine
  changes beyond the two table entries — block mode's actual rectangle
  logic lives entirely in [interpreter](interpreter.md).

  `0`/`^`/`$`/`{`/`}` (T55, 2026-10-18) extend `visual_mode.go`'s direct
  table too — same `Move`+`Noun` identities Normal mode's own directs
  resolve through, free given the identical `resolveMove` path Visual's
  existing `w`/`b`/`e`/`h`/`l`/`j`/`k` already use. `gg` isn't included
  there — it would need `g` registered as a Visual-mode verb too for the
  doubled-key mechanism below, deferred since nothing asked for it yet.

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

## `CommandContext` is workspace-global, not per-buffer (T57, 2026-10-18)

One `*CommandContext` is shared across every open buffer in a
[Workspace](workspace.md), rather than one per `Editor`. Confirmed safe by
tracing `ProcessCommand` itself, not assumed: `clearCommandContext()` runs
on **every** exit path except "this key legitimately continues the
in-progress sequence" — including the not-whitelisted branch a stray key
hits mid-sequence (e.g. `:` while an operator like `di` is pending, which
also means that keystroke doesn't even enter Command mode yet, matching
real vim's own operator-cancellation behavior) and the resolved-command
path (`finalizeDirect`/`finalize`). A pending verb/modifier/noun sequence
structurally cannot survive an attempt to do anything else, buffer
switches included — there's nothing for sharing one instance to desync.

Needed its own lock first, though: `CommandContext` had no internal
`sync.Mutex` before this, safe only because it was always reached from
inside one `Editor`'s own `e.mu`. Shared across `Editor`s with independent
mutexes, that's the same bug class T16 fixed in
[piecetable](piecetable.md) — fixed the same way, a `mu sync.Mutex` with
`SetEditorMode`/`ProcessCommand` each self-locking independently (not as
one atomic pair — a real, stated boundary: only one buffer is ever live
and receiving keystrokes at a time, even once splits exist, since input
stays serialized through whichever pane has focus; worth revisiting only
if that assumption ever changes). Confirmed as a real race, not by
inspection: `TestConcurrentSetEditorModeAndProcessCommand` was checked to
actually produce a `WARNING: DATA RACE` under `go test -race` with the
lock temporarily reverted, then clean restored.

## Depends on

`internal/types` only.

## Depended on by

[internal/editor](editor.md) (the only consumer — `keyengine` itself
never imports `editor`, `offset`, or `piecetable`), and, through it,
[internal/workspace](workspace.md) (owns the one shared `CommandContext`
instance).

## See also

- [Simple, Not Easy](../concepts/simple-not-easy.md) — the test this
  package is built to satisfy.
- [Action + Offset](../concepts/action-offset.md)
