# Action + Offset

Every keybinding in this editor reduces to the same shape: a
motion/text-object *resolves* an offset (or range), and a verb *applies*
an action to it.

The same resolver is reused whether a noun is reached after an operator
(`dw`) or pressed standalone as a motion (`w`) — this is why adding a new
keybinding should mean adding a table entry + resolver, never touching
dispatch logic. See [Simple, Not Easy](simple-not-easy.md) for the
concrete test this enables.

The piece table itself only ever knows an offset (`start, length`) and an
action (`Insert`/`Delete`) applied to it — see
[internal/piecetable](../packages/piecetable.md). The keybinding
architecture (see [internal/keyengine](../packages/keyengine.md) and
[internal/offset](../packages/offset.md)) is designed to reduce every
vim-style verb/modifier/noun combination — and eventually the rendering
viewport — to that same shape.

## See also

- [internal/offset](../packages/offset.md) — where motions/text-objects
  actually resolve an offset or range.
- [internal/keyengine](../packages/keyengine.md) — where a keystroke
  sequence resolves into a verb+modifier+noun in the first place.
- `CLAUDE.md`'s Design philosophy section, "Action + offset" — the
  original full writeup.
