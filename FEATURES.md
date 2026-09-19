# Features

What the editor actually does, where that came from, and where it's going.
This is the product view — for engineering-log-style tracking of specific
known bugs/gaps, see `CLAUDE.md`'s Known Issues and `BACKLOG.md` instead.

## Present — what works today

**Core editing** (via `internal/piecetable`, a piece-table buffer):
- Insert, delete, undo, and redo, all backed by a real piece-table
  implementation rather than a naive string buffer.
- Open and save plain text files.

**Vim-style modal editing** (via `internal/keyengine` + `internal/offset`),
reachable through the TUI today:
- Modes: Normal, Insert, Command (Visual mode exists as a concept but has
  no keybindings wired to it yet — see Future).
- Motions: `w` / `b` / `e` (word forward/backward/end).
- Text objects: `diw` (delete inner word), `daw` (delete around word,
  including trailing whitespace), `dip` (delete inner paragraph).
- Line operations: `dd` (delete line), `o` / `O` (open a new line
  below/above and enter Insert mode).
- Undo / redo: `u` / `Ctrl-r`.
- Replace: `r` swaps the character under the cursor for the next keystroke.
- Command mode: `:w [file]`, `:e file`, `:q`, `:wq` — plus `:q!` to force-quit.
- Count prefixes: e.g. `2w` moves two words forward.
- Unsaved-changes protection: `:q` refuses to quit with unsaved edits;
  `:q!` discards them deliberately.

**Delivery**: a terminal UI (`tview`/`tcell`) and a GUI (`Gio`), sharing the
same `Editor` core.

## Present — known gaps in what's above

- The **GUI doesn't route through the key engine yet** — it still
  hand-rolls a small, separate set of keybindings, and as a result can't
  save or quit. The TUI is the only front end currently exercising the
  full modal-editing engine. Closing this gap is the immediate next step.
- **Visual mode isn't reachable** — no key enters it yet.
- **Large files aren't smooth yet** — both front ends currently redraw by
  copying and re-splitting the entire buffer on every keystroke, which
  won't hold up on very large files until viewport-scoped rendering lands.

## Past — how it got here

- Started as a piece-table experiment (`internal/piecestore`), built as a
  small standalone package before the rest of the editor existed.
- Grew into a full editor shape: the piece table was hardened and renamed
  to `internal/piecetable`, and cursor/position math was extracted into its
  own layer (now `internal/viewmanager` + `internal/offset`).
- The first keybinding engine (`internal/editor/keymap`) was a crude,
  from-scratch first pass — vim-style verb/modifier/noun sequences, but
  every delete-family binding (`x`/`dd`/`dw`/`diw`) collapsed onto the same
  non-directional "backspace" primitive, and Visual mode was entirely
  unreachable.
- That engine was fully replaced (not repaired) by `internal/keyengine`,
  built on an explicit "action + offset" model: every keybinding reduces to
  a motion/text-object *resolving* an offset or range, and a verb *applying*
  an action to it — the same resolver is reused whether a noun is reached
  after an operator (`dw`) or pressed standalone as a motion (`w`).

## Future — staged, not all planned in equal detail

Governing rule throughout: keep later stages in mind architecturally so
nothing forecloses them, but don't build them until reached. See
`CLAUDE.md`'s Roadmap section for the full detail behind each stage.

1. **Stage 1 (MVP)** — the GUI routed through the same key engine as the
   TUI (true GUI/TUI parity), smooth editing on very large (500MB-class)
   files, and find/replace via nvim's `:s///` command-style rather than a
   new modal UI.
2. **Stage 2** — a single markdown or HTML file with a live preview: the
   deliberately low-cost proof case for treesitter + LSP integration, and
   for extending vim motions into syntax-tree-aware text objects (e.g.
   "change inside tag").
3. **Stage 3+** — multi-file/workspace support (multiple buffers, splits),
   workspace-wide search/replace (via an external ripgrep-style tool, not
   by forcing search into the single-buffer model).
4. **Full IDE** — broad language support via treesitter's language presets
   plus LSP, once the foundation above is solid. Debugging support is a
   distinct, later concern on top of that.

**Explicitly out of scope for now**: a plugin/scripting API (e.g. Lua) —
the architecture is deliberately built so this becomes possible later
(every capability attaches to the editor the same decoupled way a frontend
does) without needing a second, separately-designed API — but it isn't
being built yet.
