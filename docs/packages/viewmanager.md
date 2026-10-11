# internal/viewmanager

Screen/viewport mapping, nothing else — slimmed deliberately (`Cursor`
lives in [editor](editor.md) instead, since it's editor state not view
state; position math and motion resolvers live in [offset](offset.md)).

## Layout

- `virtual_grid.go` — `VirtualGrid`/`GetScreenPosition` (per-position
  mapping — currently a no-op passthrough; scroll offsets, line wrapping,
  tab expansion, and a line-number gutter aren't implemented yet).
- `viewport.go` — `ViewportSlice(doc, topRow, visibleRows, margin)`: a
  windowed read, just the lines worth rendering, recomputed fresh on every
  call (deliberately uncached — `piecetable.GetRange` on a viewport-sized
  range is already cheap regardless of document size). `Slice` also
  carries `LineOffsets`/`EndOffset` so `StyledSpan`s can be placed against
  specific runes.
- `style.go` — the styled-rendering contract: `StyledSpan{offset.TextRange,
  Style}` (reusing the same range vocabulary as motions/delete ranges, not
  a new one), `StyleAt`/`SpansForRange` as the pure lookup/filter
  functions, and `types.Style` (keyword, string, diagnostic-error, etc. —
  colors are each frontend's own concern).
- `diagnostic.go` — `DiagnosticSpan{TextRange, Style, Message}`,
  deliberately **not** folded into `StyledSpan` (which stays general-
  purpose syntax vocabulary with no room for a message string).

`viewmanager.Document` duplicates `editor`'s own `Document`-shaped
interface rather than sharing one type — intentional decoupling, see
[Decoupling rule / conveyor-belt splitter](../concepts/decoupling-splitter.md).

## Depends on

[offset](offset.md) (shares `TextRange`), `internal/types` (for `Style`).

## Depended on by

[internal/editor](editor.md), via `Editor.Viewport`/`Editor.StyleSpans`/
`Editor.DiagnosticSpans` — frontends never call this package directly,
same decoupling reasoning as everywhere else.

## See also

- [internal/syntax](syntax.md) — the first and only current real producer
  of `StyledSpan`/`DiagnosticSpan`.
