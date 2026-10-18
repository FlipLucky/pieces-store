# Entry points: cmd/gui, cmd/tui, cmd/api

## cmd/gui, cmd/tui

Working entry points. Each builds an `editor.Editor`
(`editor.NewEditor(...)` or `editor.NewEditorFromFile(path)`), wraps it in
a `workspace.Workspace` (`workspace.New(ed)`, T56, 2026-10-18), and hands
that to the corresponding front end (`guibase.CreateApp(ws).Run()` /
`tuibase.Boot(ws)`) — see [Workspace](workspace.md) and
[Front ends](frontends.md).

## cmd/api

**Not** a consumer-facing app — currently just a demo harness for
exercising [piecetable](piecetable.md) directly. Its long-term role is
undetermined but may become an extensibility/scripting entry point
(a Lua layer was floated, in the spirit of Neovim) — not dead code to
delete outright, just not built out yet.

## Depends on

[internal/editor](editor.md) and [internal/workspace](workspace.md)
(`cmd/gui`/`cmd/tui`), [internal/piecetable](piecetable.md) directly
(`cmd/api`, bypassing `Editor` entirely — intentional, since it's a
core-engine demo harness, not an application).
