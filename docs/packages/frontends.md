# Front ends: internal/gui-base, internal/tui-base

Thin, single-file (`base.go`) front ends, one per toolkit — Gio (GUI) and
tview/tcell (TUI). Both hold an `*editor.Editor` and render a Tokyo
Night–themed view + status bar. Both are meant to be **dumb** — see
[Dumb Front Ends / Editor as Spine](../concepts/dumb-front-ends.md); no
editing logic belongs here, only rendering + keystroke forwarding.

Both route every keystroke through `ed.HandleKey` (`gui-base` translates
Gio's `key.EditEvent`/`key.Event` into the same key-string vocabulary
`tui-base` gets natively from tcell). Both render via `Editor.Viewport`
(windowed, scroll-follows-cursor) and color runes per `Editor.StyleSpans`/
`Editor.DiagnosticSpans`, each owning its own `Style`→color mapping.

Both also highlight live `:s///`/`:%s///` matches while the command is
still being typed (`Editor.SubstitutePreviewRanges` — nvim's own
"inccommand" preview, see [exmode](small-packages.md)'s T29 entry), with a
shared highlight color (`#515c7e`, Tokyo Night's visual-selection shade)
but necessarily different mechanics: `gui-base` draws every glyph itself,
so a match is just another full-cell background fill layered underneath
the glyph (`withSubstitutePreviewHighlight`), same non-destructive
layering as its own diagnostic underline. `tui-base` had to grow real new
capability for this — its tag-building `openTag` previously always
hardcoded the background to the editor's own bg color; it now takes an
independent background per rune, so a match keeps its syntax-color
foreground while gaining a highlighted background, with no tradeoff
required (unlike the diagnostic-severity-color limitation below, fg and bg
are genuinely independent tag slots).

Both also render the line-number gutter (`Editor.LineNumberSettings()` +
[viewmanager](viewmanager.md)'s `GutterWidth`/`GutterText`, T28,
2026-10-17) independently — no shared rendering code between the two
frontends, so this is real duplicated work, not a one-package change:
`tui-base` prepends each line's gutter text as a muted-color tag inside
`renderContent`'s per-line loop; `gui-base` prepends a gutter
`material.Label` via a Horizontal `layout.Flex` wrapper in `renderLine`
(falling back to its old single-widget fast path when the gutter and
diagnostic virtual-text are both absent). The gutter shifts where column 0
of the cursor's own text starts on screen, so both frontends' cursor/popup
column math had to account for it: `tui-base`'s `cursorScreenCol` (feeding
the native-cursor positioning above) gained `+ gutterWidth` (a character
count); `gui-base`'s `renderHoverPopup` `x` now comes from the gutter's own
real measured pixel width baked into `cursorPx.X` (see T50 below), not a
`gutterWidth * charWidth` estimate.

`gui-base`'s hover/completion popup position is **pixel-exact as of T50**
(2026-10-18), not estimated: `cursorPixel{X, Y, Height, Found}` is threaded
through `renderBody`→`renderLine`→`renderStyledRunes`, each level
accumulating the *real* `Dimensions` Gio's own layout already returned
(every preceding line's actual height, every preceding rune's actual
width) rather than guessing from `charWidthPx`/`lineHeightPx` — Gio has no
API to query a drawn element's position after the fact (push-only, nothing
to introspect), so "real position" here means tracking it as cells are
genuinely laid out, not reading an op's transform back. Those two
font-metric constants still exist as a documented fallback for the one
real gap this leaves: a cursor sitting past the window's right edge, where
`layout.List` never calls back for that far-off-screen rune at all (no
horizontal scrolling exists yet) — also the one case where the cursor
itself isn't visible anyway. `tui-base` needed no equivalent change: its
`editorView.GetInnerRect()` + cursor-row/col math was already exact real
terminal-cell coordinates, never an estimate.

## Gio-specific gotchas (gui-base)

Three real debugging sessions' worth of findings, worth knowing before
touching `gui-base`'s key handling again:

1. A tag only receives `key.EditEvent` (actual typed text) once something
   registers a `key.FocusFilter{Target: tag}` for it — without that, focus
   is silently stripped every frame and text input never arrives.
2. A bare `Tab` keypress is a reserved system key for Gio's own
   widget-to-widget focus cycling, withheld from a wildcard
   `key.Filter{}` — must be claimed by name (`key.Filter{Name:
   key.NameTab}`).
3. A wildcard `key.Filter{}` only matches events with **zero** modifiers —
   every Ctrl-combo was silently dropped until the filter declared
   `Optional: key.ModCtrl`.

General lesson: Gio's key filters are opt-in and precise by design —
nothing is delivered "by default."

## tview-specific notes (tui-base)

The cursor itself is drawn differently from `gui-base`: `gui-base` draws
every glyph by hand, so its block/pipe cursor is just another glyph.
`tui-base` renders text through tview's tag-string API, which has no
notion of cursor shape at all — through 2026-10-16 it simulated a cursor
by inverting one character's colors in `styledLineText`, the same solid
block regardless of mode. As of 2026-10-17 it drives the terminal's own
real cursor instead: `tcell.Screen.SetCursorStyle`/`ShowCursor`
(confirmed wired to the real terminal backend in tcell v2.8.1, not just
declared on an interface), called every frame from a
`tview.Application.SetAfterDrawFunc` hook, using the exact same
`editorView.GetInnerRect()` + cursor-row/col math `renderHoverPopup`
already used for pixel-exact popup placement. Genuinely changes shape
between a block (Normal/Visual/Command) and a bar (Insert) — something
the old text-simulation approach structurally couldn't express.

Diagnostic underlines can't be colored independently of the text's own
color through tview's tag-string convenience layer (`tcell.Style.Underline`
itself *does* support a color parameter, just not reachable through tags)
— a real, checked limitation, not an oversight; resolved by accepting it
deliberately (a diagnosed rune's color becomes its severity color instead).
tview tags are also sticky across a styled string unless explicitly reset
— every emitted tag states the underline attribute explicitly, never
relying on omission to mean "unchanged."

## Depends on

[internal/editor](editor.md) — the only package either one imports from
this codebase (plus their respective UI toolkits).

## Depended on by

`cmd/gui`/`cmd/tui` — see [Entry points](entry-points.md).
