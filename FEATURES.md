# Features

What the editor actually does, where that came from, and where it's going.
This is the product view — for engineering-log-style tracking of specific
known bugs/gaps, see `CLAUDE.md`'s Known Issues and `BACKLOG.md` instead.

## Present — what works today

**Core editing** (via `internal/piecetable`, a piece-table buffer):
- Insert, delete, undo, and redo, all backed by a real piece-table
  implementation rather than a naive string buffer.
- Undo/redo report the precise byte-range delta of each edit, not just "the
  buffer changed" — the piece that made real syntax highlighting possible
  without re-parsing the whole file on every keystroke.
- Open and save text files, with unsaved-changes protection (`:q` refuses
  to quit with unsaved edits; `:q!` discards them deliberately).

**Vim-style modal editing** (via `internal/keyengine` + `internal/offset`),
with full GUI/TUI parity — both front ends route through the identical
engine, not just one of them:
- Modes: Normal, Insert, Command (Visual mode exists as a concept but has
  no keybindings wired to it yet — see Future).
- Cursor motion: `h`/`j`/`k`/`l` and the arrow keys (Normal mode; arrow
  keys also move the cursor in Insert mode), plus `w`/`b`/`e` (word
  forward/backward/end), all with count prefixes (`3l`, `2w`).
- Text objects: `diw` (delete inner word), `daw` (delete around word,
  including trailing whitespace), `dip` (delete inner paragraph).
- Line operations: `dd` (delete line), `o` / `O` (open a new line
  below/above, auto-indented — see below — and enter Insert mode).
- Undo / redo: `u` / `Ctrl-r`.
- Replace: `r` swaps the character under the cursor for the next keystroke.
- Command mode: `:w [file]`, `:e file`, `:q`/`:q!`, `:wq`, with
  vim-wildmenu-style path completion — `<Tab>` starts/confirms a candidate
  (descending into directories), `<C-n>`/`<C-p>` browse the list.
- Auto-closing brackets and quotes in Insert mode: typing `{`/`(`/`[`
  inserts the matching close too, cursor landing between them; typing the
  closing bracket yourself skips over the auto-inserted one instead of
  doubling up; backspace collapses an empty pair in one step. `"`/`'`/`` ` ``
  get the same treatment, but only pair when neither neighboring character
  is a word character — so typing a contraction like `don't` doesn't
  trigger an unwanted auto-pair on the apostrophe.

**Real syntax highlighting** (`internal/syntax`, added 2026-09-19) for a
curated 15-language set: Go, TypeScript, TSX, JavaScript, HTML, CSS, SCSS,
PHP, Dart, C, C++, YAML, JSON, Dockerfile, Markdown. Built on a pure-Go
tree-sitter runtime (no C toolchain needed to build or run this editor, on
any platform — including a working WASM build, chosen specifically to
keep the door open for a future web target). Updates incrementally as you
type rather than re-parsing the whole file.

**Real auto-indent** (also `internal/syntax`, same day): opening a new line
(`o`/`O`) or pressing `<Enter>` mid-line indents to match the actual
syntax-tree nesting at that point — not a brace-counting heuristic, which
would misfire on a `{`/`}` sitting inside a string or comment.

**Delivery**: a terminal UI (`tview`/`tcell`) and a GUI (`Gio`), sharing the
same `Editor` core, both windowed/viewport-bounded rather than redrawing
the whole document on every keystroke, both showing the same syntax
highlighting, both with a block cursor in Normal mode and a pipe/bar
cursor in Insert mode.

**A real language server installer/manager** (`:LspInstall`/
`:LspUninstall`/`:LspStatus`, added 2026-09-20), the first half of real LSP
support: installs an actual language server for any of the curated
languages, sourced from the same live `mason-registry` data Neovim's Mason
plugin uses — Go (`gopls`), TypeScript/JavaScript/TSX
(`typescript-language-server`), HTML/CSS/SCSS/JSON
(`vscode-langservers-extracted`), PHP (`intelephense`), YAML
(`yaml-language-server`), C/C++ (`clangd`), Markdown (`marksman`),
Dockerfile (`docker-language-server`), and Dart (detected on `PATH` — it
ships with its own SDK). Installs run in the background — the editor stays
responsive while a server downloads — with progress and results shown in
the status bar. A missing runtime dependency (Node, for the npm-sourced
servers) is checked and reported clearly only when you actually try to
install something that needs it, not required upfront for languages that
don't.

