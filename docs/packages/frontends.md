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
