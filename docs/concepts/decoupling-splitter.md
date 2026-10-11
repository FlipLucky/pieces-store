# Decoupling Rule / Conveyor-Belt Splitter

Every component should have no direct knowledge of any other — either a
self-contained black box, or mediated entirely through an interface. This
is a **guiding principle for new/touched code**, not a mandate to retrofit
every existing violation immediately.

## The mental picture

The user's own, from building factory-automation games: a
**conveyor-belt splitter**. A value arrives labeled with its type; the
splitter reads the label and routes it down the matching chute; nothing
upstream ever needs to know how many chutes exist or where they physically
lead. Adding a new chute later doesn't require rewiring the belt or
touching anything already connected to it — the classic open/closed shape
(open to a new case, closed against needing to change any existing caller
to add it).

Two concrete instances at very different scales:

- **Small scale**: [internal/clipboard-register](../packages/clipboard-register.md)'s
  `getRegister(RegisterType) *Register` — a caller supplies a type and a
  value and gets a result, with zero knowledge of which of several struct
  fields actually backs it.
- **Large scale**: [internal/editor](../packages/editor.md) itself,
  routing to `keyengine`/`offset`/`piecetable`/`lspservice` instead of
  struct fields — same splitter, different size.

This also maps onto domain-driven design's handler→service→repo→data-layer
shape (a pipeline branches into a domain, does its work, the result comes
back out — domains attachable/detachable because they don't know about
each other), though achieving it cleanly in a stateful desktop app took
real, deliberate effort compared to how naturally it falls out of a web
backend's request/response shape.

## See also

- [The orchestrator doesn't play an instrument](orchestrator-instrument.md)
  — the sharpest concrete test for when `Editor` itself is violating this.
- `viewmanager.Document` duplicating `editor`'s own `Document`-shaped
  interface (see [internal/viewmanager](../packages/viewmanager.md)) is
  intentional decoupling under this rule, not accidental duplication to
  merge away.