**Real diagnostics, hover, autocomplete, and formatting** — a working LSP
feature set, not just the installer: opening a file with an installed
server for its language automatically starts that server in the
background — no separate step — and:
- Real errors/warnings show up as colored spans right in the text, the
  same rendering pipeline syntax highlighting already uses.
- `K` in Normal mode shows real hover info (type signatures, docs) in a
  popup anchored near the cursor.
- Typing a trigger character the server recognizes (e.g. `.` for Go) pops
  up real completion candidates in Insert mode — browse with
  `<C-n>`/`<C-p>`, confirm with `<Enter>`.
- `:Format` reformats the buffer using the server's own formatter.

Powered by a hand-rolled LSP client (`internal/lspclient`) that speaks
real JSON-RPC to the server over its own stdin/stdout — every one of the
features above verified against an actual running `gopls`, not simulated.

## Present — known gaps in what's above

- **Visual mode isn't reachable** — no key enters it yet.
- **No registers/yank-paste, macros, or search** — none of these exist at
  all yet, a distinct gap from anything above.
- **The GUI's hover/completion popup position is an approximation, not
  pixel-exact** — Gio's text rendering here doesn't track real pixel
  coordinates for the cursor, so the popup's position is estimated from
  font size, not measured. The terminal version doesn't have this
  limitation (terminal cells are exact by nature).
- **Go-to-definition** is a further, separate plan again, since it also
  needs multi-buffer/workspace support that doesn't exist yet.
- **Find/replace isn't built yet** — planned via nvim's `:s///`
  command-style, reusing existing command-mode machinery, not a new UI.

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
- The GUI was rebuilt to route through that same engine instead of its own
  hand-rolled keybindings, closing the GUI/TUI parity gap — plus three real
  Gio input-routing bugs found and fixed along the way (a tag needing an
  explicit focus filter to ever receive typed text, bare Tab being a
  reserved system key, and a wildcard key filter silently rejecting every
  Ctrl-combo).
- `h`/`j`/`k`/`l` and the arrow keys turned out to have been missed
  entirely when the engine was rebuilt — found live, by actually using the
  editor day to day, not by inspection.
- Real syntax highlighting and tree-based auto-indent landed the same day,
  after a deliberate architecture review (undo/redo edit-deltas → a real
  `ChangeChan` payload → a styled-rendering contract → language detection)
  and a from-scratch prototype to validate the tree-sitter library choice
  before committing to it.
- Bracket and quote auto-pairing landed after that, plus a real bug found
  live in daily use (`h`/`j`/`k`/`l` had no keybinding table entries at
  all) and fixed the same day.
- The LSP installer/manager (`:LspInstall` and friends) is the first piece
  of actual LSP work, split into a deliberate 3-plan sequence (installer →
  client+features → go-to-definition/multi-buffer) rather than one large
  undertaking. Its design was grounded in real, empirically-verified data —
  a live fetch of mason-registry's actual published schema, not an assumed
  one — the same "verify before committing" discipline the tree-sitter
  library choice used.

## Future — staged, not all planned in equal detail

Governing rule throughout: keep later stages in mind architecturally so
nothing forecloses them, but don't build them until reached. See
`CLAUDE.md`'s Roadmap section for the full detail behind each stage.

1. **Stage 1 (MVP) — reached.** GUI/TUI parity is real, both front ends
   route through the same engine. Still open from the original MVP scope:
   smooth editing on very large (500MB-class) files hasn't been
   specifically stress-tested, and find/replace isn't built yet.
2. **Stage 2 — well underway, ahead of the original plan's scope.** The
   original plan was a single markdown/HTML file with treesitter as a
   narrow proof case; what actually landed is real tree-sitter highlighting
   and auto-indent across 15 languages at once, since the foundation work
   (edit-deltas, the styled-rendering contract) turned out to generalize
   cleanly. LSP — the other half of stage 2 — **is essentially done**: a
   real installer/manager, a real client, and diagnostics/hover/
   autocomplete/formatting all wired end-to-end into live rendering in
   both front ends, verified against a real `gopls`. What's left within
   stage 2 itself is polish (the GUI popup's position is a font-size
   approximation, not pixel-exact) rather than missing functionality. A
   live preview pane for markdown/HTML specifically is still unbuilt.
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
