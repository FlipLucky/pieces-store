# Concepts

A lookup table, not a second design doc. Each entry is short on purpose —
`ATOMIC_TASKS.md` and `BACKLOG.md` link here instead of re-explaining these
ideas inline every time they come up. If an entry here feels thin, that's
intentional: follow the "See:" line for the full reasoning.

### action+offset
Every keybinding reduces to the same shape: a motion/text-object *resolves*
an offset (or range), and a verb *applies* an action to it. The same
resolver is reused whether a noun is reached after an operator (`dw`) or
pressed standalone as a motion (`w`) — this is why adding a new keybinding
should mean adding a table entry + resolver, never touching dispatch logic.
See: `CLAUDE.md`'s Design philosophy ("Action + offset"), `internal/offset`.

### The orchestrator doesn't play an instrument
The concrete test for whether an `Editor` method holds logic it shouldn't.
A conductor cues the players in order and knows when the piece is done —
they don't pick up an instrument themselves. An `Editor` method should call
out to the package that actually knows how to do the work and react to the
result; the moment its own body branches on a capability check or computes
a value inline, that's the conductor reaching for a violin.
`moveCursorToLocked` is the known current offender — see [[decoupling
rule]] below, same underlying principle, this is its sharpest phrasing.
See: `CLAUDE.md`'s Design philosophy, `BACKLOG.md`'s structural-refactor
section (most of section 1 in `ATOMIC_TASKS.md` is this rule applied).

### Decoupling rule / conveyor-belt splitter
Every component should have no direct knowledge of any other — either a
self-contained black box, or mediated entirely through an interface. The
user's own mental picture for this (from building factory-automation
games): a conveyor-belt splitter. A labeled value arrives, a small router
reads the label and sends it down the matching chute; nothing upstream
needs to know how many chutes exist or where they lead. Adding a new case
later means adding a chute, not rewiring the belt or touching any existing
caller. `internal/clipboard-register`'s `getRegister(RegisterType)` is a
small-scale instance of exactly this shape.
See: `CLAUDE.md`'s Design philosophy ("Decoupling rule").

### Simple, not easy
The test for whether the keybinding foundation (`internal/keyengine`) has
actually reached "done": adding a new keybinding/text-object later should
mean adding *data* (a verb/modifier/noun table entry plus a resolver),
never touching the core dispatch logic. If a future feature needs the
execution path edited rather than just registered into the tables, the
foundation hasn't reached "simple" yet.
See: `CLAUDE.md`'s Design philosophy.

### Validator → Processor → Executor
`Editor.HandleKey`'s key-handling shape, replacing the old single
`execute()` switch (built 2026-10-11, Normal mode first as T19, Insert
mode same day as T20). Four roles, each with exactly one job, talking only
through the Orchestrator (`Editor`) — a hub, not a chain: **Validator**
(keypresses + mode in, a `KeySequence` out — `internal/keyengine` for
Normal mode; folded into Processor for Insert mode, since its grammar is
close to a no-op) → **Processor** (`KeySequence` + mode in, one or more
`Action`s out — `internal/interpreter`; pure resolution, no mutation,
no access to `*editor.Editor`) → **Executor** (`Action`s in, performs the
work, reports a result — `internal/executor`; a dispatch table keyed by
`types.ActionKind`). Resembles Mediator (the hub) + Command (`Action` as a
self-contained unit of work) + a lexer/parser/interpreter pipeline more
than any single textbook pattern — see `DESIGN_KEY_PIPELINE.md`'s history
for the full reasoning trail if the short version here doesn't answer a
specific question.

### Decoupled by shape, not by sender
`Action`/`KeySequence` are interfaces, not shared structs — each producer
(`internal/interpreter` today; a future LSP-driven edit or plugin/Lua call
tomorrow) defines its own concrete type satisfying the small method set
Executor needs, with zero import of Executor's package. Free in Go
specifically because interfaces are satisfied structurally (no
`implements` declaration, no import of wherever the interface is defined)
— the one shared vocabulary item is `types.ActionKind`, the dispatch key.
Concretely proven, not just designed-for: Insert mode's cursor motion and
backspace-with-pair-collapse resolve into the *same* `MoveAction`/
`DeleteAction` concrete types Normal mode's `w`/`dw` already produce —
Executor's handlers for those didn't need to change at all when Insert
mode was added.

### Dumb front ends / Editor as spine
`internal/editor.Editor` is the orchestrator — the backbone everything
connects through, owning all state and performing actions from key input.
`internal/piecetable` is the actual reason the project exists; the editor's
job is to connect it to everything else, not to be the interesting part
itself. Front ends (`gui-base`/Gio, `tui-base`/tview) render whatever
state the editor exposes, capture raw keystrokes, and forward them — no
editing logic belongs in a front end. Divergence between the two front
ends is a bug to fix, not an acceptable stylistic difference.
See: `CLAUDE.md`'s Vision & direction and Design philosophy
("Delegation model").
