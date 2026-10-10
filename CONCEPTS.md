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
