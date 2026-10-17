# internal/viewmanager

Screen/viewport mapping, nothing else — slimmed deliberately (`Cursor`
lives in [editor](editor.md) instead, since it's editor state not view
state; position math and motion resolvers live in [offset](offset.md)).

## Layout

- `virtual_grid.go` — `VirtualGrid`/`GetScreenPosition` (per-position
  mapping — currently a no-op passthrough; scroll offsets, line wrapping,
  and tab expansion aren't implemented yet — the line-number gutter moved
  out of this list 2026-10-17, see `gutter.go` below).
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
- `gutter.go` (added 2026-10-17, T28) — pure, dependency-free line-number
  gutter math: `GutterWidth(highestVisibleRow)` sizes the column off the
  *currently visible* highest row, not a full-document scan, same
  viewport-bounded-cost reasoning as `ViewportSlice`/`FindPieceAt`'s own
  deferred scan — the tradeoff is the gutter can grow a digit mid-scroll
  on a huge file, accepted deliberately. `GutterText(row, cursorRow,
  showNumber, relative, width)` returns one row's padded cell, modeling
  real vim's `number`/`relativenumber` as two independent options (not a
  three-way toggle) — confirmed a real "hybrid" (both on at once) habit via
  a user dotfiles audit, not just vim-parity for its own sake. Returns ""
  when both options are off, so a caller skips the column entirely. Both
  frontends read the toggle state from `Editor.LineNumberSettings()` (see
  [editor](editor.md)) and call these two functions directly — `Editor`
  itself does no gutter-rendering math of its own, same
  orchestrator-doesn't-play-an-instrument reasoning as everywhere else.

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
