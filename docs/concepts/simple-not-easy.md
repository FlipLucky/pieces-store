# Simple, Not Easy

The keybinding foundation ([internal/keyengine](../packages/keyengine.md))
is meant to be genuinely hard to get right once, so that everything built
on top of it stays simple.

The concrete test for whether this has actually been achieved: adding a
new keybinding/text-object later should mean adding *data* (a new
verb/modifier/noun table entry plus a resolver function), never touching
the core dispatch logic. If a future feature requires editing the
execution path rather than just registering into the tables, the
foundation hasn't reached "simple" yet.

This has held up in practice: `yy`/`cc` reused the generic
doubled-verb/`LineMotion` mechanism originally built for `dd` with zero
dispatch-logic changes (see
[internal/clipboard-register](../packages/clipboard-register.md)'s
history); `h`/`l` and the arrow keys resolve through the same generic
`Move`-verb pipeline as `w`/`b`/`e`.

This same decoupled black-box boundary (editor ↔ frontends) is intended to
double as the eventual plugin/scripting API surface (e.g. a future Lua
layer) rather than needing a second, separately-designed API — mirroring
how Neovim's own RPC API is the same API used by both external GUIs and
Lua plugins, not a separate surface bolted on next to it.

## See also

- [Action + offset](action-offset.md) — the shape every keybinding reduces
  to, which is what makes "just add a table entry" possible.
- [internal/keyengine](../packages/keyengine.md)
