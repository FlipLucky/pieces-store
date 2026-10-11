# Dumb Front Ends / Editor as Spine

[internal/editor](../packages/editor.md)'s `Editor` is the orchestrator —
the backbone everything connects through, owning all state (mode, cursor,
etc.) and performing actions from key input.
[internal/piecetable](../packages/piecetable.md) is the actual reason the
project exists; the editor's job is to connect it to everything else, not
to be the interesting part itself.

Front ends ([gui-base](../packages/frontends.md)/Gio,
[tui-base](../packages/frontends.md)/tview) are meant to be **dumb**: they
render whatever text/state the editor exposes, capture raw keystrokes, and
forward them to the editor — no editing logic belongs in a front end.
Divergence between the two front ends is a bug to fix, not an acceptable
stylistic difference (both are meant to reach full parity over time).

## Why this matters beyond just "clean architecture"

This is the same lever the project's whole competitive positioning rests
on: Neovim's biggest adoption wall is being terminal-only. If editing
logic lived in each front end separately, GUI/TUI parity would be a
constant, manual-sync tax. Because it doesn't, both front ends are "the
same editor behind two different rendering surfaces" by construction, not
by discipline — and a future web/WASM build (Gio has real WASM support)
is a plausible third front end on the exact same spine, not a separate
project.

## See also

- [The orchestrator doesn't play an instrument](orchestrator-instrument.md)
  — the test for whether `Editor` itself is staying in its own lane.
- [internal/editor](../packages/editor.md),
  [Front ends](../packages/frontends.md)
