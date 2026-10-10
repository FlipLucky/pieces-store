# Design note: the key-handling pipeline

**Status:** high-level shape settled; exact per-mode Go types not yet
written. This is a working scratchpad, not a finished reference — once
Normal and Insert's interpreters actually get built, promote the settled
parts into `CONCEPTS.md` and delete or shrink this file. Don't treat
anything here as "done," only as "decided in principle."

## Where this came from

`internal/keyengine/insert_mode.go` (an atomic-task item, T2) stalled on a
real design tension, not procrastination: wanting Insert mode to feel
predictable alongside Normal mode's verb/modifier/noun grammar, while
Insert mode genuinely has nothing to compose — it just reacts to whichever
key arrived. Forcing one mode's shape onto another is what actually halted
progress, twice over the project's history (first the keybinding stub,
then almost again at the interface-signature level before catching it).

## The reframe that unstuck it

Predictability across modes was never "same grammar" — it's **resolve
separately from execute, pure, dependency-free, Editor just reacts to the
decision**. Already true in different shapes across this codebase:
`internal/keyengine` (verb/mod/noun → `ExecutableCommand`), `internal/
exmode.Parse` (raw string → `Command`), `internal/autopairs.Decide` (three
runes → a pairing decision). Visual mode is already on record in
`BACKLOG.md` as needing its own different shape too — an operator acting
directly on a selection, no noun. Insert mode diverging from Normal's
grammar isn't breaking the pattern, it's confirming it.

## The pipeline: Validate → Interpret → Apply

Three sequential stages. Each one is the same shape: a label is read, work
is routed to the matching chute, nothing upstream needs to know what's
inside the box — see [decoupling rule / conveyor-belt
splitter](CONCEPTS.md#decoupling-rule--conveyor-belt-splitter). Applied
three times in a row, once per handoff.

1. **Validate** (switch by mode) — is this keypress/sequence legal in the
   current mode's grammar? Returns an opaque validated result, or "not
   yet" (still composing) / "invalid."
   - Normal: already real today — `keyengine.CommandContext.ProcessCommand`.
   - Command: already real today — `exmode.Parse`, once a full line is ready.
   - Insert: close to a no-op — almost every key is "valid"; mostly just
     classifies (a literal rune vs. one of a handful of special keys).
2. **Interpret** (switch by mode) — takes the validated result and resolves
   it into one or more `Target + Action` pairs (below). New as a named
   concept, but already half-exists: `resolveMotion`/`resolveRange` (free
   functions in `internal/editor/dispatch.go`) already do this for Normal
   mode today, and already take `offset.Document` instead of `*Editor` —
   nearly ready to relocate as-is, not a rewrite.
3. **Apply** (switch by `Target`) — takes the resolved pair(s) and routes
   to whichever subsystem actually acts (piece table vs. editor/cursor
   state).

`Editor.HandleKey` becomes three near-identical blocks: call, check for a
greenlight, relay. It holds state (current mode, cursor position) but
never *uses* that state to decide anything — only to route. That's the
concrete, repeatable test for [the orchestrator doesn't play an
instrument](CONCEPTS.md#the-orchestrator-doesnt-play-an-instrument): state
is fine to hold, logic is not fine to perform.

## Target + Action

- **Target: PieceTable** — Action = `Insert`/`Delete` over a `Range` (plus
  content, for Insert). This *is* [action+offset](CONCEPTS.md#actionoffset),
  reused rather than reinvented. Covers `dw`, a literal character insert,
  and — eventually — `:s///`'s replacements.
- **Target: Editor** — Action = `ChangeMode`, `MoveCursor`,
  `StoreSelection` (for Visual mode), and anything similar. Everything
  that mutates Editor/Cursor state but never touches the piece table.
  `MoveCursor`, `ChangeMode`, and selection-storage are *not* three
  separate kinds — they're the same target, different actions on it.
- **Deliberately not modeled by Target+Action at all**: Command mode's
  named subcommands (`:w`, `:q`, `:LspInstall`). Genuinely heterogeneous
  side effects — `executeCommandLocked`'s existing per-command dispatch
  correctly stays exactly as it is (`BACKLOG.md` already settled this, this
  design doesn't reopen it).

Worked examples:

| Key(-sequence)     | Target             | Action |
|--------------------|---------------------|--------|
| `<Esc>`            | Editor              | ChangeMode(Normal) |
| `diw`              | PieceTable          | Delete at resolved range |
| `viw`              | Editor              | ChangeMode(Visual) + StoreSelection(range) |
| `o`                | PieceTable + Editor | Insert(`\n`+indent) + MoveCursor + ChangeMode(Insert) |
| `{` (auto-pair)    | PieceTable + Editor | Insert(`{`) + Insert(`}`) + MoveCursor |

A single keypress can need more than one Target+Action pair at once — the
interpreter's output is a short *list*, not always exactly one pair.

## Where the new code lives

A new package, working name `internal/interpreter`, sitting *above* both
`keyengine` and `offset`, importing both. This resolves a real dependency
question rather than papering over it: `keyengine` and `offset` were
deliberately built as zero-dependency siblings (`CLAUDE.md`), specifically
so neither depends on the other. Making `keyengine` absorb the
range-resolution step itself would give that up. A new consumer package
above both changes neither — `keyengine` and `offset` stay exactly as they
are today.

Likely one file per mode inside it (`interpreter/normal.go`,
`interpreter/insert.go`), mirroring how `keyengine` is already split
internally by mode (`keyengine.go` core + `normal_mode.go` +
`visual_mode.go`).

## Scope boundary (deliberate, not an oversight)

Command mode stays almost entirely out of this redesign for now. `:w`/
`:q`/`:e` already work and don't need to fit this model — they're
SideEffect, not Target+Action, and that's correct, not a gap. `:s///`,
whenever it lands, is the one Command-mode feature that *would* fit
(Target: PieceTable, N replace actions). Nothing about Command mode's
broader surface (ranges, `:g/pattern/cmd`, `:normal`, `:sort`, and the rest
of `VIM_ACTION_CHECKLIST.md`'s Ex-commands section) needs solving now —
explicit choice to avoid a much bigger rabbit hole than this one.

## Open questions — not yet settled

- Exact Go type(s) for Normal mode's and Insert mode's validated-result
  values. Likely `any` + a type-assert inside the matching `Interpret`
  branch — the same idiom already needed for `Executable` itself — but not
  yet written down as real code.
- Insert mode's interpreter needs read access to surrounding runes (for
  auto-pairing decisions) without importing `Editor` directly. Likely a
  small interface the Insert package declares for itself, satisfied by
  whatever `Editor` passes at construction — the same move already made
  for `viewmanager.Document` — but not yet named or written.
- What replaces `internal/keyengine/insert_mode.go`'s current stub.
  Almost certainly: delete it outright. Its placement (inside `keyengine`)
  doesn't fit once Insert mode's actual shape is clear — there's no
  verb/mod/noun grammar for it to validate.
- `CommandBuffer` becoming its own type (already tracked as
  `BACKLOG.md`/`ATOMIC_TASKS.md`'s T5) is effectively a *prerequisite* for
  Command mode's eventual Validate stage, not an unrelated refactor item —
  worth remembering when T5 comes up, so it isn't done twice.

## Status

High-level shape — Validate → Interpret → Apply, Target+Action, mode/
Target-keyed routing throughout — is settled. Per-mode concrete types are
not. This file is the holding place until Normal and Insert's interpreters
actually exist as code; at that point the settled parts belong in
`CONCEPTS.md` instead, and this file should shrink to just the still-open
questions or be deleted.
