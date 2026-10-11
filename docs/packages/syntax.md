# internal/syntax

Real syntax highlighting. Wraps `github.com/odvcencio/gotreesitter` — a
pure-Go tree-sitter runtime (no cgo anywhere, works on every platform
including WASM), chosen deliberately over the canonical C library
specifically to avoid this project's cgo/cross-compile/WASM tension (see
[Dumb Front Ends / Editor as Spine](../concepts/dumb-front-ends.md)'s "why
this matters" section — web/WASM is a real future front end).

## `Highlighter`

Keeps one incrementally-updated parse tree per buffer:

- `Reparse` — a full parse (new buffer, or the table replaced wholesale).
- `Update(edit, source)` — an incremental one, fed the same
  `piecetable.Edit` shape `Editor.ChangeChan` already carries.
- `Spans(start, end)` — translates the tree into
  `viewmanager.StyledSpan`s via a small capture-name→`Style` prefix
  mapping. **O(log n + k)**, not O(n) — see the performance note below.
- `IndentAt(offset)` — real tree-based auto-indent (counts nested
  `indentContainerTypes` ancestors at a position, a per-language table of
  which node type means "one more indent level"), deliberately not
  brace-counting the raw text (which misfires on a `{`/`}` sitting inside
  a string or comment).

`grammarName` is the one place mapping `types.Language` to the library's
grammar registry names — adding a language is adding a case there.
Build-tag-curated (`grammar_subset_<lang>` — see `Taskfile.yaml`'s
`GRAMMAR_TAGS`), roughly halving binary size versus the library's full
~200-language fleet.

## A real performance bug, fixed

`Spans` originally did a linear scan over *every* highlight range in the
whole document on every call, not just the visible window — a synthetic
2000-function Go file produced 16,000+ ranges, and `Spans` is called once
per render frame, including pure cursor movement. Fixed with a
`maxEndSoFar` prefix-max array enabling real binary search even for a long
range (e.g. a block comment) starting well before the query window.
Measured, not assumed: 2.7x faster at 500 functions, 29x at 8000 — the gap
widens with file size exactly as the complexity analysis predicts.
`viewmanager.DiagnosticSpansForRange` has the same *class* of issue,
deliberately not fixed (diagnostic counts are orders of magnitude smaller
than syntax token counts in practice).

## Depends on

`github.com/odvcencio/gotreesitter`, [viewmanager](viewmanager.md) (for
`StyledSpan`), `internal/types` (for `Language`/`Style`).

## Depended on by

[internal/editor](editor.md) — wired at `moveCursorToLocked` (every
incremental edit) and `setupHighlighterLocked` (construction, or whenever
the table is replaced wholesale).
